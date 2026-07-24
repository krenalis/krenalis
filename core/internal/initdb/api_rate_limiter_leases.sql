-- acquire_api_rate_limit_leases refills authoritative buckets and leases
-- capacity for a batch of subjects. It removes granted capacity before
-- returning, so concurrent application nodes cannot lease it twice.
CREATE OR REPLACE FUNCTION acquire_api_rate_limit_leases(p_requests jsonb)
    RETURNS TABLE (
        subject_kind varchar,
        subject_id varchar,
        granted_units integer,
        capacity_units integer
    )
    LANGUAGE plpgsql
    VOLATILE
AS $$
    -- Names declared by RETURNS TABLE are also PL/pgSQL variables. Prefer table
    -- columns when a name is ambiguous, and assign the output variables
    -- explicitly before RETURN NEXT.
    #variable_conflict use_column
DECLARE
    v_request record;
    v_quota_per_hour integer;
    v_burst_capacity integer;
    v_acquisition_time timestamptz := clock_timestamp();
    v_bucket api_rate_limit_buckets%ROWTYPE;
    v_elapsed_microseconds numeric;
    v_refilled_units bigint;
    v_refill_remainder bigint;
    v_available_units integer;
    v_refill_numerator numeric;
    v_granted_units integer;
BEGIN
    IF p_requests IS NULL OR jsonb_typeof(p_requests) <> 'array' OR jsonb_array_length(p_requests) = 0 THEN
        RAISE EXCEPTION 'API rate-limit lease requests must be a non-empty JSON array';
    END IF;

    IF jsonb_array_length(p_requests) > 64 THEN
        RAISE EXCEPTION 'too many API rate-limit lease requests';
    END IF;

    IF EXISTS (
        SELECT
        FROM jsonb_to_recordset(p_requests) AS r(subject_kind text, subject_id text, requested_units integer)
        WHERE r.subject_kind IS NULL
           OR r.subject_kind NOT IN ('workspace', 'ingestion', 'nonspecific')
           OR r.subject_id IS NULL
           OR r.subject_id !~ '^[1-9A-HJ-NP-Za-km-z]{12}$'
           OR r.requested_units IS NULL
           OR r.requested_units < 1
           OR (r.subject_kind = 'workspace' AND r.requested_units > 100)
           OR (
                r.subject_kind = 'ingestion'
                AND r.requested_units > 20000
            )
           OR (r.subject_kind = 'nonspecific' AND r.requested_units > 100)
    ) THEN
        RAISE EXCEPTION 'invalid API rate-limit lease request';
    END IF;

    IF EXISTS (
        SELECT r.subject_kind, r.subject_id
        FROM jsonb_to_recordset(p_requests) AS r(subject_kind text, subject_id text, requested_units integer)
        GROUP BY r.subject_kind, r.subject_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate API rate-limit lease request';
    END IF;

    -- Process buckets in key order so overlapping batches acquire row locks in
    -- the same order. This reduces the risk of deadlocks between application
    -- nodes, including batches that mix organizations and workspaces.
    FOR v_request IN
        SELECT r.subject_kind, r.subject_id, r.requested_units
        FROM jsonb_to_recordset(p_requests) AS r(subject_kind text, subject_id text, requested_units integer)
        ORDER BY r.subject_kind, r.subject_id
    LOOP
        -- Read the rate-limit configuration from the subject's authoritative
        -- domain table. A concurrent configuration update may become visible only
        -- to a later lease acquisition.
        IF v_request.subject_kind = 'workspace' THEN
            SELECT o.api_workspace_quota_per_hour, o.api_workspace_burst_capacity
            INTO v_quota_per_hour, v_burst_capacity
            FROM workspaces w
            JOIN organizations o ON o.id = w.organization
            WHERE w.id = v_request.subject_id;
        ELSIF v_request.subject_kind = 'ingestion' THEN
            SELECT o.api_ingestion_quota_per_hour, o.api_ingestion_burst_capacity
            INTO v_quota_per_hour, v_burst_capacity
            FROM workspaces w
            JOIN organizations o ON o.id = w.organization
            WHERE w.id = v_request.subject_id;
        ELSE
            SELECT api_nonspecific_quota_per_hour, api_nonspecific_burst_capacity
            INTO v_quota_per_hour, v_burst_capacity
            FROM organizations
            WHERE id = v_request.subject_id;
        END IF;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'API rate-limit subject % does not exist', v_request.subject_id;
        END IF;

        INSERT INTO api_rate_limit_buckets (
            subject_kind,
            subject_id,
            organization,
            workspace,
            available_units,
            capacity_units,
            quota_per_hour,
            last_refill_at,
            refill_remainder
        ) VALUES (
            v_request.subject_kind,
            v_request.subject_id,
            CASE WHEN v_request.subject_kind = 'nonspecific' THEN v_request.subject_id END,
            CASE WHEN v_request.subject_kind IN ('workspace', 'ingestion') THEN v_request.subject_id END,
            v_burst_capacity,
            v_burst_capacity,
            v_quota_per_hour,
            v_acquisition_time,
            0
        )
        ON CONFLICT (subject_kind, subject_id) DO NOTHING;

        SELECT *
        INTO v_bucket
        FROM api_rate_limit_buckets AS b
        WHERE b.subject_kind = v_request.subject_kind
          AND b.subject_id = v_request.subject_id
        FOR UPDATE;

        IF v_bucket.capacity_units <> v_burst_capacity
            OR v_bucket.quota_per_hour <> v_quota_per_hour
        THEN
                -- Preserve available capacity only up to the new burst capacity.
                -- Refill accrued under the previous configuration is discarded
                -- because the exact time at which the new configuration took effect
                -- is unknown.
            v_available_units := LEAST(
                v_bucket.available_units,
                v_burst_capacity
            );
            v_refill_remainder := 0;
        ELSE
                -- Calculate accrued capacity at microsecond precision. The remainder
                -- carries fractional units into the next lease acquisition so they
                -- are not lost through integer rounding.
            v_elapsed_microseconds := GREATEST(
                0,
                EXTRACT(EPOCH FROM v_acquisition_time - v_bucket.last_refill_at) * 1000000
            );
            v_refill_numerator :=
                v_elapsed_microseconds * v_quota_per_hour + v_bucket.refill_remainder;
            v_refilled_units := FLOOR(v_refill_numerator / 3600000000)::bigint;

            v_available_units := LEAST(
                v_burst_capacity,
                v_bucket.available_units
                    + LEAST(v_refilled_units, v_burst_capacity::bigint)::integer
            );

            IF v_available_units = v_burst_capacity THEN
                v_refill_remainder := 0;
            ELSE
                v_refill_remainder := MOD(v_refill_numerator, 3600000000)::bigint;
            END IF;
        END IF;

            -- Remove granted capacity from PostgreSQL before returning it to the
            -- application node. A process crash may therefore lose unused leased
            -- capacity, but it cannot create additional capacity.
        v_granted_units := LEAST(
            v_request.requested_units,
            v_available_units
        );

        UPDATE api_rate_limit_buckets AS b
        SET available_units = v_available_units - v_granted_units,
            capacity_units = v_burst_capacity,
            quota_per_hour = v_quota_per_hour,
            last_refill_at = v_acquisition_time,
            refill_remainder = v_refill_remainder
        WHERE b.subject_kind = v_request.subject_kind
          AND b.subject_id = v_request.subject_id;

        subject_kind := v_request.subject_kind;
        subject_id := v_request.subject_id;
        granted_units := v_granted_units;
        capacity_units := v_burst_capacity;
        RETURN NEXT;
    END LOOP;
END;
$$;

COMMENT ON FUNCTION acquire_api_rate_limit_leases(jsonb) IS
    'Refills authoritative API rate-limit buckets and leases capacity to application nodes.';

-- SQL formatting guidelines for this file:
-- - Use uppercase SQL and PL/pgSQL keywords, including SELECT, FROM, IF, LOOP,
--   BEGIN, END, and RETURN.
-- - Use lowercase snake_case for database objects, columns, variables, and
--   function parameters.
-- - Indent nested SQL and PL/pgSQL blocks with four spaces per level.
-- - Align IF/END IF and LOOP/END LOOP at the same indentation level.
-- - Put each column in long INSERT, SELECT, and RETURNS TABLE lists on its own
--   line, and align the closing parenthesis with the opening construct.
-- - Break long expressions across lines at logical operators or function
--   arguments, using one additional indentation level for continuations.
-- - Keep comments at the indentation level of the block they explain. Comments
--   should document invariants, concurrency decisions, and non-obvious behavior.

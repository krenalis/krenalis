-- acquire_rate_limit_leases refills authoritative buckets and leases
-- capacity for a batch of subjects. It removes granted capacity before
-- returning, so concurrent application nodes cannot lease it twice.
CREATE OR REPLACE FUNCTION acquire_rate_limit_leases(p_requests jsonb)
    RETURNS TABLE (
        subject_kind varchar,
        subject_id varchar,
        granted_units integer,
        capacity_units integer,
        available_units integer,
        rate_per_minute integer,
        refill_remainder integer
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
    v_rate_per_minute integer;
    v_burst_capacity integer;
    v_refill_time timestamptz;
    v_bucket rate_limit_buckets%ROWTYPE;
    v_elapsed_microseconds numeric;
    v_refilled_units bigint;
    v_refill_remainder integer;
    v_available_units integer;
    v_refill_numerator numeric;
    v_granted_units integer;
BEGIN
    IF p_requests IS NULL OR jsonb_typeof(p_requests) <> 'array' OR jsonb_array_length(p_requests) = 0 THEN
        RAISE EXCEPTION 'rate-limit lease requests must be a non-empty JSON array';
    END IF;

    IF jsonb_array_length(p_requests) > 64 THEN
        RAISE EXCEPTION 'too many rate-limit lease requests';
    END IF;

    IF EXISTS (
        SELECT
        FROM jsonb_to_recordset(p_requests) AS r(subject_kind text, subject_id text, requested_units integer)
        WHERE r.subject_kind IS NULL
           OR r.subject_kind NOT IN ('platform', 'organization', 'workspace', 'events')
           OR r.subject_id IS NULL
           OR (r.subject_kind = 'platform' AND r.subject_id <> 'platform')
           OR (r.subject_kind <> 'platform' AND r.subject_id !~ '^[1-9A-HJ-NP-Za-km-z]{12}$')
           OR r.requested_units IS NULL
           OR r.requested_units < 1
           OR (r.subject_kind IN ('platform', 'organization', 'workspace') AND r.requested_units > 100)
           OR (
                r.subject_kind = 'events'
                AND r.requested_units > 20000
            )
    ) THEN
        RAISE EXCEPTION 'invalid rate-limit lease request';
    END IF;

    IF EXISTS (
        SELECT r.subject_kind, r.subject_id
        FROM jsonb_to_recordset(p_requests) AS r(subject_kind text, subject_id text, requested_units integer)
        GROUP BY r.subject_kind, r.subject_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate rate-limit lease request';
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
        IF v_request.subject_kind = 'platform' THEN
            SELECT requests_rate_per_minute, requests_burst_capacity
            INTO v_rate_per_minute, v_burst_capacity
            FROM metadata
            WHERE singleton;
        ELSIF v_request.subject_kind = 'organization' THEN
            SELECT organization_requests_rate_per_minute, organization_requests_burst_capacity
            INTO v_rate_per_minute, v_burst_capacity
            FROM organizations
            WHERE id = v_request.subject_id;
        ELSIF v_request.subject_kind = 'workspace' THEN
            SELECT o.workspace_requests_rate_per_minute, o.workspace_requests_burst_capacity
            INTO v_rate_per_minute, v_burst_capacity
            FROM workspaces w
            JOIN organizations o ON o.id = w.organization
            WHERE w.id = v_request.subject_id;
        ELSE
            SELECT o.workspace_events_rate_per_minute, o.workspace_events_burst_capacity
            INTO v_rate_per_minute, v_burst_capacity
            FROM workspaces w
            JOIN organizations o ON o.id = w.organization
            WHERE w.id = v_request.subject_id;
        END IF;

        IF NOT FOUND THEN
            -- A result with all numeric fields set to zero is the
            -- missing-subject sentinel. Returning one row preserves batch
            -- completeness and lets the caller reject only this subject while
            -- applying valid results for the others.
            subject_kind := v_request.subject_kind;
            subject_id := v_request.subject_id;
            granted_units := 0;
            capacity_units := 0;
            available_units := 0;
            rate_per_minute := 0;
            refill_remainder := 0;
            RETURN NEXT;
            CONTINUE;
        END IF;

        INSERT INTO rate_limit_buckets (
            subject_kind,
            subject_id,
            organization,
            workspace,
            available_units,
            capacity_units,
            rate_per_minute,
            last_refill_at,
            refill_remainder
        ) VALUES (
            v_request.subject_kind,
            v_request.subject_id,
            CASE WHEN v_request.subject_kind = 'organization' THEN v_request.subject_id END,
            CASE WHEN v_request.subject_kind IN ('workspace', 'events') THEN v_request.subject_id END,
            v_burst_capacity,
            v_burst_capacity,
            v_rate_per_minute,
            clock_timestamp(),
            0
        )
        ON CONFLICT (subject_kind, subject_id) DO NOTHING;

        SELECT *
        INTO v_bucket
        FROM rate_limit_buckets AS b
        WHERE b.subject_kind = v_request.subject_kind
          AND b.subject_id = v_request.subject_id
        FOR UPDATE;

        IF NOT FOUND THEN
            -- The subject may have been deleted after its configuration was
            -- read. Return the same missing-subject sentinel without granting
            -- capacity that was not deducted from an authoritative bucket.
            subject_kind := v_request.subject_kind;
            subject_id := v_request.subject_id;
            granted_units := 0;
            capacity_units := 0;
            available_units := 0;
            rate_per_minute := 0;
            refill_remainder := 0;
            RETURN NEXT;
            CONTINUE;
        END IF;

        -- Choose the refill time only after acquiring the row lock. A batch
        -- may have waited while a newer acquisition updated this bucket, so
        -- never move its authoritative refill time backwards.
        v_refill_time := GREATEST(clock_timestamp(), v_bucket.last_refill_at);

        IF v_bucket.capacity_units <> v_burst_capacity
            OR v_bucket.rate_per_minute <> v_rate_per_minute
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
            v_elapsed_microseconds :=
                EXTRACT(EPOCH FROM v_refill_time - v_bucket.last_refill_at) * 1000000;
            v_refill_numerator :=
                v_elapsed_microseconds * v_rate_per_minute + v_bucket.refill_remainder;
            v_refilled_units := FLOOR(v_refill_numerator / 60000000)::bigint;

            v_available_units := LEAST(
                v_burst_capacity,
                v_bucket.available_units
                    + LEAST(v_refilled_units, v_burst_capacity::bigint)::integer
            );

            IF v_available_units = v_burst_capacity THEN
                v_refill_remainder := 0;
            ELSE
                v_refill_remainder := MOD(v_refill_numerator, 60000000)::integer;
            END IF;
        END IF;

        -- Remove granted capacity from PostgreSQL before returning it to the
        -- application node. A process crash may therefore lose unused leased
        -- capacity, but it cannot create additional capacity.
        v_granted_units := LEAST(
            v_request.requested_units,
            v_available_units
        );

        UPDATE rate_limit_buckets AS b
        SET available_units = v_available_units - v_granted_units,
            capacity_units = v_burst_capacity,
            rate_per_minute = v_rate_per_minute,
            last_refill_at = v_refill_time,
            refill_remainder = v_refill_remainder
        WHERE b.subject_kind = v_request.subject_kind
          AND b.subject_id = v_request.subject_id;

        subject_kind := v_request.subject_kind;
        subject_id := v_request.subject_id;
        granted_units := v_granted_units;
        capacity_units := v_burst_capacity;
        available_units := v_available_units - v_granted_units;
        rate_per_minute := v_rate_per_minute;
        refill_remainder := v_refill_remainder;
        RETURN NEXT;
    END LOOP;
END;
$$;

COMMENT ON FUNCTION acquire_rate_limit_leases(jsonb) IS
    'Refills authoritative rate-limit buckets and leases capacity to application nodes.';

-- restore_rate_limit_capacity adds unused process-local capacity back to the
-- authoritative buckets. It is used for asynchronous excess restoration and
-- during an orderly shutdown. Subjects deleted after leasing are ignored.
CREATE OR REPLACE FUNCTION restore_rate_limit_capacity(p_restorations jsonb)
    RETURNS void
    LANGUAGE plpgsql
    VOLATILE
AS $$
BEGIN
    IF p_restorations IS NULL OR jsonb_typeof(p_restorations) <> 'array' OR jsonb_array_length(p_restorations) = 0 THEN
        RAISE EXCEPTION 'rate-limit capacity restorations must be a non-empty JSON array';
    END IF;

    IF jsonb_array_length(p_restorations) > 64 THEN
        RAISE EXCEPTION 'too many rate-limit capacity restorations';
    END IF;

    IF EXISTS (
        SELECT
        FROM jsonb_to_recordset(p_restorations) AS r(subject_kind text, subject_id text, units integer)
        WHERE r.subject_kind IS NULL
           OR r.subject_kind NOT IN ('platform', 'organization', 'workspace', 'events')
           OR r.subject_id IS NULL
           OR (r.subject_kind = 'platform' AND r.subject_id <> 'platform')
           OR (r.subject_kind <> 'platform' AND r.subject_id !~ '^[1-9A-HJ-NP-Za-km-z]{12}$')
           OR r.units IS NULL
           OR r.units < 1
           OR r.units > 100000
    ) THEN
        RAISE EXCEPTION 'invalid rate-limit capacity restoration';
    END IF;

    IF EXISTS (
        SELECT r.subject_kind, r.subject_id
        FROM jsonb_to_recordset(p_restorations) AS r(subject_kind text, subject_id text, units integer)
        GROUP BY r.subject_kind, r.subject_id
        HAVING COUNT(*) > 1
    ) THEN
        RAISE EXCEPTION 'duplicate rate-limit capacity restoration subject';
    END IF;

    -- Lock rows in the same order as acquisition to avoid deadlocks with
    -- another process that is acquiring or restoring capacity for overlapping
    -- subjects. The materialized CTE completes that locking before the update.
    WITH restored AS MATERIALIZED (
        SELECT r.subject_kind, r.subject_id, r.units
        FROM jsonb_to_recordset(p_restorations) AS r(subject_kind text, subject_id text, units integer)
    ), locked AS MATERIALIZED (
        SELECT b.subject_kind, b.subject_id, r.units
        FROM restored r
        JOIN rate_limit_buckets b
            ON b.subject_kind = r.subject_kind
            AND b.subject_id = r.subject_id
        ORDER BY b.subject_kind, b.subject_id
        FOR UPDATE OF b
    )
    UPDATE rate_limit_buckets AS b
    SET available_units = LEAST(b.capacity_units, b.available_units + locked.units)
    FROM locked
    WHERE b.subject_kind = locked.subject_kind
      AND b.subject_id = locked.subject_id;
END;
$$;

COMMENT ON FUNCTION restore_rate_limit_capacity(jsonb) IS
    'Restores unused process-local capacity to authoritative rate-limit buckets.';

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

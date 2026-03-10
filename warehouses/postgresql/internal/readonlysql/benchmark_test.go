// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import (
	"strings"
	"testing"
)

type benchmarkQueryCase struct {
	name    string
	query   string
	wantErr bool
}

var benchmarkBIQueries = []benchmarkQueryCase{
	{
		name: "ProfilesProjectionFilter",
		query: `
SELECT
    customer_id,
    first_name,
    last_name,
    email,
    preferences_language
FROM meergo_profiles_5
WHERE preferences_newsletter = TRUE
  AND address_country = 'IT'
  AND email IS NOT NULL
ORDER BY last_name, first_name
LIMIT 500`,
	},
	{
		name: "ProfilesArrayAndTextFunctions",
		query: `
SELECT
    customer_id,
    lower(email) AS email_normalized,
    cardinality(preferences_categories) AS category_count,
    array_length(_identities, 1) AS identities_count,
    coalesce(first_name, 'unknown') AS first_name_fallback
FROM meergo_profiles_5
WHERE cardinality(preferences_categories) > 0
  AND lower(address_country) = 'it'
ORDER BY identities_count DESC
LIMIT 200`,
	},
	{
		name: "DailyEventsAggregation",
		query: `
SELECT
    date_trunc('day', received_at) AS event_day,
    event,
    count(*) AS event_count,
    sum(CASE WHEN context_network_wifi = TRUE THEN 1 ELSE 0 END) AS wifi_count,
    avg(context_location_latitude) AS avg_latitude
FROM meergo_events
WHERE received_at >= CURRENT_DATE
  AND received_at < CURRENT_DATE + interval '7 day'
GROUP BY date_trunc('day', received_at), event
ORDER BY event_day DESC, event_count DESC
LIMIT 1000`,
	},
	{
		name: "ProfilesEventsJoin",
		query: `
WITH recent_events AS (
    SELECT
        mpid,
        event,
        received_at,
        connection_id
    FROM meergo_events
    WHERE received_at >= CURRENT_TIMESTAMP (3) - interval '30 day'
),
event_counts AS (
    SELECT
        mpid,
        count(*) AS total_events,
        min(received_at) AS first_seen_at,
        max(received_at) AS last_seen_at
    FROM recent_events
    GROUP BY mpid
)
SELECT
    p.customer_id,
    p.email,
    p.preferences_language,
    e.total_events,
    e.first_seen_at,
    e.last_seen_at
FROM meergo_profiles_5 AS p
JOIN event_counts AS e
  ON p._mpid = e.mpid
WHERE p.email IS NOT NULL
ORDER BY e.total_events DESC, e.last_seen_at DESC
LIMIT 250`,
	},
	{
		name: "JsonAndCampaignBreakdown",
		query: `
SELECT
    date_part('month', received_at) AS month_number,
    context_campaign_source,
    context_campaign_medium,
    jsonb_extract_path_text(properties, 'source') AS property_source,
    jsonb_array_length(properties->'items') AS item_count,
    count(*) AS total_events
FROM meergo_events
WHERE context_campaign_source IS NOT NULL
  AND jsonb_array_length(properties->'items') > 0
GROUP BY
    date_part('month', received_at),
    context_campaign_source,
    context_campaign_medium,
    jsonb_extract_path_text(properties, 'source'),
    jsonb_array_length(properties->'items')
ORDER BY total_events DESC
LIMIT 500`,
	},
	{
		name: "CategoryUnnestRollup",
		query: `
WITH profile_categories AS (
    SELECT
        _mpid,
        unnest(preferences_categories) AS category
    FROM meergo_profiles_5
    WHERE cardinality(preferences_categories) > 0
)
SELECT
    category,
    count(*) AS profile_count,
    string_agg(customer_id, ',') AS sample_customers
FROM profile_categories AS c
JOIN meergo_profiles_5 AS p
  ON p._mpid = c._mpid
GROUP BY category
ORDER BY profile_count DESC
LIMIT 50`,
	},
	{
		name: "EventFormattingAndDimensions",
		query: `
SELECT
    to_char(received_at, 'YYYY-MM-DD') AS received_day,
    connection_id,
    channel,
    type,
    count(*) AS total_events,
    max(context_app_version) AS latest_app_version
FROM meergo_events
WHERE received_at >= LOCALTIMESTAMP (0) - interval '14 day'
GROUP BY
    to_char(received_at, 'YYYY-MM-DD'),
    connection_id,
    channel,
    type
ORDER BY received_day DESC, total_events DESC
LIMIT 1000`,
	},
}

var benchmarkIdentifierPathQueries = []benchmarkQueryCase{
	{
		name:  "AllowedFunctionSimple",
		query: "SELECT lower('x')",
	},
	{
		name:  "AllowedFunctionMixedCaseUnderscore",
		query: "SELECT DaTe_TrUnC('day', received_at) FROM meergo_events",
	},
	{
		name:  "QualifiedColumnReference",
		query: "SELECT e.connection_id FROM meergo_events AS e",
	},
	{
		name:  "QualifiedColumnReferenceWithComments",
		query: "SELECT e /*x*/ . /*y*/ connection_id FROM meergo_events AS e",
	},
	{
		name:    "RejectedFunctionDigits",
		query:   "SELECT AbC123(1)",
		wantErr: true,
	},
	{
		name:    "RejectedFunctionLeadingUnderscore",
		query:   "SELECT _FoO(1)",
		wantErr: true,
	},
	{
		name:    "RejectedQualifiedFunctionMixedCase",
		query:   "SELECT Pg_Catalog.LoWeR('ABC')",
		wantErr: true,
	},
	{
		name:    "RejectedQualifiedFunctionThreeParts",
		query:   "SELECT A.B.C(1)",
		wantErr: true,
	},
	{
		name:    "RejectedQualifiedFunctionThreePartsWithComments",
		query:   "SELECT A /*x*/ . /*y*/ B /*z*/ . /*w*/ C(1)",
		wantErr: true,
	},
}

// BenchmarkValidateReadOnlyBIQueries benchmarks ValidateReadOnly on representative BI queries.
func BenchmarkValidateReadOnlyBIQueries(b *testing.B) {
	b.Run("CurrentASCIIWordSetLookup", func(b *testing.B) {
		benchmarkValidateReadOnly(b, ValidateReadOnly)
	})
	b.Run("ASCIILenSwitchLookup", func(b *testing.B) {
		benchmarkValidateReadOnly(b, validateReadOnlyASCIILenSwitch)
	})
}

// BenchmarkValidateReadOnlyIdentifierPaths benchmarks focused identifier and
// dotted-name cases that are relevant when refactoring scanIdentifierChain.
func BenchmarkValidateReadOnlyIdentifierPaths(b *testing.B) {
	b.Run("CurrentASCIIWordSetLookup", func(b *testing.B) {
		benchmarkValidateReadOnlyCases(b, ValidateReadOnly, benchmarkIdentifierPathQueries)
	})
	b.Run("ASCIILenSwitchLookup", func(b *testing.B) {
		benchmarkValidateReadOnlyCases(b, validateReadOnlyASCIILenSwitch, benchmarkIdentifierPathQueries)
	})
}

func benchmarkValidateReadOnly(b *testing.B, validate func(string) error) {
	benchmarkValidateReadOnlyCases(b, validate, benchmarkBIQueries)
}

func benchmarkValidateReadOnlyCases(b *testing.B, validate func(string) error, cases []benchmarkQueryCase) {
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				err := validate(tc.query)
				switch {
				case tc.wantErr && err == nil:
					b.Fatalf("validate returned nil error for rejected query")
				case !tc.wantErr && err != nil:
					b.Fatalf("validate returned unexpected error: %v", err)
				}
			}
		})
	}
}

func validateReadOnlyASCIILenSwitch(query string) error {
	var seenSelect bool

	for i := 0; i < len(query); {
		switch c := query[i]; {
		case isSpace(c):
			i++
		case hasUnicodeQuotedIdentifierPrefix(query, i):
			return rejectUnicodeQuotedIdentifier()
		case hasUnicodeEscapeStringConstantPrefix(query, i):
			return rejectUnicodeEscapeStringConstant()
		case hasEscapeStringConstantPrefix(query, i):
			return rejectEscapeStringConstant()
		case hasBitStringConstantPrefix(query, i):
			return rejectBitStringConstant()
		case hasHexStringConstantPrefix(query, i):
			return rejectHexStringConstant()
		case c == ';':
			return rejectSemicolon()
		case c == '\'':
			next, err := skipSingleQuotedString(query, i)
			if err != nil {
				return err
			}
			i = next
		case c == '"':
			next, byteLen, err := scanDoubleQuotedIdentifier(query, i)
			if err != nil {
				return err
			}
			if byteLen > maxIdentifierBytes {
				return rejectIdentifierTooLong()
			}
			nextChar, err := nextVisibleChar(query, next)
			if err != nil {
				return err
			}
			if nextChar == '(' {
				return rejectQuotedIdentifierFunctionCall()
			}
			i = next
		case c == '-' && i+1 < len(query) && query[i+1] == '-':
			i = skipLineComment(query, i)
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			next, err := skipBlockComment(query, i)
			if err != nil {
				return err
			}
			i = next
		case c == '$':
			return rejectDollarSign()
		case isWordStart(c):
			name, err := scanIdentifierChain(query, i)
			if err != nil {
				return err
			}
			if handled, next, err := handleSpecialFormASCIILenSwitch(query, name); handled {
				if err != nil {
					return err
				}
				i = next
				continue
			}
			if name.isFunctionCall {
				if inNonFunctionCallTokenList(name.token) && !name.isQualified {
					i = name.next
					continue
				}
				if name.isQualified {
					return rejectQualifiedFunctionCall(name.normalizedChain(query))
				}
				if !inAllowedFunctionList(name.token) {
					return newFunctionNotAllowedError(name.normalizedToken())
				}
				if name.isSelect() {
					seenSelect = true
				}
				i = name.next
				continue
			}
			if name.isSelect() {
				seenSelect = true
				i = name.next
				continue
			}
			if inForbiddenTokenList(name.token) {
				return rejectForbiddenToken(name.token)
			}
			i = name.next
		default:
			i++
		}
	}

	if !seenSelect {
		return rejectNoVisibleSelect()
	}

	return nil
}

func handleSpecialFormASCIILenSwitch(query string, name scannedName) (handled bool, next int, err error) {
	if inDisallowedSpecialFormList(name.token) {
		return true, 0, rejectSpecialFormNotAllowed(strings.ToUpper(name.token))
	}
	if !inAllowedSpecialFormList(name.token) {
		return false, 0, nil
	}
	if !name.isFunctionCall {
		return true, name.next, nil
	}
	next, err = parseSpecialFormSuffix(query, name.token, name.next)
	return true, next, err
}

func inForbiddenTokenList(s string) bool {
	switch len(s) {
	case 2:
		return asciiEqualFold(s, "do")
	case 3:
		return asciiEqualFold(s, "set")
	case 4:
		return asciiEqualFold(s, "call") ||
			asciiEqualFold(s, "copy") ||
			asciiEqualFold(s, "drop") ||
			asciiEqualFold(s, "into") ||
			asciiEqualFold(s, "lock") ||
			asciiEqualFold(s, "move") ||
			asciiEqualFold(s, "show")
	case 5:
		return asciiEqualFold(s, "alter") ||
			asciiEqualFold(s, "begin") ||
			asciiEqualFold(s, "close") ||
			asciiEqualFold(s, "fetch") ||
			asciiEqualFold(s, "grant") ||
			asciiEqualFold(s, "merge") ||
			asciiEqualFold(s, "reset") ||
			asciiEqualFold(s, "start")
	case 6:
		return asciiEqualFold(s, "commit") ||
			asciiEqualFold(s, "create") ||
			asciiEqualFold(s, "delete") ||
			asciiEqualFold(s, "insert") ||
			asciiEqualFold(s, "listen") ||
			asciiEqualFold(s, "notify") ||
			asciiEqualFold(s, "revoke") ||
			asciiEqualFold(s, "update") ||
			asciiEqualFold(s, "vacuum")
	case 7:
		return asciiEqualFold(s, "analyze") ||
			asciiEqualFold(s, "cluster") ||
			asciiEqualFold(s, "comment") ||
			asciiEqualFold(s, "declare") ||
			asciiEqualFold(s, "discard") ||
			asciiEqualFold(s, "execute") ||
			asciiEqualFold(s, "prepare") ||
			asciiEqualFold(s, "refresh") ||
			asciiEqualFold(s, "release") ||
			asciiEqualFold(s, "reindex")
	case 8:
		return asciiEqualFold(s, "rollback") ||
			asciiEqualFold(s, "security") ||
			asciiEqualFold(s, "truncate") ||
			asciiEqualFold(s, "unlisten")
	case 9:
		return asciiEqualFold(s, "savepoint")
	case 10:
		return asciiEqualFold(s, "checkpoint") ||
			asciiEqualFold(s, "deallocate")
	}
	return false
}

func inNonFunctionCallTokenList(s string) bool {
	switch len(s) {
	case 2:
		return asciiEqualFold(s, "as") || asciiEqualFold(s, "in")
	case 3:
		return asciiEqualFold(s, "all") ||
			asciiEqualFold(s, "any") ||
			asciiEqualFold(s, "row")
	case 4:
		return asciiEqualFold(s, "some")
	case 6:
		return asciiEqualFold(s, "exists")
	}
	return false
}

func inAllowedFunctionList(s string) bool {
	switch len(s) {
	case 3:
		return asciiEqualFold(s, "abs") ||
			asciiEqualFold(s, "avg") ||
			asciiEqualFold(s, "max") ||
			asciiEqualFold(s, "min") ||
			asciiEqualFold(s, "sum")
	case 4:
		return asciiEqualFold(s, "ceil") ||
			asciiEqualFold(s, "left")
	case 5:
		return asciiEqualFold(s, "btrim") ||
			asciiEqualFold(s, "count") ||
			asciiEqualFold(s, "floor") ||
			asciiEqualFold(s, "least") ||
			asciiEqualFold(s, "lower") ||
			asciiEqualFold(s, "ltrim") ||
			asciiEqualFold(s, "right") ||
			asciiEqualFold(s, "round") ||
			asciiEqualFold(s, "rtrim") ||
			asciiEqualFold(s, "upper")
	case 6:
		return asciiEqualFold(s, "concat") ||
			asciiEqualFold(s, "length") ||
			asciiEqualFold(s, "nullif") ||
			asciiEqualFold(s, "unnest")
	case 7:
		return asciiEqualFold(s, "bool_or") ||
			asciiEqualFold(s, "ceiling") ||
			asciiEqualFold(s, "extract") ||
			asciiEqualFold(s, "replace") ||
			asciiEqualFold(s, "to_char")
	case 8:
		return asciiEqualFold(s, "bool_and") ||
			asciiEqualFold(s, "coalesce") ||
			asciiEqualFold(s, "greatest") ||
			asciiEqualFold(s, "json_agg")
	case 9:
		return asciiEqualFold(s, "array_agg") ||
			asciiEqualFold(s, "concat_ws") ||
			asciiEqualFold(s, "date_part") ||
			asciiEqualFold(s, "jsonb_agg") ||
			asciiEqualFold(s, "substring")
	case 10:
		return asciiEqualFold(s, "date_trunc") ||
			asciiEqualFold(s, "split_part") ||
			asciiEqualFold(s, "string_agg")
	case 11:
		return asciiEqualFold(s, "cardinality")
	case 12:
		return asciiEqualFold(s, "array_length") ||
			asciiEqualFold(s, "jsonb_typeof")
	case 17:
		return asciiEqualFold(s, "jsonb_object_keys")
	case 18:
		return asciiEqualFold(s, "jsonb_array_length") ||
			asciiEqualFold(s, "jsonb_extract_path")
	case 23:
		return asciiEqualFold(s, "jsonb_extract_path_text")
	}
	return false
}

func inAllowedSpecialFormList(s string) bool {
	switch len(s) {
	case 9:
		return asciiEqualFold(s, "localtime")
	case 12:
		return asciiEqualFold(s, "current_date")
	case 13:
		return asciiEqualFold(s, "current_time")
	case 14:
		return asciiEqualFold(s, "localtimestamp")
	case 17:
		return asciiEqualFold(s, "current_timestamp")
	}
	return false
}

func inDisallowedSpecialFormList(s string) bool {
	switch len(s) {
	case 4:
		return asciiEqualFold(s, "user")
	case 12:
		return asciiEqualFold(s, "current_role") ||
			asciiEqualFold(s, "current_user") ||
			asciiEqualFold(s, "session_user")
	case 14:
		return asciiEqualFold(s, "current_schema")
	case 15:
		return asciiEqualFold(s, "current_catalog")
	}
	return false
}

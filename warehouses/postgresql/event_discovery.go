// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/krenalis/krenalis/warehouses"
)

// DiscoverEventSources returns a row- and byte-bounded prefix of stored names.
// Long names are omitted visibly; read failures discard the entire result.
func (warehouse *PostgreSQL) DiscoverEventSources(ctx context.Context, request warehouses.EventSourceDiscoveryRequest) (warehouses.EventSourceDiscoveryPage, error) {

	err := request.Validate()
	if err != nil {
		return warehouses.EventSourceDiscoveryPage{}, err
	}
	query, args := eventDiscoverySourcesQuery(request)
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return warehouses.EventSourceDiscoveryPage{}, err
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return warehouses.EventSourceDiscoveryPage{}, eventDiscoveryReadError(ctx)
	}
	defer rows.Close()
	columnCount := len(rows.FieldDescriptions())
	if columnCount != 5 {
		rows.Close()
		err = rows.Err()
		if err != nil {
			return warehouses.EventSourceDiscoveryPage{}, eventDiscoveryReadError(ctx)
		}
		return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: event source schema mismatch")
	}

	result := warehouses.EventSourceDiscoveryPage{Sources: []warehouses.ObservedEventSource{}}
	eventType := eventDiscoveryText{maxBytes: 16}
	nameSize := eventDiscoveryText{maxBytes: 32}
	rank, charged := eventDiscoveryText{maxBytes: 32}, eventDiscoveryText{maxBytes: 32}
	remaining := warehouses.MaxEventDiscoverySourceBytes
	name := eventDiscoveryText{maxBytes: warehouses.MaxEventDiscoveryNameBytes, remaining: &remaining}
	count, previousCharge, readBytes := 0, 0, 0
	for rows.Next() {
		count++
		if count > request.Limit+1 {
			return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: event source row limit exceeded")
		}
		err = rows.Scan(&eventType, &nameSize, &name, &rank, &charged)
		if err != nil {
			return warehouses.EventSourceDiscoveryPage{}, eventDiscoveryReadError(ctx)
		}
		n, err := eventDiscoveryNumber(rank.value)
		if err != nil {
			return warehouses.EventSourceDiscoveryPage{}, err
		}
		used, err := eventDiscoveryNumber(charged.value)
		if err != nil {
			return warehouses.EventSourceDiscoveryPage{}, err
		}
		typ := string(eventType.value)
		if n != count || used < previousCharge || used > (request.Limit+1)*warehouses.MaxEventDiscoveryNameBytes ||
			(typ != "track" && typ != "page" && typ != "screen") {
			return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: invalid event source metadata")
		}
		previousCharge = used
		if n > request.Limit || used > warehouses.MaxEventDiscoverySourceBytes {
			if name.value != nil {
				return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: unexpected event source payload")
			}
			result.Truncated = true
			continue
		}
		source := warehouses.ObservedEventSource{Type: typ}
		if nameSize.value != nil {
			size, err := eventDiscoveryNumber(nameSize.value)
			if err != nil {
				return warehouses.EventSourceDiscoveryPage{}, err
			}
			if size > warehouses.MaxEventDiscoveryNameBytes {
				if name.value != nil {
					return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: unexpected long event name")
				}
				result.OmittedLongNames++
				result.Truncated = true
				continue
			}
			if name.value == nil || len(name.value) != size || !utf8.Valid(name.value) ||
				size > warehouses.MaxEventDiscoverySourceBytes-readBytes {
				return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: invalid recorded event name")
			}
			readBytes += size
			recordedName := string(name.value)
			source.RecordedName = &recordedName
		} else if name.value != nil {
			return warehouses.EventSourceDiscoveryPage{}, fmt.Errorf("warehouses/postgresql: unexpected unnamed source payload")
		}
		result.Sources = append(result.Sources, source)
	}
	err = rows.Err()
	if err != nil {
		return warehouses.EventSourceDiscoveryPage{}, eventDiscoveryReadError(ctx)
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return warehouses.EventSourceDiscoveryPage{}, eventDiscoveryReadError(ctx)
	}

	return result, nil
}

// SampleEventFields returns only a bounded recent prefix's two JSON roots.
// Oversized roots are counted as unread; no payload is returned for control
// rows.
func (warehouse *PostgreSQL) SampleEventFields(ctx context.Context, request warehouses.EventFieldSampleRequest) (warehouses.EventFieldSample, error) {

	err := request.Validate()
	if err != nil {
		return warehouses.EventFieldSample{}, err
	}
	query, args := eventDiscoverySampleQuery(request)
	pool, _, err := warehouse.connectionPool(ctx, false)
	if err != nil {
		return warehouses.EventFieldSample{}, err
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return warehouses.EventFieldSample{}, eventDiscoveryReadError(ctx)
	}
	defer rows.Close()
	columnCount := len(rows.FieldDescriptions())
	if columnCount != 6 {
		rows.Close()
		err = rows.Err()
		if err != nil {
			return warehouses.EventFieldSample{}, eventDiscoveryReadError(ctx)
		}
		return warehouses.EventFieldSample{}, fmt.Errorf("warehouses/postgresql: event sample schema mismatch")
	}

	result := warehouses.EventFieldSample{Rows: []warehouses.EventFieldSampleRow{}}
	rank, charged := eventDiscoveryText{maxBytes: 32}, eventDiscoveryText{maxBytes: 32}
	traitsSize, propertiesSize := eventDiscoveryText{maxBytes: 32}, eventDiscoveryText{maxBytes: 32}
	remaining := warehouses.MaxEventDiscoverySampleBytes
	traits := eventDiscoveryText{maxBytes: warehouses.MaxEventDiscoveryRootBytes, remaining: &remaining}
	properties := eventDiscoveryText{maxBytes: warehouses.MaxEventDiscoveryRootBytes, remaining: &remaining}
	count, previousCharge, readBytes := 0, 0, 0
	for rows.Next() {
		count++
		if count > request.Limit+1 {
			return warehouses.EventFieldSample{}, fmt.Errorf("warehouses/postgresql: event sample row limit exceeded")
		}
		err = rows.Scan(&rank, &charged, &traitsSize, &propertiesSize, &traits, &properties)
		if err != nil {
			return warehouses.EventFieldSample{}, eventDiscoveryReadError(ctx)
		}
		n, err := eventDiscoveryNumber(rank.value)
		if err != nil {
			return warehouses.EventFieldSample{}, err
		}
		used, err := eventDiscoveryNumber(charged.value)
		if err != nil {
			return warehouses.EventFieldSample{}, err
		}
		if n != count || used < previousCharge || used > 2*warehouses.MaxEventDiscoveryRootBytes*(request.Limit+1) {
			return warehouses.EventFieldSample{}, fmt.Errorf("warehouses/postgresql: invalid event sample metadata")
		}
		previousCharge = used
		if n > request.Limit || used > warehouses.MaxEventDiscoverySampleBytes {
			if traits.value != nil || properties.value != nil {
				return warehouses.EventFieldSample{}, fmt.Errorf("warehouses/postgresql: unexpected event sample payload")
			}
			result.HasMoreRows = true
			continue
		}
		sampled := warehouses.EventFieldSampleRow{}
		sampled.Traits, err = eventDiscoveryRoot(traitsSize.value, traits.value)
		if err != nil {
			return warehouses.EventFieldSample{}, err
		}
		sampled.Properties, err = eventDiscoveryRoot(propertiesSize.value, properties.value)
		if err != nil {
			return warehouses.EventFieldSample{}, err
		}
		for _, root := range []warehouses.EventJSONSample{sampled.Traits, sampled.Properties} {
			if len(root.Value) > warehouses.MaxEventDiscoverySampleBytes-readBytes {
				return warehouses.EventFieldSample{}, fmt.Errorf("warehouses/postgresql: event sample byte limit exceeded")
			}
			readBytes += len(root.Value)
		}
		result.Rows = append(result.Rows, sampled)
	}
	err = rows.Err()
	if err != nil {
		return warehouses.EventFieldSample{}, eventDiscoveryReadError(ctx)
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return warehouses.EventFieldSample{}, eventDiscoveryReadError(ctx)
	}

	return result, nil
}

// eventDiscoveryText checks raw text before copying any warehouse-controlled
// bytes.
type eventDiscoveryText struct {
	value     []byte
	maxBytes  int
	remaining *int // optional cumulative cursor budget, checked before copying
}

// ScanBytes copies bounded text from a pgx row.
func (value *eventDiscoveryText) ScanBytes(raw []byte) error {
	err := value.reserve(len(raw))
	if err != nil {
		return err
	}
	value.value = bytes.Clone(raw)
	return nil
}

func (value *eventDiscoveryText) reserve(size int) error {
	if size > value.maxBytes || value.remaining != nil && size > *value.remaining {
		return fmt.Errorf("warehouses/postgresql: event discovery text exceeds its limit")
	}
	if value.remaining != nil {
		*value.remaining -= size
	}
	return nil
}

func eventDiscoveryNumber(value []byte) (int, error) {
	n, err := strconv.Atoi(string(value))
	if err != nil {
		return 0, fmt.Errorf("warehouses/postgresql: invalid event discovery number")
	}
	if n < 0 {
		return 0, fmt.Errorf("warehouses/postgresql: negative event discovery number")
	}
	return n, nil
}

// eventDiscoveryReadError preserves cancellation without exposing warehouse
// data.
func eventDiscoveryReadError(ctx context.Context) error {
	if cause := ctx.Err(); cause != nil {
		return cause
	}
	return fmt.Errorf("warehouses/postgresql: event discovery read failed")
}

func eventDiscoveryRoot(size, value []byte) (warehouses.EventJSONSample, error) {
	if size == nil {
		if value != nil {
			return warehouses.EventJSONSample{}, fmt.Errorf("warehouses/postgresql: unexpected absent JSON payload")
		}
		return warehouses.EventJSONSample{}, nil
	}
	n, err := eventDiscoveryNumber(size)
	if err != nil {
		return warehouses.EventJSONSample{}, err
	}
	if n > warehouses.MaxEventDiscoveryRootBytes {
		if value != nil {
			return warehouses.EventJSONSample{}, fmt.Errorf("warehouses/postgresql: unexpected oversized JSON payload")
		}
		return warehouses.EventJSONSample{Present: true, Uninspected: true}, nil
	}
	if value == nil || n != len(value) || !utf8.Valid(value) {
		return warehouses.EventJSONSample{}, fmt.Errorf("warehouses/postgresql: invalid sampled JSON payload")
	}
	return warehouses.EventJSONSample{Present: true, Value: value}, nil
}

// eventDiscoverySampleQuery keeps the control row free of serialized JSON.
func eventDiscoverySampleQuery(request warehouses.EventFieldSampleRequest) (string, []any) {
	var selectedName any
	if request.Source.RecordedName != nil {
		selectedName = *request.Source.RecordedName
	}
	query := fmt.Sprintf(`WITH parameters AS (
SELECT CAST($1 AS timestamp(6)) AS start_at,
CAST($2 AS timestamp(6)) AS end_at,
(CAST($3 AS text)) COLLATE "C" AS selected_type,(CAST($4 AS text)) COLLATE "C" AS selected_name
), source_values AS (
SELECT (CAST("type" AS text)) COLLATE "C" AS event_type,
(CASE WHEN (CAST("type" AS text)) COLLATE "C"='track'
THEN CAST("event" AS text) ELSE CAST("name" AS text) END) COLLATE "C" AS recorded_name,
"timestamp" AS sample_time,"message_id" AS sample_id,"traits","properties"
FROM "krenalis_events" CROSS JOIN parameters p WHERE "timestamp">=p.start_at AND "timestamp"<p.end_at
), limited AS (
SELECT source_values.* FROM source_values CROSS JOIN parameters p
WHERE event_type=p.selected_type AND recorded_name IS NOT DISTINCT FROM p.selected_name
ORDER BY sample_time DESC,sample_id DESC LIMIT %[1]d
), ranked AS (
SELECT limited.*,ROW_NUMBER() OVER (ORDER BY sample_time DESC,sample_id DESC) AS rn FROM limited
), sizes AS (
SELECT ranked.*,CASE WHEN rn<=%[2]d THEN OCTET_LENGTH(CAST("traits" AS text)) END AS traits_size,
CASE WHEN rn<=%[2]d THEN OCTET_LENGTH(CAST("properties" AS text)) END AS properties_size FROM ranked
), budgeted AS (
SELECT sizes.*,SUM(CASE WHEN traits_size<=%[3]d THEN traits_size ELSE 0 END +
CASE WHEN properties_size<=%[3]d THEN properties_size ELSE 0 END)
OVER (ORDER BY rn ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS charged FROM sizes
)
SELECT CAST(rn AS text),CAST(charged AS text),CAST(traits_size AS text),CAST(properties_size AS text),
CASE WHEN rn<=%[2]d AND charged<=%[4]d AND traits_size<=%[3]d THEN CAST("traits" AS text) END,
CASE WHEN rn<=%[2]d AND charged<=%[4]d AND properties_size<=%[3]d THEN CAST("properties" AS text) END
FROM budgeted ORDER BY budgeted.rn`, request.Limit+1, request.Limit,
		warehouses.MaxEventDiscoveryRootBytes, warehouses.MaxEventDiscoverySampleBytes)
	return query, []any{request.Window.Start.UTC(), request.Window.End.UTC(), request.Source.Type, selectedName}
}

// eventDiscoverySourcesQuery limits distinct names before transporting text.
func eventDiscoverySourcesQuery(request warehouses.EventSourceDiscoveryRequest) (string, []any) {
	query := fmt.Sprintf(`WITH parameters AS (
SELECT CAST($1 AS timestamp(6)) AS start_at,
CAST($2 AS timestamp(6)) AS end_at,
(CAST($3 AS text)) COLLATE "C" AS type_filter,(CAST($4 AS text)) COLLATE "C" AS needle
), source_values AS (
SELECT (CAST("type" AS text)) COLLATE "C" AS event_type,
(CASE WHEN (CAST("type" AS text)) COLLATE "C"='track'
THEN CAST("event" AS text) ELSE CAST("name" AS text) END) COLLATE "C" AS recorded_name
FROM "krenalis_events" CROSS JOIN parameters p WHERE "timestamp">=p.start_at AND "timestamp"<p.end_at
), distinct_names AS (
SELECT DISTINCT event_type,recorded_name FROM source_values CROSS JOIN parameters p
WHERE event_type IN ('track','page','screen') AND (p.type_filter='' OR event_type=p.type_filter)
AND (p.needle='' OR strpos(recorded_name, p.needle)>0)
), limited AS (
SELECT event_type,recorded_name,OCTET_LENGTH(recorded_name) AS name_size,
ROW_NUMBER() OVER (ORDER BY event_type,recorded_name NULLS FIRST) AS rn
FROM distinct_names ORDER BY event_type,recorded_name NULLS FIRST LIMIT %[1]d
), budgeted AS (
SELECT event_type,recorded_name,name_size,rn,
SUM(CASE WHEN name_size<=%[3]d THEN name_size ELSE 0 END)
OVER (ORDER BY rn ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS charged FROM limited
)
SELECT event_type,CAST(name_size AS text),
CASE WHEN rn<=%[2]d AND charged<=%[4]d AND name_size<=%[3]d THEN recorded_name END,
CAST(rn AS text),CAST(charged AS text) FROM budgeted ORDER BY budgeted.rn`, request.Limit+1, request.Limit,
		warehouses.MaxEventDiscoveryNameBytes, warehouses.MaxEventDiscoverySourceBytes)
	return query, []any{request.Window.Start.UTC(), request.Window.End.UTC(), request.Type, request.NameContains}
}

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	_connector "chichi/connector"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// importFromDatabase imports the users from a database.
func (ac *Action) importFromDatabase() error {

	connection := ac.action.Connection()
	connector := connection.Connector()

	query, err := compileActionQuery(ac.action.Query, noQueryLimit)
	if err != nil {
		return actionExecutionError{err}
	}
	fh := ac.newFirehose(context.Background())
	c, err := _connector.RegisteredDatabase(connector.Name).Open(fh.ctx, &_connector.DatabaseConfig{
		Role:     _connector.SourceRole,
		Settings: connection.Settings,
		Firehose: fh,
	})
	if err != nil {
		return actionExecutionError{fmt.Errorf("cannot connect to the connector: %s", err)}
	}
	schema, rawRows, err := c.Query(query)
	if err != nil {
		return actionExecutionError{err}
	}
	defer rawRows.Close()
	properties := schema.Properties()
	var hasIdentity bool
	var hasTimestamp bool
	for _, p := range properties {
		if p.Name == identityColumn {
			hasIdentity = true
		}
		if p.Name == timestampColumn {
			hasTimestamp = true
		}
	}
	if !hasIdentity {
		return actionExecutionError{fmt.Errorf("missing identity column %q", identityColumn)}
	}
	now := time.Now().UTC()
	dest := make([]any, len(properties))
	for rawRows.Next() {
		row := make(map[string]any, len(properties))
		for i, p := range properties {
			dest[i] = databaseScanValue{property: p, row: row}
		}
		if err := rawRows.Scan(dest...); err != nil {
			return actionExecutionError{fmt.Errorf("query execution failed: %s", err)}
		}
		ts := now
		if hasTimestamp {
			ts = row[timestampColumn].(time.Time)
		}
		fh.SetUser(row[identityColumn].(string), row, ts, nil)
	}
	if err = rawRows.Err(); err != nil {
		return actionExecutionError{fmt.Errorf("an error occurred closing the database: %s", err)}
	}
	// Handle errors occurred in the firehose.
	if fh.err != nil {
		return fh.err
	}

	return nil
}

// databaseScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type databaseScanValue struct {
	property types.Property
	row      map[string]any
}

func (sv databaseScanValue) Scan(src any) error {
	name := sv.property.Name
	if src == nil {
		if !sv.property.Nullable {
			return fmt.Errorf("column %s is non-nullable, but the database returned a NULL value", name)
		}
		sv.row[name] = nil
		return nil
	}
	typ := sv.property.Type
	var value any
	var valid bool
	switch typ.PhysicalType() {
	case types.PtBoolean:
		if _, ok := src.(bool); ok {
			value = src
			valid = true
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var v int64
		switch src := src.(type) {
		case int64:
			v = src
			valid = true
		case []byte:
			var err error
			v, err = strconv.ParseInt(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.IntRange()
			if v < min || v > max {
				return fmt.Errorf("database returnd a value of %d for column %s which is not within the expected range of [%d, %d]", v,
					name, min, max)
			}
			value = int(v)
		}
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		var v uint64
		switch src := src.(type) {
		case []byte:
			var err error
			v, err = strconv.ParseUint(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.UIntRange()
			if v < min || v > max {
				return fmt.Errorf("database returnd a value of %d for column %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = uint(v)
		}
	case types.PtFloat, types.PtFloat32:
		var v float64
		switch src := src.(type) {
		case float64:
			v = src
			valid = true
		case []byte:
			var err error
			size := 64
			if typ.PhysicalType() == types.PtFloat32 {
				size = 32
			}
			v, err = strconv.ParseFloat(string(src), size)
			valid = err == nil
		}
		if valid {
			min, max := typ.FloatRange()
			if v < min || v > max {
				return fmt.Errorf("database returnd a value of %f for column %s which is not within the expected range of [%f, %f]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDecimal:
		var v decimal.Decimal
		switch src := src.(type) {
		case string:
			var err error
			v, err = decimal.NewFromString(src)
			valid = err == nil
		case []byte:
			var err error
			v, err = decimal.NewFromString(string(src))
			valid = err == nil
		}
		if valid {
			min, max := typ.DecimalRange()
			if v.LessThan(min) || v.GreaterThan(max) {
				return fmt.Errorf("database returnd a value of %s for column %s which is not within the expected range of [%s, %s]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDate, types.PtDateTime:
		if _, ok := src.(time.Time); ok {
			value = src
			valid = true
		}
	case types.PtTime:
		if src, ok := src.([]byte); ok {
			var err error
			value, err = time.Parse(time.TimeOnly, string(src))
			valid = err == nil
		}
	case types.PtYear:
		switch y := src.(type) {
		case int64:
			if valid = types.MinYear <= y && y <= types.MaxYear; valid {
				value = int(y)
			}
		case []byte:
			year, err := strconv.Atoi(string(y))
			value = year
			valid = err == nil && types.MinYear <= year && year <= types.MaxYear
		}
	case types.PtUUID:
		if s, ok := src.(string); ok {
			var err error
			value, err = uuid.Parse(s)
			valid = err == nil
		}
	case types.PtJSON:
		if s, ok := src.([]byte); ok {
			if valid = json.Valid(s); valid {
				value = json.RawMessage(s)
			}
		}
	case types.PtInet:
		if s, ok := src.(string); ok {
			var err error
			value, err = netip.ParseAddr(s)
			valid = err == nil
		}
	case types.PtText:
		var v string
		switch s := src.(type) {
		case string:
			v = s
			valid = true
		case []byte:
			v = string(s)
			valid = true
		}
		if valid {
			if !utf8.ValidString(v) {
				return fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
					abbreviate(v, 20), name)
			}
			if l, ok := typ.ByteLen(); ok && len(v) > l {
				return fmt.Errorf("database returned a value of %q for column %s, which is longer than %d bytes",
					abbreviate(v, 20), name, l)
			}
			if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
				return fmt.Errorf("database returned a value of %q for column %s, which is longer than %d characters",
					abbreviate(v, 20), name, l)
			}
			value = v
		}
	}
	if !valid {
		return fmt.Errorf("database returned a value of %v for column %s, but it cannot be converted to the %s type",
			src, name, typ.PhysicalType())
	}
	sv.row[name] = value
	return nil
}

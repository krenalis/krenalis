//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// CanBeIdentifier reports whether a property with type t can be used as
// identifier in the Identity Resolution.
func CanBeIdentifier(t types.Type) bool {
	switch t.Kind() {
	case types.IntKind,
		types.UintKind,
		types.UUIDKind,
		types.InetKind,
		types.TextKind:
		return true
	case types.DecimalKind:
		return t.Scale() == 0
	default:
		return false
	}
}

// An unflatRowFunc function unflats a row read from the data warehouse into a
// map[string]any value.
type unflatRowFunc func(row []any) map[string]any

// columnsFromProperties returns the columns corresponding to the provided
// properties paths, based on the mapping defined in columnByProperty that can
// be used in a query to a data warehouse. Additionally, it returns a function
// that can be used to transform a row retrieved from a data warehouse into its
// map representation.
func columnsFromProperties(properties []string, columnByProperty map[string]warehouses.Column) ([]warehouses.Column, unflatRowFunc) {
	pk := &propertyKey{}
	columns := make([]warehouses.Column, 0, len(properties))
	for _, path := range properties {
		column, ok := columnByProperty[path]
		if ok {
			pk.add(path, len(columns))
			columns = append(columns, column)
			continue
		}
		path += "."
		for property, column := range columnByProperty {
			if strings.HasPrefix(property, path) {
				pk.add(property, len(columns))
				columns = append(columns, column)
			}
		}
	}
	unflat := func(row []any) map[string]any {
		return unflatRow(pk, row)
	}
	return columns, unflat
}

type propertyKey struct {
	name       string
	index      int
	properties []propertyKey
}

// add adds a property path with a given index.
func (pk *propertyKey) add(path string, index int) {
	var ok bool
	var name string
Path:
	for {
		name, path, ok = strings.Cut(path, ".")
		if !ok {
			pk.properties = append(pk.properties, propertyKey{name: name, index: index})
			return
		}
		for i := 0; i < len(pk.properties); i++ {
			if pk.properties[i].name == name {
				pk = &pk.properties[i]
				continue Path
			}
		}
		p := propertyKey{name: name, properties: []propertyKey{}}
		pk.properties = append(pk.properties, p)
		pk = &pk.properties[len(pk.properties)-1]
	}
}

func unflatRow(pk *propertyKey, row []any) map[string]any {
	v := map[string]any{}
	for _, p := range pk.properties {
		if p.properties == nil {
			v[p.name] = row[p.index]
			continue
		}
		v[p.name] = unflatRow(&p, row)
	}
	return v
}

// exprFromFilter returns a warehouses.Expr expression from a filter.
// columnFromProperty maps each non-Object property with its column.
func exprFromFilter(filter *state.Filter, columnFromProperty map[string]warehouses.Column) (warehouses.Expr, error) {
	op := warehouses.LogicalOperatorAnd
	if filter.Logical == "any" {
		op = warehouses.LogicalOperatorOr
	}
	exp := warehouses.NewMultiExpr(op, make([]warehouses.Expr, len(filter.Conditions)))
	for i, cond := range filter.Conditions {
		column, ok := columnFromProperty[cond.Property]
		if !ok {
			return nil, fmt.Errorf("property path %s does not exist", cond.Property)
		}
		var op warehouses.Operator
		switch cond.Operator {
		case "is":
			op = warehouses.OperatorEqual
		case "is not":
			op = warehouses.OperatorNotEqual
		default:
			return nil, errors.New("invalid operator")
		}
		var value any
		switch column.Type.Kind() {
		case types.BooleanKind:
			value = false
			if cond.Value == "true" {
				value = true
			}
		case types.IntKind:
			value, _ = strconv.Atoi(cond.Value)
		case types.UintKind:
			v, _ := strconv.ParseUint(cond.Value, 10, 64)
			value = uint(v)
		case types.FloatKind:
			value, _ = strconv.ParseFloat(cond.Value, 64)
		case types.DecimalKind:
			value = decimal.RequireFromString(cond.Value)
		case types.DateTimeKind:
			value, _ = time.Parse(time.DateTime, cond.Value)
		case types.DateKind:
			value, _ = time.Parse(time.DateOnly, cond.Value)
		case types.TimeKind:
			value, _ = time.Parse("15:04:05.999999999", cond.Value)
		case types.YearKind:
			value, _ = strconv.Atoi(cond.Value)
		case types.UUIDKind, types.TextKind:
			value = cond.Value
		case types.JSONKind:
			value = json.RawMessage(cond.Value)
		case types.InetKind:
			value, _ = netip.ParseAddr(cond.Value)
		default:
			return nil, fmt.Errorf("unexpected type %s", column.Type)
		}
		exp.Operands[i] = warehouses.NewBaseExpr(column, op, value)
	}
	return exp, nil
}

// columnByProperty returns a mapping from properties of the schema to their
// respective columns. It assumes that for a property path like "a.b.c", the
// corresponding column is named "a_b_c".
func columnByProperty(schema types.Type) map[string]warehouses.Column {
	columnByProperty := map[string]warehouses.Column{}
	for path, p := range types.Walk(schema) {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		name := strings.ReplaceAll(path, ".", "_")
		columnByProperty[path] = warehouses.Column{Name: name, Type: p.Type, Nullable: p.Nullable}
	}
	return columnByProperty
}

// identityColumnByProperty returns a mapping from user identity properties to
// their corresponding columns.
//
// This mapping is derived from the user's property-to-column mapping,
// substituting meta properties with the meta properties of user identity.
func identityColumnByProperty(userColumnByProperty map[string]warehouses.Column) map[string]warehouses.Column {
	columns := map[string]warehouses.Column{
		"__pk__":               {Name: "__pk__", Type: types.Int(32)},
		"__action__":           {Name: "__action__", Type: types.Int(32)},
		"__is_anonymous__":     {Name: "__is_anonymous__", Type: types.Boolean()},
		"__identity_id__":      {Name: "__identity_id__", Type: types.Text()},
		"__connection__":       {Name: "__connection__", Type: types.Int(32)},
		"__anonymous_ids__":    {Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true},
		"__last_change_time__": {Name: "__last_change_time__", Type: types.DateTime()},
		"__gid__":              {Name: "__gid__", Type: types.UUID()},
	}
	for property, column := range userColumnByProperty {
		if !isMetaProperty(property) {
			columns[property] = column
		}
	}
	return columns
}

// isMetaProperty reports whether the given property name refers to a property
// considered a meta property by a data warehouse.
func isMetaProperty(name string) bool {
	return len(name) > 5 && strings.HasPrefix(name, "__") && strings.HasSuffix(name, "__")
}

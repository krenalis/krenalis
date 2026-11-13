// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Statement is an application-generated statement with a normalized result
// schema. SQL must never come from a filter payload or an external request.
type Statement struct {
	SQL       string
	Arguments []Argument
	Columns   []warehouses.Column
	// CountNodes identifies the Count rows in an explanation's preorder.
	CountNodes []bool
	// SumNodes identifies total rows; nil retains the Count-only protocol.
	SumNodes []bool
	// AverageNodes distinguishes Average inputs from Sum bounds.
	AverageNodes []bool
	// ExperienceNodes identifies selection candidate/absence metadata rows.
	ExperienceNodes []bool
	// PairNodes identifies ordered Last/Previous selection metadata rows.
	PairNodes []bool
}

// ReadExplanation consumes the explanation protocol compiled by this statement.
// Decoding, cancellation, source or Close failure discards the whole result.
func (q Statement) ReadExplanation(ctx context.Context, rows warehouses.Rows, profile string) ([]warehouses.FilterNodeOutcome, error) {

	result, err := readExplanation(ctx, rows, q.CountNodes, q.SumNodes, q.ExperienceNodes, q.PairNodes, profile)
	if err != nil {
		return nil, err
	}

	if q.AverageNodes != nil && len(q.AverageNodes) != len(q.CountNodes) {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid Average explanation nodes")
	}
	for i, average := range q.AverageNodes {
		if average && (q.SumNodes == nil || !q.SumNodes[i] || q.CountNodes[i]) {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid Average explanation inputs")
		}
	}

	for i := range result {
		if q.AverageNodes == nil || !q.AverageNodes[i] {
			continue
		}
		bounds := result[i].Sum
		if bounds == nil {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: missing Average explanation inputs")
		}
		values := &warehouses.OccurrenceAverageValues{Certain: bounds.Certain, Possible: bounds.Possible}
		if bounds.Certain == bounds.Possible {
			values.Count = &bounds.Certain
			if bounds.Lower != nil && bounds.Upper != nil {
				if !bounds.Lower.Equal(*bounds.Upper) {
					return nil, fmt.Errorf("warehouses/internal/profilefilter: inconsistent Average total")
				}
				values.Sum = bounds.Lower
			}
		}
		result[i].Sum, result[i].Average = nil, values
	}

	return result, nil
}

// Values validates the transport envelope and returns driver arguments.
// DateTime arguments preserve microseconds as UTC timestamp-without-zone text.
func (q Statement) Values() ([]any, error) {
	if len(q.SQL) == 0 || len(q.SQL) > 1<<20 || len(q.Columns) == 0 || len(q.Columns) > 100 || len(q.Arguments) > 1000 {
		return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid compiled query envelope")
	}
	for _, column := range q.Columns {
		if !warehouses.IsValidIdentifier(column.Name) || !column.Type.Valid() {
			return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid compiled result column")
		}
	}
	values := make([]any, len(q.Arguments))
	for i, arg := range q.Arguments {
		switch arg.Type.Kind() {
		case types.StringKind:
			value, ok := arg.Value.(string)
			if !ok || len(value) > 4096 {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid compiled text argument")
			}
			values[i] = value
		case types.DateTimeKind:
			value, ok := arg.Value.(time.Time)
			value = value.UTC()
			if !ok || value.IsZero() || value.Year() < 1 || value.Year() > 9999 || value.Nanosecond()%1000 != 0 {
				return nil, fmt.Errorf("warehouses/internal/profilefilter: invalid compiled timestamp argument")
			}
			values[i] = value.UTC().Format("2006-01-02 15:04:05.000000")
		default:
			return nil, fmt.Errorf("warehouses/internal/profilefilter: unsupported compiled argument type")
		}
	}
	return values, nil
}

// Argument is a typed internal SQL parameter.
type Argument struct {
	Type  types.Type
	Value any
}

// Dialect supplies physical operations used only below the warehouse boundary.
type Dialect struct {
	ValueSize       func(string) string
	QuoteIdentifier func(string) string
	Record          func(warehouses.Expr) (string, error)
	ExactText       func(warehouses.Column) (string, error)
	KnownText       func(warehouses.Column, []string) (string, error)
	ExactDecimal    func(warehouses.DecimalReading) (string, error)
	Parameter       func(int, types.Type) (string, error)
}

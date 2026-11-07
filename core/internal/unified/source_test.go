// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestMonetarySourceType keeps same-named pages out of monetary readings.
func TestMonetarySourceType(t *testing.T) {
	for _, typ := range []string{"", "track", "page", "screen"} {
		for _, kind := range []string{"currency", "sum"} {
			for _, entry := range []string{"Validate", "Prepare"} {
				t.Run(typ+"/"+kind+"/"+entry, func(t *testing.T) {
					at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
					scope := Scope{Type: typ, Event: "Order Completed", Grain: "registered-message",
						Start: at.Add(-time.Hour), End: at}
					var root *Rule
					if kind == "currency" {
						currency, err := ResolveNativeCurrency("commerce.order_completed.currency",
							"1", "EUR", scope.Event)
						if err != nil {
							t.Fatalf("expected resolved currency fixture, got %v", err)
						}
						root = &Rule{Count: &Count{Scope: scope, Currency: currency}}
					} else {
						field, err := ResolveMonetaryField("commerce.order_completed.total", "1")
						if err != nil {
							t.Fatalf("expected resolved monetary fixture, got %v", err)
						}
						root = &Rule{Sum: &Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
							Test: SumTest{Operator: GreaterOrEqual, Value: decimal.MustParse("30")}}}
					}
					source := SourceBinding{Occurrences: warehouses.OccurrenceSource{Table: "events",
						Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
						Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
						Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
						Properties: warehouses.Column{Name: "properties", Type: types.JSON()},
						Record: func(*state.Where) (warehouses.Expr, error) {
							return warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.String()},
								warehouses.OpIs, "track"), nil
						}}
					var err error
					switch entry {
					case "Validate":
						err = Validate(root, at, Page{Limit: 1})
					case "Prepare":
						_, err = Prepare(root, Bindings{}, at, source)
					default:
						t.Fatalf("expected internal query entry point, got %s", entry)
					}
					invalid := typ == "page" || typ == "screen"
					if err != nil {
						if _, unsupported := errors.AsType[*UnsupportedError](err); unsupported || !invalid {
							t.Fatalf("expected invalid=%v for monetary source %s, got %v", invalid, typ, err)
						}
						return
					}
					if invalid {
						t.Fatal("expected non-track monetary source rejected, got success")
					}
				})
			}
		}
	}
}

// TestRecordedSourceAdmission protects both internal query entry points.
func TestRecordedSourceAdmission(t *testing.T) {
	for _, test := range []struct {
		typ, name string
		invalid   bool
	}{
		{"", "Purchase", false},
		{"track", "Purchase", false},
		{"page", "Pricing", false},
		{"page", "", false},
		{"screen", "Home", false},
		{"screen", "", false},
		{"track", "", true},
		{"", "", true},
		{"identify", "Pricing", true},
		{"Page", "Pricing", true},
		{"page", "\x00", true},
	} {
		for _, entry := range []string{"Validate", "Prepare"} {
			t.Run(fmt.Sprintf("%s/%s/%s", test.typ, test.name, entry), func(t *testing.T) {
				at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
				root := &Rule{Count: &Count{Scope: Scope{Type: test.typ, Event: test.name,
					Grain: "registered-message", Start: at.Add(-time.Hour), End: at}}}
				source := SourceBinding{Occurrences: warehouses.OccurrenceSource{Table: "events",
					Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
					Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
					Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
					Record: func(*state.Where) (warehouses.Expr, error) {
						return warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.String()},
							warehouses.OpIs, "page"), nil
					}}
				var err error
				switch entry {
				case "Validate":
					err = Validate(root, at, Page{Limit: 1})
				case "Prepare":
					_, err = Prepare(root, Bindings{}, at, source)
				default:
					t.Fatalf("expected internal query entry point, got %s", entry)
				}
				if err != nil {
					if _, unsupported := errors.AsType[*UnsupportedError](err); unsupported || !test.invalid {
						t.Fatalf("expected invalid=%v, got %v", test.invalid, err)
					}
					return
				}
				if test.invalid {
					t.Fatal("expected invalid recorded source, got success")
				}
			})
		}
	}
}

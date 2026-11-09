// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"fmt"
	"slices"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func (c *compiler) money(node *warehouses.MoneyQualification, base warehouses.RowQualification) (string, error) {
	if c.dialect.ExactDecimal == nil {
		return "", fmt.Errorf("warehouses/internal/profilefilter: missing exact decimal reading")
	}
	currency := ""
	if group, ok := base.(*warehouses.QualificationGroup); ok && group.Operator == warehouses.OpAnd {
		for _, child := range group.Children {
			known, ok := child.(*warehouses.KnownTextEqual)
			if !ok || known.Column.Name != node.Currency.Column.Name ||
				!types.Equal(known.Column.Type, node.Currency.Column.Type) || !slices.Equal(known.Keys, node.Currency.Keys) {
				continue
			}
			if known.Value != node.Currency.Value {
				return "FALSE", nil
			}
			currency = "TRUE"
			break
		}
	}
	if currency == "" {
		var err error
		currency, err = c.qualification(&node.Currency)
		if err != nil {
			return "", err
		}
	}
	amount, err := c.dialect.ExactDecimal(node.Amount)
	if err != nil {
		return "", err
	}
	threshold := fmt.Sprintf("CAST('%.4f' AS DECIMAL(38,4))", node.Threshold)
	operator := "<="
	if node.AtLeast {
		operator = ">="
	}
	comparison := "(" + amount + ")" + operator + threshold
	if currency == "TRUE" {
		return comparison, nil
	}
	return "(" + currency + ") AND (" + comparison + ")", nil
}

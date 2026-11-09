// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"fmt"

	"github.com/krenalis/krenalis/warehouses"
)

func (c *compiler) money(node *warehouses.MoneyQualification) (string, error) {
	if c.dialect.ExactDecimal == nil {
		return "", fmt.Errorf("warehouses/internal/profilefilter: missing exact decimal reading")
	}
	currency, err := c.qualification(&node.Currency)
	if err != nil {
		return "", err
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
	return "(" + currency + ") AND (" + comparison + ")", nil
}

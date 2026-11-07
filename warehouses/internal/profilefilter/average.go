// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"fmt"

	"github.com/krenalis/krenalis/warehouses"
)

func (c *compiler) average(node *warehouses.DeterminedOccurrenceAverage) (string, error) {

	alias, err := c.monetaryPopulation(&warehouses.IndependentOccurrenceSum{
		Source: node.Source, Start: node.Start, End: node.End, UnboundedStart: node.UnboundedStart,
		Qualification: node.Qualification, Value: node.Value})
	if err != nil {
		return "", err
	}

	lo, hi := "COALESCE("+alias+".lo,0)", "COALESCE("+alias+".hi,0)"
	operator := "<="
	if node.AtLeast {
		operator = ">="
	}
	if node.Strict {
		operator = operator[:1]
	}
	// Both operands fit DECIMAL(38,4): |t|<1e14 and n<=MaxInt64.
	// In particular no quotient, floating point or rounded average is used.
	test := "COALESCE(" + alias + ".sum_lo,0)" + operator +
		fmt.Sprintf("(CAST('%.4f' AS DECIMAL(18,4))*CAST(%s AS DECIMAL(19,0)))", node.Threshold, lo)
	if node.DeterminedResult != nil {
		test = "FALSE"
		if *node.DeterminedResult {
			test = "TRUE"
		}
	}
	test = "CASE WHEN " + lo + "=" + hi + " AND " + lo + ">0 AND COALESCE(" + alias +
		".unknowns,0)=0 THEN (" + test + ") ELSE NULL END"

	return withSampleMinimum(node.Minimum, lo, hi, test), nil
}

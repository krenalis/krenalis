// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"fmt"
	"strconv"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func (c *compiler) sum(node *warehouses.IndependentOccurrenceSum) (string, error) {

	if c.dialect.ExactDecimal == nil {
		return "", fmt.Errorf("warehouses/internal/profilefilter: missing exact decimal reading")
	}
	n := strconv.Itoa(c.counts)
	c.counts++
	q := c.dialect.QuoteIdentifier
	lower := ""
	if !node.UnboundedStart {
		start, err := c.parameter(types.DateTime(), node.Start)
		if err != nil {
			return "", err
		}
		lower = start + " AS lower_bound,"
	}
	end, err := c.parameter(types.DateTime(), node.End)
	if err != nil {
		return "", err
	}
	qual := "TRUE"
	if node.Qualification != nil {
		qual, err = c.qualification(node.Qualification)
		if err != nil {
			return "", err
		}
	}
	reading := node.Value
	reading.Column.Name = "sum_value"
	value, err := c.dialect.ExactDecimal(reading)
	if err != nil {
		return "", err
	}
	c.ctes = append(c.ctes, "bounds"+n+" AS (SELECT "+lower+end+" AS upper_bound)")
	within := q(node.Source.Timestamp.Name) + "<(SELECT upper_bound FROM bounds" + n + ")"
	if !node.UnboundedStart {
		within = q(node.Source.Timestamp.Name) + ">=(SELECT lower_bound FROM bounds" + n + ") AND " + within
	}
	table, id, profile := q(node.Source.Table), q(node.Source.Identity.Name), q(node.Source.Profile.Name)
	profileOnly, identityProfile := "", ""
	if c.profile != "" {
		profileOnly = " AND CAST(" + profile + " AS VARCHAR(36))=" + c.profile
		identityProfile = " AND " + id + " IN (SELECT " + id + " FROM " + table +
			" WHERE CAST(" + profile + " AS VARCHAR(36))=" + c.profile + ")"
	}

	// As in Count, inspect every copy of a seeded identity before reducing it.
	// Read amounts after qualification, so certainly excluded values are inert.
	c.ctes = append(c.ctes, "ids"+n+" AS (SELECT DISTINCT "+id+" AS occurrence_id FROM "+table+
		" WHERE "+within+identityProfile+")")
	c.ctes = append(c.ctes, "copies"+n+" AS (SELECT "+profile+" AS profile_id,"+id+" AS occurrence_id,"+
		"CASE WHEN "+within+" THEN 1 ELSE 0 END AS in_period,"+
		"CASE WHEN "+within+" THEN ("+qual+") ELSE CAST(NULL AS BOOLEAN) END AS qualification,"+
		q(node.Value.Column.Name)+" AS sum_value FROM "+table+
		" WHERE "+id+" IN (SELECT occurrence_id FROM ids"+n+"))")
	c.ctes = append(c.ctes, "reads"+n+" AS (SELECT profile_id,occurrence_id,in_period,qualification,"+
		"CASE WHEN in_period=1 AND qualification IS DISTINCT FROM FALSE THEN ("+value+") END AS amount FROM copies"+n+")")
	state := "CASE WHEN qualification=TRUE THEN 1 WHEN qualification=FALSE THEN 0 ELSE 2 END"
	known := "CASE WHEN in_period=1 AND qualification IS DISTINCT FROM FALSE THEN " +
		"CASE WHEN amount IS NULL THEN 0 ELSE 1 END END"
	c.ctes = append(c.ctes, "facts"+n+` AS (SELECT occurrence_id,
 MIN(CAST(profile_id AS VARCHAR(36))) AS profile_id,
 COUNT(DISTINCT profile_id)+CASE WHEN COUNT(*)>COUNT(profile_id) THEN 1 ELSE 0 END AS profiles,
 MIN(in_period) AS period_min,MAX(in_period) AS period_max,
 MIN(CASE WHEN in_period=1 THEN `+state+` END) AS state_min,
 MAX(CASE WHEN in_period=1 THEN `+state+` END) AS state_max,
 MIN(`+known+`) AS known_min,MAX(`+known+`) AS known_max,
 MIN(amount) AS amount_min,MAX(amount) AS amount_max
 FROM reads`+n+" GROUP BY occurrence_id)")
	c.ctes = append(c.ctes, "cnt"+n+` AS (SELECT profile_id,COUNT(*) AS occurrences,
 SUM(CASE WHEN state_min=1 THEN 1 ELSE 0 END) AS lo,
 SUM(CASE WHEN state_min=0 THEN 0 ELSE 1 END) AS hi,
 SUM(CASE WHEN state_min<>0 AND known_min=0 THEN 1 ELSE 0 END) AS unknowns,
 CAST(SUM(CASE WHEN state_min=1 OR state_min=2 AND amount_min<0 THEN amount_min ELSE 0 END) AS DECIMAL(38,4)) AS sum_lo,
 CAST(SUM(CASE WHEN state_min=1 OR state_min=2 AND amount_min>0 THEN amount_min ELSE 0 END) AS DECIMAL(38,4)) AS sum_hi
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=period_max
 AND state_min=state_max AND (known_min=known_max OR known_min IS NULL)
 AND (amount_min=amount_max OR amount_min IS NULL) GROUP BY profile_id)`)
	c.ctes = append(c.ctes, "fault"+n+` AS (SELECT
 (SELECT COALESCE(SUM(CASE WHEN profile_id IS NOT NULL AND
 (profiles>1 OR period_min<>period_max OR state_min<>state_max OR known_min<>known_max OR amount_min<>amount_max)
 THEN 1 ELSE 0 END),0) FROM facts`+n+`)
 +(SELECT COUNT(*) FROM cnt`+n+` WHERE occurrences>9223372036854775807)
 +(SELECT COUNT(*) FROM `+table+" WHERE "+within+profileOnly+
		" AND "+profile+" IS NOT NULL AND "+id+" IS NULL) AS failures)")
	c.faults = append(c.faults, "fault"+n)
	alias := "v" + n
	c.joins = append(c.joins, "LEFT JOIN cnt"+n+" "+alias+" ON "+alias+".profile_id=p."+q("$profile_id"))
	lo, hi := sumBounds(alias)
	threshold := fmt.Sprintf("CAST('%.4f' AS DECIMAL(38,4))", node.Threshold)
	test := "CASE WHEN (" + hi + ")<=" + threshold + " THEN TRUE WHEN (" + lo + ")>" + threshold +
		" THEN FALSE ELSE NULL END"
	if node.AtLeast {
		test = "CASE WHEN (" + lo + ")>=" + threshold + " THEN TRUE WHEN (" + hi + ")<" + threshold +
			" THEN FALSE ELSE NULL END"
	}

	return withSampleMinimum(node.Minimum, "COALESCE("+alias+".lo,0)", "COALESCE("+alias+".hi,0)", test), nil
}

// sumBounds returns qualification extrema, or NULL for an unbounded side.
func sumBounds(alias string) (string, string) {
	prefix := "CASE WHEN COALESCE(" + alias + ".unknowns,0)=0 THEN COALESCE("
	return prefix + alias + ".sum_lo,0) END", prefix + alias + ".sum_hi,0) END"
}

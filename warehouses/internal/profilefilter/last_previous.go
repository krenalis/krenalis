// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// lastPrevious closes ordered differences on compatible Last/Previous pairs.
// Separate strictly later and tied groups avoid pair joins and self-pairs.
func (c *compiler) lastPrevious(node *warehouses.LastPreviousDifference) (string, error) {

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
	qual, faultQual := "TRUE", "TRUE"
	if node.Qualification != nil {
		qual, err = c.qualification(node.Qualification)
		if err != nil {
			return "", err
		}
		faultQual, err = c.qualification(node.Qualification)
		if err != nil {
			return "", err
		}
	}
	reading := node.Value
	reading.Column.Name = "pair_value"
	amount, err := c.dialect.ExactDecimal(reading)
	if err != nil {
		return "", err
	}
	c.ctes = append(c.ctes, "bounds"+n+" AS (SELECT "+lower+end+" AS upper_bound)")
	timestamp := q(node.Source.Timestamp.Name)
	within := timestamp + "<(SELECT upper_bound FROM bounds" + n + ")"
	if !node.UnboundedStart {
		within = timestamp + ">=(SELECT lower_bound FROM bounds" + n + ") AND " + within
	}
	table, profile := q(node.Source.Table), q(node.Source.Profile.Name)
	id, err := c.dialect.ExactText(node.Source.Identity)
	if err != nil {
		return "", err
	}
	occurrenceID, err := c.dialect.ExactText(warehouses.Column{Name: "occurrence_id", Type: node.Source.Identity.Type})
	if err != nil {
		return "", err
	}
	profileOnly, identityProfile := "", ""
	if c.profile != "" {
		profileOnly = " AND CAST(" + profile + " AS VARCHAR(36))=" + c.profile
		identityProfile = " AND " + id + " IN (SELECT " + id + " FROM " + table + " WHERE CAST(" +
			profile + " AS VARCHAR(36))=" + c.profile + ")"
	}

	// Inspect all copies of each seeded identity before selecting pairs.
	c.ctes = append(c.ctes, "ids"+n+" AS (SELECT DISTINCT "+id+" AS occurrence_id FROM "+table+
		" WHERE "+within+identityProfile+")")
	c.ctes = append(c.ctes, "copies"+n+" AS (SELECT "+profile+" AS profile_id,"+id+" AS occurrence_id,"+
		timestamp+" AS event_time,CASE WHEN "+within+" THEN 1 ELSE 0 END AS in_period,"+
		"CASE WHEN "+within+" THEN ("+qual+") ELSE CAST(NULL AS BOOLEAN) END AS qualification,"+
		q(node.Value.Column.Name)+" AS pair_value FROM "+table+" WHERE "+id+" IN (SELECT "+occurrenceID+" FROM ids"+n+"))")
	c.ctes = append(c.ctes, "reads"+n+" AS (SELECT profile_id,occurrence_id,event_time,in_period,qualification,"+
		"CASE WHEN in_period=1 AND qualification IS DISTINCT FROM FALSE THEN ("+amount+") END AS amount FROM copies"+n+")")
	state := "CASE WHEN qualification=TRUE THEN 1 WHEN qualification=FALSE THEN 0 ELSE 2 END"
	known := "CASE WHEN in_period=1 AND qualification IS DISTINCT FROM FALSE THEN " +
		"CASE WHEN amount IS NULL THEN 0 ELSE 1 END END"
	c.ctes = append(c.ctes, "facts"+n+` AS (SELECT occurrence_id,
 MIN(CAST(profile_id AS VARCHAR(36))) AS profile_id,
 COUNT(DISTINCT profile_id)+CASE WHEN COUNT(*)>COUNT(profile_id) THEN 1 ELSE 0 END AS profiles,
 MIN(event_time) AS time_min,MAX(event_time) AS time_max,COUNT(*)-COUNT(event_time) AS missing_times,
 MIN(in_period) AS period_min,MAX(in_period) AS period_max,
 MIN(CASE WHEN in_period=1 THEN `+state+` END) AS state_min,
 MAX(CASE WHEN in_period=1 THEN `+state+` END) AS state_max,
 MIN(`+known+`) AS known_min,MAX(`+known+`) AS known_max,
 MIN(amount) AS amount_min,MAX(amount) AS amount_max FROM reads`+n+` GROUP BY occurrence_id)`)
	c.ctes = append(c.ctes, "base"+n+` AS (SELECT profile_id,time_min,state_min,known_min,amount_min AS amount
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=1 AND period_max=1
 AND missing_times=0 AND time_min=time_max AND state_min=state_max AND state_min<>0
 AND known_min=known_max AND (known_min=0 OR amount_min=amount_max))`)

	// Build strict suffixes over one timestamp group per profile.
	c.ctes = append(c.ctes, "timegroups"+n+` AS (SELECT profile_id,time_min,COUNT(*) AS n_t,
 SUM(CASE WHEN state_min=1 THEN 1 ELSE 0 END) AS c_t,
 SUM(CASE WHEN known_min=1 THEN 1 ELSE 0 END) AS k_t,
 SUM(CASE WHEN known_min=0 THEN 1 ELSE 0 END) AS u_t,
 SUM(CASE WHEN state_min=1 AND known_min=0 THEN 1 ELSE 0 END) AS cu_t,
 MIN(amount) AS gmin,MAX(amount) AS gmax,
 MIN(CASE WHEN state_min=1 THEN amount END) AS cmin
 FROM base`+n+` GROUP BY profile_id,time_min)`)
	frame := "PARTITION BY profile_id ORDER BY time_min DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"
	cumulative := "cumulative" + n + " AS (SELECT profile_id,time_min,n_t,k_t,u_t,gmin,gmax,"
	for _, name := range []string{"c", "n", "k", "u", "cu"} {
		cumulative += "SUM(" + name + "_t) OVER (" + frame + ") AS cum_" + name + ","
	}
	cumulative += "MIN(gmin) OVER (" + frame + ") AS cum_min,MAX(gmax) OVER (" + frame + ") AS cum_max," +
		"MIN(cmin) OVER (" + frame + ") AS cum_cmin," +
		"SUM(c_t) OVER (PARTITION BY profile_id) AS total_c," +
		"SUM(n_t) OVER (PARTITION BY profile_id) AS total_n FROM timegroups" + n + ")"
	c.ctes = append(c.ctes, cumulative)
	order := "PARTITION BY profile_id ORDER BY time_min DESC"
	suffix := "suffix" + n + " AS (SELECT profile_id,time_min,n_t,k_t,u_t,gmin,gmax,total_c,total_n,"
	for _, name := range []string{"c", "n", "k", "u", "cu"} {
		suffix += "COALESCE(LAG(cum_" + name + ") OVER (" + order + "),0) AS later_" + name + ","
	}
	suffix += "LAG(cum_min) OVER (" + order + ") AS later_min,LAG(cum_max) OVER (" + order + ") AS later_max," +
		"LAG(cum_cmin) OVER (" + order + ") AS later_cmin FROM cumulative" + n + ")"
	c.ctes = append(c.ctes, suffix)

	// Each family contributes reachable extrema, using distinct identities.
	difference := func(left, right string) string {
		return "CAST(CAST(" + left + " AS DECIMAL(19,4))-CAST(" + right + " AS DECIMAL(19,4)) AS DECIMAL(19,4))"
	}
	var differences []string
	for _, family := range []struct{ condition, lo, hi string }{
		{"later_c=0 AND later_n>0 AND later_k>0 AND k_t>0", difference("later_min", "gmax"), difference("later_max", "gmin")},
		{"later_c=0 AND k_t>=2", difference("gmin", "gmax"), difference("gmax", "gmin")},
		{"later_c=1 AND later_cu=0 AND k_t>0", difference("later_cmin", "gmax"), difference("later_cmin", "gmin")},
	} {
		differences = append(differences, "SELECT profile_id,"+family.lo+" AS lo,"+
			family.hi+" AS hi FROM suffix"+n+" WHERE "+family.condition)
	}
	c.ctes = append(c.ctes, "differences"+n+" AS ("+strings.Join(differences, " UNION ALL ")+")")
	c.ctes = append(c.ctes, "difference_bounds"+n+" AS (SELECT profile_id,MIN(lo) AS lo,MAX(hi) AS hi "+
		"FROM differences"+n+" GROUP BY profile_id)")
	c.ctes = append(c.ctes, "pair_measures"+n+` AS (SELECT profile_id,
 SUM(CASE WHEN total_n>=2 AND later_c<2 THEN n_t ELSE 0 END) AS candidates,
 MAX(CASE WHEN total_c<2 THEN 1 ELSE 0 END) AS absent,
 MAX(CASE WHEN (later_c=0 AND later_n>0 AND (u_t>0 OR later_u>0))
 OR (later_c=0 AND n_t>=2 AND u_t>0) OR (later_c=1 AND (u_t>0 OR later_cu>0))
 THEN 1 ELSE 0 END) AS unknowns,
 MIN(CASE WHEN total_n>=2 AND later_c<2 THEN time_min END) AS first_time,
 MAX(CASE WHEN total_n>=2 AND later_c<2 THEN time_min END) AS last_time
 FROM suffix`+n+` GROUP BY profile_id)`)
	c.ctes = append(c.ctes, "selected"+n+` AS (SELECT m.profile_id,m.candidates,m.absent,m.unknowns,
 m.first_time,m.last_time,b.lo,b.hi FROM pair_measures`+n+` m
 LEFT JOIN difference_bounds`+n+` b ON b.profile_id=m.profile_id)`)

	// Preserve source faults and close the test without assuming dense deltas.
	c.ctes = append(c.ctes, "fault"+n+` AS (SELECT
 (SELECT COALESCE(SUM(CASE WHEN profile_id IS NOT NULL AND
 (profiles>1 OR period_min<>period_max OR missing_times<>0 OR (time_min<>time_max AND state_max<>0)
 OR state_min<>state_max OR known_min<>known_max OR amount_min<>amount_max) THEN 1 ELSE 0 END),0) FROM facts`+n+`)
 +(SELECT COUNT(*) FROM selected`+n+` WHERE candidates>9223372036854775807
 OR (candidates>0 AND unknowns=0 AND (lo IS NULL OR hi IS NULL OR lo>hi)))
 +(SELECT COUNT(*) FROM `+table+" WHERE (("+within+" AND "+q(node.Source.Identity.Name)+" IS NULL) OR ("+
		timestamp+" IS NULL AND ("+faultQual+") IS DISTINCT FROM FALSE)) AND "+profile+
		" IS NOT NULL"+profileOnly+") AS failures)")
	c.faults = append(c.faults, "fault"+n)
	alias := "v" + n
	c.joins = append(c.joins, "LEFT JOIN selected"+n+" "+alias+" ON "+alias+".profile_id=p."+q("$profile_id"))
	prefix := "CASE WHEN COALESCE(" + alias + ".candidates,0)=0 THEN FALSE WHEN " + alias +
		".unknowns>0 THEN CAST(NULL AS BOOLEAN)"
	if node.ReadableDifferences == "none" {
		return prefix + " ELSE FALSE END", nil
	}
	if node.ReadableDifferences == "all" {
		return prefix + " WHEN " + alias + ".absent=0 THEN TRUE ELSE CAST(NULL AS BOOLEAN) END", nil
	}
	threshold := fmt.Sprintf("CAST('%.4f' AS DECIMAL(19,4))", *node.Threshold)
	if node.AtLeast {
		return prefix + " WHEN " + alias + ".hi<" + threshold + " THEN FALSE WHEN " + alias + ".absent=0 AND " +
			alias + ".lo>=" + threshold + " THEN TRUE ELSE CAST(NULL AS BOOLEAN) END", nil
	}

	return prefix + " WHEN " + alias + ".lo>" + threshold + " THEN FALSE WHEN " + alias + ".absent=0 AND " +
		alias + ".hi<=" + threshold + " THEN TRUE ELSE CAST(NULL AS BOOLEAN) END", nil
}

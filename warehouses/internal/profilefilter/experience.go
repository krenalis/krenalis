// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"strconv"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

func (c *compiler) experience(node *warehouses.SelectedOccurrence) (string, error) {
	return c.selection(node, 0, nil)
}

func (c *compiler) latestN(node *warehouses.LatestOccurrences) (string, error) {
	base := &warehouses.SelectedOccurrence{Source: node.Source, Start: node.Start, End: node.End,
		UnboundedStart: node.UnboundedStart, Qualification: node.Qualification, Verify: node.Predicate}
	return c.selection(base, node.Size, node.Accepted)
}

// selection preserves compatible occurrences or complete LatestN blocks.
// First/Last use the earliest/latest certain timestamp to exclude later/older
// optional occurrences; ties remain candidates. Nth preserves possible ranks.
// A positive size selects LatestN blocks. No ID tie-break or greedy choice
// closes any selection.
func (c *compiler) selection(node *warehouses.SelectedOccurrence, size int, accepted *warehouses.CountRange) (string, error) {

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
	verify, err := c.qualificationGiven(node.Verify, node.Qualification)
	if err != nil {
		return "", err
	}
	// Render the repeated qualification again for positional parameter binds.
	faultQual := "TRUE"
	if node.Qualification != nil {
		faultQual, err = c.qualification(node.Qualification)
		if err != nil {
			return "", err
		}
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
		identityProfile = " AND " + id + " IN (SELECT " + id + " FROM " + table +
			" WHERE CAST(" + profile + " AS VARCHAR(36))=" + c.profile + ")"
	}

	// Seed by period, but inspect all copies of each identity before selection.
	// Even two timestamps within the same period must not be merged silently.
	c.ctes = append(c.ctes, "ids"+n+" AS (SELECT DISTINCT "+id+" AS occurrence_id FROM "+table+
		" WHERE "+within+identityProfile+")")
	c.ctes = append(c.ctes, "copies"+n+" AS (SELECT "+profile+" AS profile_id,"+id+" AS occurrence_id,"+
		timestamp+" AS event_time,CASE WHEN "+within+" THEN 1 ELSE 0 END AS in_period,"+
		"CASE WHEN "+within+" THEN ("+qual+") ELSE CAST(NULL AS BOOLEAN) END AS qualification,"+
		"CASE WHEN "+within+" THEN ("+verify+") ELSE CAST(NULL AS BOOLEAN) END AS verification FROM "+table+
		" WHERE "+id+" IN (SELECT "+occurrenceID+" FROM ids"+n+"))")
	extreme, direction := "MAX", ">="
	if node.First {
		extreme, direction = "MIN", "<="
	}
	state := "CASE WHEN qualification=TRUE THEN 1 WHEN qualification=FALSE THEN 0 ELSE 2 END"
	body := "CASE WHEN verification=TRUE THEN 1 WHEN verification=FALSE THEN 0 ELSE 2 END"
	c.ctes = append(c.ctes, "facts"+n+` AS (SELECT occurrence_id,
 MIN(CAST(profile_id AS VARCHAR(36))) AS profile_id,
 COUNT(DISTINCT profile_id)+CASE WHEN COUNT(*)>COUNT(profile_id) THEN 1 ELSE 0 END AS profiles,
 MIN(event_time) AS time_min,MAX(event_time) AS time_max,COUNT(*)-COUNT(event_time) AS missing_times,
 MIN(in_period) AS period_min,MAX(in_period) AS period_max,
 MIN(CASE WHEN in_period=1 THEN `+state+` END) AS state_min,
 MAX(CASE WHEN in_period=1 THEN `+state+` END) AS state_max,
 MIN(CASE WHEN in_period=1 AND (qualification IS DISTINCT FROM FALSE) THEN `+body+` END) AS verify_min,
 MAX(CASE WHEN in_period=1 AND (qualification IS DISTINCT FROM FALSE) THEN `+body+` END) AS verify_max
 FROM copies`+n+` GROUP BY occurrence_id)`)
	if size != 0 {
		if accepted == nil {
			c.latestBlocks(n, size)
		} else {
			c.latestBlockCounts(n, size)
		}
	} else if node.Nth != 0 {
		position, before := strconv.Itoa(node.Nth), strconv.Itoa(node.Nth-1)
		c.ctes = append(c.ctes, "base"+n+` AS (SELECT profile_id,time_min,state_min,verify_min
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=1 AND period_max=1
 AND missing_times=0 AND time_min=time_max AND state_min=state_max AND state_min<>0
 AND verify_min=verify_max)`)
		// Group ties before cumulative counts; each group has one timestamp.
		c.ctes = append(c.ctes, "timegroups"+n+` AS (SELECT profile_id,time_min,
 SUM(CASE WHEN state_min=1 THEN 1 ELSE 0 END) AS c_t,COUNT(*) AS n_t
 FROM base`+n+` GROUP BY profile_id,time_min)`)
		frame := "PARTITION BY profile_id ORDER BY time_min ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"
		c.ctes = append(c.ctes, "ranks"+n+` AS (SELECT profile_id,time_min,c_t,
 SUM(c_t) OVER (`+frame+`) AS cum_c,SUM(n_t) OVER (`+frame+`) AS cum_n,
 SUM(c_t) OVER (PARTITION BY profile_id) AS total_c FROM timegroups`+n+`)`)
		c.ctes = append(c.ctes, "selected"+n+` AS (SELECT b.profile_id,COUNT(*) AS candidates,
 MAX(CASE WHEN r.total_c<`+position+` THEN 1 ELSE 0 END) AS absent,
 SUM(CASE WHEN b.verify_min=1 THEN 1 ELSE 0 END) AS passed,
 SUM(CASE WHEN b.verify_min=0 THEN 1 ELSE 0 END) AS failed
 FROM base`+n+` b JOIN ranks`+n+` r ON r.profile_id=b.profile_id AND r.time_min=b.time_min
 WHERE r.cum_c-r.c_t<=`+before+` AND r.cum_n-1>=`+before+` GROUP BY b.profile_id)`)
	} else {
		c.ctes = append(c.ctes, "base"+n+` AS (SELECT profile_id,time_min,state_min,verify_min,
 `+extreme+`(CASE WHEN state_min=1 THEN time_min END) OVER (PARTITION BY profile_id) AS certain_time
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=1 AND period_max=1
 AND missing_times=0 AND time_min=time_max AND state_min=state_max AND state_min<>0
 AND verify_min=verify_max)`)
		c.ctes = append(c.ctes, "selected"+n+` AS (SELECT profile_id,COUNT(*) AS candidates,
 MAX(CASE WHEN certain_time IS NULL THEN 1 ELSE 0 END) AS absent,
 SUM(CASE WHEN verify_min=1 THEN 1 ELSE 0 END) AS passed,
 SUM(CASE WHEN verify_min=0 THEN 1 ELSE 0 END) AS failed
 FROM base`+n+` WHERE certain_time IS NULL OR time_min`+direction+`certain_time GROUP BY profile_id)`)
	}
	c.ctes = append(c.ctes, "fault"+n+` AS (SELECT
 (SELECT COALESCE(SUM(CASE WHEN profile_id IS NOT NULL AND
 (profiles>1 OR period_min<>period_max OR missing_times<>0 OR (time_min<>time_max AND state_max<>0)
 OR state_min<>state_max OR verify_min<>verify_max) THEN 1 ELSE 0 END),0) FROM facts`+n+`)
 +(SELECT COUNT(*) FROM selected`+n+` WHERE candidates>9223372036854775807)
 +(SELECT COUNT(*) FROM `+table+" WHERE (("+within+" AND "+q(node.Source.Identity.Name)+" IS NULL) OR ("+
		timestamp+" IS NULL AND ("+faultQual+") IS DISTINCT FROM FALSE)) AND "+profile+
		" IS NOT NULL"+profileOnly+") AS failures)")
	c.faults = append(c.faults, "fault"+n)
	alias := "v" + n
	c.joins = append(c.joins, "LEFT JOIN selected"+n+" "+alias+" ON "+alias+".profile_id=p."+q("$profile_id"))
	candidates, absent := "COALESCE("+alias+".candidates,0)", "COALESCE("+alias+".absent,1)"

	if accepted != nil {
		lower, upper := strconv.FormatInt(accepted.Lower, 10), strconv.FormatInt(*accepted.Upper, 10)
		return "CASE WHEN " + candidates + "=0 OR " + alias + ".max_count<" + lower +
			" OR " + alias + ".min_count>" + upper + " THEN FALSE WHEN " + absent + "=0 AND " +
			alias + ".min_count>=" + lower + " AND " + alias + ".max_count<=" + upper +
			" THEN TRUE ELSE CAST(NULL AS BOOLEAN) END", nil
	}

	if size != 0 {
		return "CASE WHEN " + absent + "=0 AND COALESCE(" + alias + ".passed,0)=" + candidates +
			" THEN TRUE WHEN COALESCE(" + alias + ".reachable,0)=0 THEN FALSE ELSE CAST(NULL AS BOOLEAN) END", nil
	}

	return "CASE WHEN " + absent + "=0 AND COALESCE(" + alias + ".passed,0)=" + candidates +
		" THEN TRUE WHEN COALESCE(" + alias + ".failed,0)=" + candidates +
		" THEN FALSE ELSE CAST(NULL AS BOOLEAN) END", nil
}

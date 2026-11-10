// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import "strconv"

// latestBlockCounts computes exact global extrema for complete blocks.
// The independent-membership model makes every intermediate count reachable.
// Frontier relaxation may choose a newer boundary; global extrema stay exact.
func (c *compiler) latestBlockCounts(n string, size int) {
	count := strconv.Itoa(size)
	c.ctes = append(c.ctes, "base"+n+` AS (SELECT profile_id,time_min,state_min,verify_min
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=1 AND period_max=1
 AND missing_times=0 AND time_min=time_max AND state_min=state_max AND state_min<>0
 AND verify_min=verify_max)`)
	c.ctes = append(c.ctes, "timegroups"+n+` AS (SELECT profile_id,time_min,COUNT(*) AS n_t,
 SUM(CASE WHEN state_min=1 THEN 1 ELSE 0 END) AS c_t,
 SUM(CASE WHEN state_min=1 AND verify_min=1 THEN 1 ELSE 0 END) AS c_true_t,
 SUM(CASE WHEN state_min=1 AND verify_min=2 THEN 1 ELSE 0 END) AS c_unknown_t,
 SUM(CASE WHEN state_min=2 THEN 1 ELSE 0 END) AS o_t,
 SUM(CASE WHEN state_min=2 AND verify_min<>1 THEN 1 ELSE 0 END) AS o_nontrue_t,
 SUM(CASE WHEN state_min=2 AND verify_min<>0 THEN 1 ELSE 0 END) AS o_nonfalse_t,
 SUM(CASE WHEN verify_min<>1 THEN 1 ELSE 0 END) AS nontrue_t,
 SUM(CASE WHEN verify_min<>0 THEN 1 ELSE 0 END) AS nonfalse_t
 FROM base`+n+` GROUP BY profile_id,time_min)`)
	frame := "PARTITION BY profile_id ORDER BY time_min DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"
	c.ctes = append(c.ctes, "ranks"+n+` AS (SELECT profile_id,time_min,n_t,nontrue_t,nonfalse_t,
 SUM(c_t) OVER (`+frame+`) - c_t AS later_c,
 SUM(c_true_t) OVER (`+frame+`) - c_true_t AS later_true,
 SUM(c_unknown_t) OVER (`+frame+`) - c_unknown_t AS later_unknown,
 SUM(o_t) OVER (`+frame+`) - o_t AS later_o,
 SUM(o_nontrue_t) OVER (`+frame+`) - o_nontrue_t AS later_nontrue,
 SUM(o_nonfalse_t) OVER (`+frame+`) - o_nonfalse_t AS later_nonfalse,
 SUM(c_t) OVER (PARTITION BY profile_id) AS total_c,
 SUM(n_t) OVER (PARTITION BY profile_id) AS total_n FROM timegroups`+n+`)`)
	// Inclusive frames over nonempty timestamp groups keep these operands known.
	// CASE avoids dialect-dependent NULL behavior in GREATEST/LEAST.
	slots := "(" + count + "-r.later_c)"
	nontrue, nonfalse := "(r.later_nontrue+r.nontrue_t)", "(r.later_nonfalse+r.nonfalse_t)"
	eligible := slots + ">=1 AND " + slots + "<=r.later_o+r.n_t"
	lower := "r.later_true+CASE WHEN " + slots + ">" + nontrue + " THEN " + slots + "-" + nontrue + " ELSE 0 END"
	upper := "r.later_true+r.later_unknown+CASE WHEN " + slots + "<" + nonfalse +
		" THEN " + slots + " ELSE " + nonfalse + " END"
	c.ctes = append(c.ctes, "selected"+n+` AS (SELECT b.profile_id,
 SUM(CASE WHEN r.total_n>=`+count+` AND r.later_c<`+count+` THEN 1 ELSE 0 END) AS candidates,
 MAX(CASE WHEN r.total_c<`+count+` THEN 1 ELSE 0 END) AS absent,
 MIN(CASE WHEN `+eligible+` THEN `+lower+` END) AS min_count,
 MAX(CASE WHEN `+eligible+` THEN `+upper+` END) AS max_count
 FROM base`+n+` b JOIN ranks`+n+` r ON r.profile_id=b.profile_id AND r.time_min=b.time_min
 GROUP BY b.profile_id)`)
}

// latestBlocks closes All on compatible blocks, not separate position winners.
// Unknown local bodies can pass in a completion. A boundary timestamp permits
// a passing block only if every later certain identity can pass, fewer than N
// are forced later, and enough passable identities fill the remaining slots.
func (c *compiler) latestBlocks(n string, size int) {
	count := strconv.Itoa(size)
	c.ctes = append(c.ctes, "base"+n+` AS (SELECT profile_id,time_min,state_min,verify_min
 FROM facts`+n+` WHERE profile_id IS NOT NULL AND profiles=1 AND period_min=1 AND period_max=1
 AND missing_times=0 AND time_min=time_max AND state_min=state_max AND state_min<>0
 AND verify_min=verify_max)`)
	c.ctes = append(c.ctes, "timegroups"+n+` AS (SELECT profile_id,time_min,COUNT(*) AS n_t,
 SUM(CASE WHEN state_min=1 THEN 1 ELSE 0 END) AS c_t,
 SUM(CASE WHEN verify_min<>0 THEN 1 ELSE 0 END) AS good_t,
 SUM(CASE WHEN state_min=2 AND verify_min<>0 THEN 1 ELSE 0 END) AS optional_good_t,
 SUM(CASE WHEN state_min=1 AND verify_min=0 THEN 1 ELSE 0 END) AS bad_c_t
 FROM base`+n+` GROUP BY profile_id,time_min)`)
	frame := "PARTITION BY profile_id ORDER BY time_min DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"
	c.ctes = append(c.ctes, "ranks"+n+` AS (SELECT profile_id,time_min,good_t,
 SUM(c_t) OVER (`+frame+`) - c_t AS later_c,
 SUM(optional_good_t) OVER (`+frame+`) - optional_good_t AS later_optional_good,
 SUM(bad_c_t) OVER (`+frame+`) - bad_c_t AS later_bad_c,
 SUM(c_t) OVER (PARTITION BY profile_id) AS total_c,
 SUM(n_t) OVER (PARTITION BY profile_id) AS total_n FROM timegroups`+n+`)`)
	c.ctes = append(c.ctes, "selected"+n+` AS (SELECT b.profile_id,
 SUM(CASE WHEN r.total_n>=`+count+` AND r.later_c<`+count+` THEN 1 ELSE 0 END) AS candidates,
 MAX(CASE WHEN r.total_c<`+count+` THEN 1 ELSE 0 END) AS absent,
 SUM(CASE WHEN r.total_n>=`+count+` AND r.later_c<`+count+` AND b.verify_min=1 THEN 1 ELSE 0 END) AS passed,
 MAX(CASE WHEN r.good_t>0 AND r.later_bad_c=0 AND r.later_c<`+count+`
 AND r.later_c+r.later_optional_good+r.good_t>=`+count+` THEN 1 ELSE 0 END) AS reachable
 FROM base`+n+` b JOIN ranks`+n+` r ON r.profile_id=b.profile_id AND r.time_min=b.time_min
 GROUP BY b.profile_id)`)
}

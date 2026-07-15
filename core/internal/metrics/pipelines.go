// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
)

const (
	Minute = time.Minute
	Hour   = time.Hour
	Day    = 24 * time.Hour
	Month  = 30 * 24 * time.Hour
)

// Error represents an error that occurred while executing a pipeline.
type Error struct {
	Pipeline     string    `json:"pipeline"`
	Step         Step      `json:"step"`
	Count        int       `json:"count"`
	Message      string    `json:"message"`
	LastOccurred time.Time `json:"lastOccurred"`
}

// Metrics represents the metrics for a time period.
type Metrics struct {
	Start  time.Time
	End    time.Time
	Series []Series
}

// Series represents metrics for a single grouping.
type Series struct {
	Workspace  string
	Connection string
	Pipeline   string
	Passed     [][6]int
	Failed     [][6]int
}

// Selection describes which metric series are returned.
type Selection struct {
	Workspaces  []string
	Connections []string
	Pipelines   []string
	Target      Target
}

// Target represents a target.
type Target int

const (
	TargetNone = iota
	TargetEvent
	TargetUser
	TargetGroup
)

func (t Target) String() string {
	switch t {
	case TargetEvent:
		return "Event"
	case TargetUser:
		return "User"
	case TargetGroup:
		return "Group"
	}
	panic("core/internal/metrics: invalid Target")
}

// Errors returns the errors for the provided pipelines within the time range
// [start,end). The end time must not precede the start time, and both must be
// within [MinTime,MaxTime]. pipelines must not be empty. Returned errors are
// limited to [first, first+limit), where first >= 0 and 0 < limit <= 100.
func (c *Collector) Errors(ctx context.Context, start, end time.Time, pipelines []string, step *Step, first, limit int) ([]Error, error) {

	tsStart := TimeSlotFromTime(start)
	tsEnd := TimeSlotFromTime(end) - 1

	query := bytes.NewBufferString("SELECT pipeline, MAX(timeslot) AS timeslot, step, sum(count), message\n" +
		"FROM pipelines_errors\nWHERE ")

	query.WriteString("timeslot BETWEEN ")
	query.WriteString(strconv.Itoa(int(tsStart)))
	query.WriteString(" AND ")
	query.WriteString(strconv.Itoa(int(tsEnd)))
	query.WriteString(" AND pipeline IN (")
	for i, pipeline := range pipelines {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(db.Quote(pipeline))
	}
	query.WriteByte(')')

	if step != nil {
		query.WriteString(" AND step = ")
		s := *step
		query.WriteString(strconv.Itoa(int(s)))
	}

	query.WriteString("\nGROUP BY pipeline, step, message\nORDER BY timeslot DESC, pipeline, message\nLIMIT $1\nOFFSET $2")

	rows, err := c.db.Query(ctx, query.String(), limit, first)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	i := 0
	errs := make([]Error, limit)
	var ts int32
	for rows.Next() {
		if err = rows.Scan(&errs[i].Pipeline, &ts, &errs[i].Step, &errs[i].Count, &errs[i].Message); err != nil {
			return nil, err
		}
		if ts < 0 || ts > maxTimeslot {
			return nil, fmt.Errorf("pipelines_errors table contains a timeslot that is out of range")
		}
		errs[i].LastOccurred = TimeSlotToTime(ts)
		i++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return errs[:i], nil
}

// MetricsPerDate returns metrics aggregated by day for the time interval
// between the specified start and end dates.
func (c *Collector) MetricsPerDate(ctx context.Context, start, end time.Time, selection Selection) (Metrics, error) {
	return c.queryMetrics(ctx, start, end, Day, selection)
}

// MetricsPerTimeUnit returns metrics for the specified number of minutes,
// hours, or days based on the unit up to the current time.
func (c *Collector) MetricsPerTimeUnit(ctx context.Context, number int, unit time.Duration, selection Selection) (Metrics, error) {
	now := time.Now().UTC()
	end := now.Truncate(unit).Add(unit)
	start := end.Add(-time.Duration(number) * unit)
	return c.queryMetrics(ctx, start, end, unit, selection)
}

func (c *Collector) queryMetrics(ctx context.Context, start, end time.Time, resolution time.Duration, selection Selection) (Metrics, error) {

	if !end.After(start) {
		return Metrics{}, fmt.Errorf("metrics end must be after start")
	}
	if resolution < time.Minute || resolution%time.Minute != 0 {
		return Metrics{}, fmt.Errorf("metrics resolution must be a positive multiple of one minute")
	}
	if end.Sub(start)%resolution != 0 {
		return Metrics{}, fmt.Errorf("metrics interval must be a multiple of the resolution")
	}
	group, ids, err := selection.group()
	if err != nil {
		return Metrics{}, err
	}

	number := int(end.Sub(start) / resolution)

	metrics := Metrics{
		Start: start,
		End:   end,
	}
	if len(ids) == 0 {
		return metrics, nil
	}

	divisor := int32(resolution / time.Minute)
	tsStart := TimeSlotFromTime(metrics.Start)
	tsEnd := TimeSlotFromTime(metrics.End) - 1

	sql := bytes.NewBufferString("SELECT ")
	switch group {
	case groupWorkspace:
		sql.WriteString("workspace, ")
	case groupConnection:
		sql.WriteString("connection, ")
	case groupPipeline:
		sql.WriteString("pipeline, ")
	}
	sql.WriteString("timeslot/$1 AS slot, SUM(passed_0), SUM(passed_1), SUM(passed_2), SUM(passed_3), SUM(passed_4), SUM(passed_5)," +
		" SUM(failed_0), SUM(failed_1), SUM(failed_2), SUM(failed_3), SUM(failed_4), SUM(failed_5)\n" +
		"FROM pipelines_metrics\nWHERE timeslot BETWEEN $2 AND $3")
	switch group {
	case groupWorkspace:
		sql.WriteString(" AND workspace IN (")
		writeQuotedList(sql, ids)
		sql.WriteByte(')')
	case groupConnection:
		sql.WriteString(" AND connection IN (")
		writeQuotedList(sql, ids)
		sql.WriteByte(')')
	case groupPipeline:
		sql.WriteString(" AND pipeline IN (")
		writeQuotedList(sql, ids)
		sql.WriteByte(')')
	}
	if selection.Target != TargetNone {
		sql.WriteString(" AND target = ")
		sql.WriteString(db.Quote(selection.Target.String()))
	}
	sql.WriteString("\nGROUP BY ")
	switch group {
	case groupWorkspace:
		sql.WriteString("workspace, slot\nORDER BY workspace, slot")
	case groupConnection:
		sql.WriteString("connection, slot\nORDER BY connection, slot")
	case groupPipeline:
		sql.WriteString("pipeline, slot\nORDER BY pipeline, slot")
	}

	rows, err := c.db.Query(ctx, sql.String(), divisor, tsStart, tsEnd)
	if err != nil {
		return Metrics{}, err
	}
	defer rows.Close()

	seriesByID := map[string]int{}
	for _, id := range ids {
		seriesByID[id] = len(metrics.Series)
		metrics.Series = append(metrics.Series, newMetricSeries(group, id, number))
	}
	var slot int32
	var id string
	var passed, failed [6]int
	for rows.Next() {
		if err = rows.Scan(&id, &slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
			&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5]); err != nil {
			return Metrics{}, err
		}
		i := int(slot - tsStart/divisor)
		if i < 0 || i >= number {
			return Metrics{}, fmt.Errorf("pipelines_metrics table contains a timeslot that is out of range")
		}
		seriesIndex, ok := seriesByID[id]
		if !ok {
			seriesIndex = len(metrics.Series)
			seriesByID[id] = seriesIndex
			metrics.Series = append(metrics.Series, newMetricSeries(group, id, number))
		}
		metrics.Series[seriesIndex].Passed[i] = passed
		metrics.Series[seriesIndex].Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return Metrics{}, err
	}

	return metrics, nil
}

type metricGroup string

const (
	groupWorkspace  metricGroup = "workspace"
	groupConnection metricGroup = "connection"
	groupPipeline   metricGroup = "pipeline"
)

func (selection Selection) group() (metricGroup, []string, error) {
	groups := 0
	var group metricGroup
	var ids []string
	if selection.Workspaces != nil {
		groups++
		group = groupWorkspace
		ids = selection.Workspaces
	}
	if selection.Connections != nil {
		groups++
		group = groupConnection
		ids = selection.Connections
	}
	if selection.Pipelines != nil {
		groups++
		group = groupPipeline
		ids = selection.Pipelines
	}
	if groups != 1 {
		return "", nil, fmt.Errorf("metrics selection must contain exactly one group")
	}
	return group, ids, nil
}

func newMetricSeries(group metricGroup, id string, number int) Series {
	series := Series{
		Passed: make([][6]int, number),
		Failed: make([][6]int, number),
	}
	switch group {
	case groupWorkspace:
		series.Workspace = id
	case groupConnection:
		series.Connection = id
	case groupPipeline:
		series.Pipeline = id
	}
	return series
}

func writeQuotedList(buf *bytes.Buffer, values []string) {
	for i, value := range values {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(db.Quote(value))
	}
}

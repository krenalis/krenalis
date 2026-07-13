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
	Series []MetricSeries
}

// MetricSeries represents metrics for a single grouping.
type MetricSeries struct {
	Workspace  string
	Connection string
	Pipeline   string
	Passed     [][6]int
	Failed     [][6]int
}

// Query describes the metrics interval, filters and grouping.
type Query struct {
	// Time range and bucket size.
	Start      time.Time
	End        time.Time
	Resolution time.Duration

	// Scope.
	Organization string
	Filter       Filter

	// Output grouping.
	GroupBy Group
}

// Filter describes which metrics are included in a query.
type Filter struct {
	Workspaces  []string
	Connections []string
	Pipelines   []string
	Target      string
}

// Group describes the dimension used to group metrics.
type Group string

const (
	GroupByOrganization Group = "Organization"
	GroupByWorkspace    Group = "Workspace"
	GroupByConnection   Group = "Connection"
	GroupByPipeline     Group = "Pipeline"
)

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

// Metrics returns metrics grouped by the query for the requested time interval.
func (c *Collector) Metrics(ctx context.Context, query Query) (Metrics, error) {

	if !query.End.After(query.Start) {
		return Metrics{}, fmt.Errorf("metrics end must be after start")
	}
	if query.Resolution < time.Minute || query.Resolution%time.Minute != 0 {
		return Metrics{}, fmt.Errorf("metrics resolution must be a positive multiple of one minute")
	}
	if query.End.Sub(query.Start)%query.Resolution != 0 {
		return Metrics{}, fmt.Errorf("metrics interval must be a multiple of the resolution")
	}
	switch query.GroupBy {
	case GroupByOrganization, GroupByWorkspace, GroupByConnection, GroupByPipeline:
	default:
		return Metrics{}, fmt.Errorf("metrics group %q is not valid", query.GroupBy)
	}

	number := int(query.End.Sub(query.Start) / query.Resolution)

	metrics := Metrics{
		Start: query.Start,
		End:   query.End,
	}

	divisor := int32(query.Resolution / time.Minute)
	tsStart := TimeSlotFromTime(metrics.Start)
	tsEnd := TimeSlotFromTime(metrics.End) - 1

	sql := bytes.NewBufferString("SELECT ")
	switch query.GroupBy {
	case GroupByWorkspace:
		sql.WriteString("workspace, ")
	case GroupByConnection:
		sql.WriteString("connection, ")
	case GroupByPipeline:
		sql.WriteString("pipeline, ")
	}
	sql.WriteString("timeslot/$1 AS slot, SUM(passed_0), SUM(passed_1), SUM(passed_2), SUM(passed_3), SUM(passed_4), SUM(passed_5)," +
		" SUM(failed_0), SUM(failed_1), SUM(failed_2), SUM(failed_3), SUM(failed_4), SUM(failed_5)\n" +
		"FROM pipelines_metrics\nWHERE organization = ")
	sql.WriteString(db.Quote(query.Organization))
	sql.WriteString(" AND timeslot BETWEEN $2 AND $3")
	if len(query.Filter.Workspaces) > 0 {
		sql.WriteString(" AND workspace IN (")
		writeQuotedList(sql, query.Filter.Workspaces)
		sql.WriteByte(')')
	}
	if len(query.Filter.Pipelines) > 0 {
		sql.WriteString(" AND pipeline IN (")
		writeQuotedList(sql, query.Filter.Pipelines)
		sql.WriteByte(')')
	}
	if len(query.Filter.Connections) > 0 {
		sql.WriteString(" AND connection IN (")
		writeQuotedList(sql, query.Filter.Connections)
		sql.WriteByte(')')
	}
	if query.Filter.Target != "" {
		sql.WriteString(" AND target = ")
		sql.WriteString(db.Quote(query.Filter.Target))
	}
	sql.WriteString("\nGROUP BY ")
	switch query.GroupBy {
	case GroupByWorkspace:
		sql.WriteString("workspace, slot\nORDER BY workspace, slot")
	case GroupByConnection:
		sql.WriteString("connection, slot\nORDER BY connection, slot")
	case GroupByPipeline:
		sql.WriteString("pipeline, slot\nORDER BY pipeline, slot")
	default:
		sql.WriteString("slot\nORDER BY slot")
	}

	rows, err := c.db.Query(ctx, sql.String(), divisor, tsStart, tsEnd)
	if err != nil {
		return Metrics{}, err
	}
	defer rows.Close()

	var seriesByID map[string]int
	if query.GroupBy == GroupByOrganization {
		metrics.Series = []MetricSeries{{
			Passed: make([][6]int, number),
			Failed: make([][6]int, number),
		}}
	} else {
		seriesByID = map[string]int{}
		for _, id := range query.groupIDs() {
			seriesByID[id] = len(metrics.Series)
			metrics.Series = append(metrics.Series, newMetricSeries(query.GroupBy, id, number))
		}
	}
	var slot int32
	var id string
	var passed, failed [6]int
	for rows.Next() {
		var err error
		switch query.GroupBy {
		case GroupByWorkspace, GroupByConnection, GroupByPipeline:
			err = rows.Scan(&id, &slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
				&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5])
		default:
			err = rows.Scan(&slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
				&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5])
		}
		if err != nil {
			return Metrics{}, err
		}
		i := int(slot - tsStart/divisor)
		if i < 0 || i >= number {
			return Metrics{}, fmt.Errorf("pipelines_metrics table contains a timeslot that is out of range")
		}
		seriesIndex := 0
		if query.GroupBy != GroupByOrganization {
			var ok bool
			seriesIndex, ok = seriesByID[id]
			if !ok {
				seriesIndex = len(metrics.Series)
				seriesByID[id] = seriesIndex
				metrics.Series = append(metrics.Series, newMetricSeries(query.GroupBy, id, number))
			}
		}
		metrics.Series[seriesIndex].Passed[i] = passed
		metrics.Series[seriesIndex].Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return Metrics{}, err
	}

	return metrics, nil
}

func (query Query) groupIDs() []string {
	switch query.GroupBy {
	case GroupByWorkspace:
		return query.Filter.Workspaces
	case GroupByConnection:
		return query.Filter.Connections
	case GroupByPipeline:
		return query.Filter.Pipelines
	default:
		return nil
	}
}

func newMetricSeries(group Group, id string, number int) MetricSeries {
	series := MetricSeries{
		Passed: make([][6]int, number),
		Failed: make([][6]int, number),
	}
	switch group {
	case GroupByWorkspace:
		series.Workspace = id
	case GroupByConnection:
		series.Connection = id
	case GroupByPipeline:
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

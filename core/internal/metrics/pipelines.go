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
)

const (
	Minute = time.Minute
	Hour   = time.Hour
	Day    = 24 * time.Hour
	Month  = 30 * 24 * time.Hour
)

// Error represents an error that occurred while executing a pipeline.
type Error struct {
	Pipeline     int       `json:"pipeline"`
	Step         Step      `json:"step"`
	Count        int       `json:"count"`
	Message      string    `json:"message"`
	LastOccurred time.Time `json:"lastOccurred"`
}

// Metrics represents the metrics for a time period.
type Metrics struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Passed [][6]int  `json:"passed"`
	Failed [][6]int  `json:"failed"`
}

// Errors returns the errors for the provided pipelines within the time range
// [start,end). The end time must not precede the start time, and both must be
// within [MinTime,MaxTime]. pipelines must not be empty. Returned errors are
// limited to [first, first+limit), where first >= 0 and 0 < limit <= 100.
func (c *Collector) Errors(ctx context.Context, start, end time.Time, pipelines []int, step *Step, first, limit int) ([]Error, error) {

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
		query.WriteString(strconv.Itoa(pipeline))
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
// between the specified start and end dates. Both dates must be within the
// range [MinTime,MaxTime], and the day of the start date must be at least one
// day before the day of the end date. pipelines specifies the pipelines for
// which metrics are returned and cannot be empty.
func (c *Collector) MetricsPerDate(ctx context.Context, start, end time.Time, pipelines []int) (Metrics, error) {

	number := int(end.Sub(start).Hours() / 24)

	metrics := Metrics{
		Start:  start,
		End:    end,
		Passed: make([][6]int, number),
		Failed: make([][6]int, number),
	}

	tsStart := TimeSlotFromTime(start)
	tsEnd := TimeSlotFromTime(end) - 1

	query := bytes.NewBufferString("SELECT timeslot/(24*60) AS day, SUM(passed_0), SUM(passed_1), SUM(passed_2), SUM(passed_3), SUM(passed_4), SUM(passed_5)," +
		" SUM(failed_0), SUM(failed_1), SUM(failed_2), SUM(failed_3), SUM(failed_4), SUM(failed_5)\n" +
		"FROM pipelines_metrics\nWHERE timeslot BETWEEN $1 AND $2 AND pipeline IN (")
	for i, pipeline := range pipelines {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(strconv.Itoa(pipeline))
	}
	query.WriteString(")\nGROUP BY day\nORDER BY day")

	rows, err := c.db.Query(ctx, query.String(), tsStart, tsEnd)
	if err != nil {
		return Metrics{}, err
	}
	defer rows.Close()

	var slot int32
	var passed, failed [6]int
	for rows.Next() {
		if err = rows.Scan(&slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
			&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5]); err != nil {
			return Metrics{}, err
		}
		i := int(slot - tsStart/(24*60))
		if i < 0 || i >= number {
			return Metrics{}, fmt.Errorf("pipelines_metrics table contains a timeslot that is out of range")
		}
		metrics.Passed[i] = passed
		metrics.Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return Metrics{}, err
	}

	return metrics, nil
}

// MetricsPerTimeUnit returns metrics for the specified number of minutes,
// hours, or days based on the unit, which can be Minute, Hour, or Day, up to
// the current time. number must be in the following ranges: [1,60] for minutes,
// [1,48] for hours, and [1,30] for days. pipelines represents the pipelines for
// which metrics are returned and cannot be empty.
func (c *Collector) MetricsPerTimeUnit(ctx context.Context, number int, unit time.Duration, pipelines []int) (Metrics, error) {

	now := time.Now().UTC()
	end := now.Truncate(unit).Add(unit)

	metrics := Metrics{
		Start:  end.Add(-time.Duration(number) * unit),
		End:    end,
		Passed: make([][6]int, number),
		Failed: make([][6]int, number),
	}

	divisor := int32(unit / time.Minute)
	tsStart := TimeSlotFromTime(metrics.Start)
	tsEnd := TimeSlotFromTime(metrics.End) - 1

	query := bytes.NewBufferString("SELECT timeslot/$1 AS slot, SUM(passed_0), SUM(passed_1), SUM(passed_2), SUM(passed_3), SUM(passed_4), SUM(passed_5)," +
		" SUM(failed_0), SUM(failed_1), SUM(failed_2), SUM(failed_3), SUM(failed_4), SUM(failed_5)\n" +
		"FROM pipelines_metrics\nWHERE timeslot BETWEEN $2 AND $3 AND pipeline IN (")
	for i, pipeline := range pipelines {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(strconv.Itoa(pipeline))
	}
	query.WriteString(")\nGROUP BY slot\nORDER BY slot")

	rows, err := c.db.Query(ctx, query.String(), divisor, tsStart, tsEnd)
	if err != nil {
		return Metrics{}, err
	}
	defer rows.Close()

	var slot int32
	var passed, failed [6]int
	for rows.Next() {
		if err = rows.Scan(&slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
			&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5]); err != nil {
			return Metrics{}, err
		}
		i := int(slot - tsStart/divisor)
		if i < 0 || i >= number {
			return Metrics{}, fmt.Errorf("pipelines_metrics table contains a timeslot that is out of range")
		}
		metrics.Passed[i] = passed
		metrics.Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return Metrics{}, err
	}

	return metrics, nil
}

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package statistics

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

// Error represents an error that occurred while executing an action.
type Error struct {
	Action       int
	Step         Step
	Count        int
	Message      string
	LastOccurred time.Time
}

// Stats represents the statistics for a time period.
type Stats struct {
	Start, End time.Time
	Passed     [][6]int
	Failed     [][6]int
}

// Errors returns the errors for the provided actions within the time range
// [start,end). The end time must not precede the start time, and both must be
// within [MinTime,MaxTime]. actions must not be empty. Returned errors are
// limited to [first, first+limit), where first >= 0 and 0 < limit <= 100.
func (s *Statistics) Errors(ctx context.Context, start, end time.Time, actions []int, step *Step, first, limit int) ([]Error, error) {

	tsStart := TimeSlotFromTime(start)
	tsEnd := TimeSlotFromTime(end) - 1

	query := bytes.NewBufferString("SELECT action, MAX(timeslot) AS timeslot, step, count(*), message\n" +
		"FROM actions_errors\nWHERE ")

	query.WriteString("timeslot BETWEEN ")
	query.WriteString(strconv.Itoa(int(tsStart)))
	query.WriteString(" AND ")
	query.WriteString(strconv.Itoa(int(tsEnd)))
	query.WriteString(" AND action IN (")
	for i, action := range actions {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(strconv.Itoa(action))
	}
	query.WriteByte(')')

	if step != nil {
		query.WriteString(" AND step = ")
		s := *step
		query.WriteString(strconv.Itoa(int(s)))
	}

	query.WriteString("\nGROUP BY action, step, message\nORDER BY timeslot DESC, action, message\nLIMIT $1\nOFFSET $2")

	rows, err := s.db.Query(ctx, query.String(), limit, first)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	i := 0
	errs := make([]Error, limit)
	var ts int32
	for rows.Next() {
		if err = rows.Scan(&errs[i].Action, &ts, &errs[i].Step, &errs[i].Count, &errs[i].Message); err != nil {
			return nil, err
		}
		if ts < 0 || ts > maxTimeslot {
			return nil, fmt.Errorf("actions_errors table contains a timeslot that is out of range")
		}
		errs[i].LastOccurred = TimeSlotToTime(ts)
		i++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return errs[:i], nil
}

// StatsPerDate returns statistics aggregated by day for the time interval
// between the specified start and end dates. Both dates must be within the
// range [MinTime,MaxTime], and the day of the start date must be at least one
// day before the day of the end date. actions specifies the actions for which
// statistics are returned and cannot be empty.
func (s *Statistics) StatsPerDate(ctx context.Context, start, end time.Time, actions []int) (Stats, error) {

	number := int(end.Sub(start).Hours() / 24)

	stats := Stats{
		Start:  start,
		End:    end,
		Passed: make([][6]int, number),
		Failed: make([][6]int, number),
	}

	tsStart := TimeSlotFromTime(start)
	tsEnd := TimeSlotFromTime(end) - 1

	query := bytes.NewBufferString("SELECT timeslot/(24*60) AS day, SUM(passed_0), SUM(passed_1), SUM(passed_2), SUM(passed_3), SUM(passed_4), SUM(passed_5)," +
		" SUM(failed_0), SUM(failed_1), SUM(failed_2), SUM(failed_3), SUM(failed_4), SUM(failed_5)\n" +
		"FROM actions_stats\nWHERE timeslot BETWEEN $1 AND $2 AND action IN (")
	for i, action := range actions {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(strconv.Itoa(action))
	}
	query.WriteString(")\nGROUP BY day\nORDER BY day")

	rows, err := s.db.Query(ctx, query.String(), tsStart, tsEnd)
	if err != nil {
		var a any
		a = err
		print(a)
		return Stats{}, err
	}
	defer rows.Close()

	var slot int32
	var passed, failed [6]int
	for rows.Next() {
		if err = rows.Scan(&slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
			&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5]); err != nil {
			return Stats{}, err
		}
		i := int(slot - tsStart/(24*60))
		if i < 0 || i >= number {
			return Stats{}, fmt.Errorf("actions_errors table contains a timeslot that is out of range")
		}
		stats.Passed[i] = passed
		stats.Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return Stats{}, err
	}

	return stats, nil
}

// StatsPerTimeUnit returns statistics for the specified number of minutes,
// hours, or days based on the unit, which can be Minute, Hour, or Day, up to
// the current time. number must be in the following ranges: [1,60] for minutes,
// [1,48] for hours, and [1,30] for days. actions represents the actions for
// which statistics are returned and cannot be empty.
func (s *Statistics) StatsPerTimeUnit(ctx context.Context, number int, unit time.Duration, actions []int) (Stats, error) {

	now := time.Now().UTC()
	end := now.Truncate(unit).Add(unit)

	stats := Stats{
		Start:  end.Add(-time.Duration(number) * unit),
		End:    end,
		Passed: make([][6]int, number),
		Failed: make([][6]int, number),
	}

	divisor := int32(unit / time.Minute)
	tsStart := TimeSlotFromTime(stats.Start)
	tsEnd := TimeSlotFromTime(stats.End) - 1

	query := bytes.NewBufferString("SELECT timeslot/$1 AS slot, SUM(passed_0), SUM(passed_1), SUM(passed_2), SUM(passed_3), SUM(passed_4), SUM(passed_5)," +
		" SUM(failed_0), SUM(failed_1), SUM(failed_2), SUM(failed_3), SUM(failed_4), SUM(failed_5)\n" +
		"FROM actions_stats\nWHERE timeslot BETWEEN $2 AND $3 AND action IN (")
	for i, action := range actions {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteString(strconv.Itoa(action))
	}
	query.WriteString(")\nGROUP BY slot\nORDER BY slot")

	rows, err := s.db.Query(ctx, query.String(), divisor, tsStart, tsEnd)
	if err != nil {
		return Stats{}, err
	}
	defer rows.Close()

	var slot int32
	var passed, failed [6]int
	for rows.Next() {
		if err = rows.Scan(&slot, &passed[0], &passed[1], &passed[2], &passed[3], &passed[4], &passed[5],
			&failed[0], &failed[1], &failed[2], &failed[3], &failed[4], &failed[5]); err != nil {
			return Stats{}, err
		}
		i := int(slot - tsStart/divisor)
		if i < 0 || i >= number {
			return Stats{}, fmt.Errorf("actions_errors table contains a timeslot that is out of range")
		}
		stats.Passed[i] = passed
		stats.Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return Stats{}, err
	}

	return stats, nil
}

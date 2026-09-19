// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/backoff"
	"github.com/krenalis/krenalis/tools/errors"
)

const (
	numSteps         = 7
	timeslotDuration = time.Minute
	maxTimeslot      = int32(math.MaxInt64 / timeslotDuration) // 153722867
)

const (
	Minute = time.Minute
	Hour   = time.Hour
	Day    = 24 * time.Hour
	Month  = 30 * 24 * time.Hour
)

// Pipelines collects, stores, and queries the pipeline metrics.
type Pipelines struct {
	metrics *Metrics
	sync.RWMutex
	pending map[string]*pipelineMetrics // Pending metrics indexed by pipeline ID.
	tick    int
	buf     bytes.Buffer
	stored  struct {
		sync.Cond
		tick int // latest stored tick.
	}
}

// PipelineStep represents a step of a pipeline run.
type PipelineStep int

const (
	ReceiveStep PipelineStep = iota
	InputValidationStep
	FilterStep
	ConsentStep
	TransformationStep
	OutputValidationStep
	FinalizeStep
)

func (s PipelineStep) String() string {
	switch s {
	case ReceiveStep:
		return "Receive"
	case InputValidationStep:
		return "InputValidation"
	case FilterStep:
		return "Filter"
	case ConsentStep:
		return "Consent"
	case TransformationStep:
		return "Transformation"
	case OutputValidationStep:
		return "OutputValidation"
	case FinalizeStep:
		return "Finalize"
	}
	panic("core/metrics: invalid PipelineStep")
}

// PipelineError represents an error that occurred while executing a pipeline.
type PipelineError struct {
	Pipeline     string       `json:"pipeline"`
	Step         PipelineStep `json:"step"`
	Count        int          `json:"count"`
	Message      string       `json:"message"`
	LastOccurred time.Time    `json:"lastOccurred"`
}

// PipelineMetrics represents pipeline metrics for a time period.
type PipelineMetrics struct {
	Start  time.Time
	End    time.Time
	Series []PipelineSeries
}

// PipelineSeries represents pipeline metrics for a single grouping.
type PipelineSeries struct {
	Workspace  string
	Connection string
	Pipeline   string
	Passed     [][numSteps]int
	Failed     [][numSteps]int
}

// PipelineSelection describes which pipeline metric series are returned.
type PipelineSelection struct {
	Workspaces  []string
	Connections []string
	Pipelines   []string
	Target      PipelineTarget
}

// PipelineTarget represents a pipeline target.
type PipelineTarget int

const (
	TargetNone PipelineTarget = iota
	TargetEvent
	TargetUser
	TargetGroup
)

func (t PipelineTarget) String() string {
	switch t {
	case TargetNone:
		return "None"
	case TargetEvent:
		return "Event"
	case TargetUser:
		return "User"
	case TargetGroup:
		return "Group"
	}
	panic("core/internal/metrics: invalid PipelineTarget")
}

// Failed increases the failed count for the specified step and pipeline by the
// given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) Failed(step PipelineStep, pipeline string, count int, message string) {
	p.RLock()
	if m, ok := p.pending[pipeline]; ok {
		m.Lock()
		m.failed[step] += count
		if message != "" {
			m.errors = append(m.errors, pipelineError{step: step, count: count, message: message})
		}
		m.Unlock()
		p.RUnlock()
		return
	}
	p.RUnlock()
	p.Lock()
	m, ok := p.pending[pipeline]
	if !ok {
		var err error
		m, err = p.newMetrics(pipeline)
		if err != nil {
			// The pipeline no longer exists.
			p.Unlock()
			return
		}
		p.pending[pipeline] = m
	}
	m.failed[step] += count
	if message != "" {
		m.errors = append(m.errors, pipelineError{step: step, count: count, message: message})
	}
	p.Unlock()
}

// ConsentFailed increases the failed count for the Consent step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) ConsentFailed(pipeline string, count int) {
	p.Failed(ConsentStep, pipeline, count, "")
}

// ConsentPassed increases the passed count for the Consent step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) ConsentPassed(pipeline string, count int) {
	p.Passed(ConsentStep, pipeline, count)
}

// FilterFailed increases the failed count for the Filter step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) FilterFailed(pipeline string, count int) {
	p.Failed(FilterStep, pipeline, count, "")
}

// FilterPassed increases the passed count for the Filter step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) FilterPassed(pipeline string, count int) {
	p.Passed(FilterStep, pipeline, count)
}

// FinalizeFailed increases the failed count for the Finalize step and pipeline
// by the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) FinalizeFailed(pipeline string, count int, message string) {
	p.Failed(FinalizeStep, pipeline, count, message)
}

// FinalizePassed increases the passed count for the Finalize step and pipeline
// by the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) FinalizePassed(pipeline string, count int) {
	p.Passed(FinalizeStep, pipeline, count)
}

// InputValidationFailed increases the failed count for the InputValidation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (p *Pipelines) InputValidationFailed(pipeline string, count int, message string) {
	p.Failed(InputValidationStep, pipeline, count, message)
}

// InputValidationPassed increases the passed count for the InputValidation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (p *Pipelines) InputValidationPassed(pipeline string, count int) {
	p.Passed(InputValidationStep, pipeline, count)
}

// OutputValidationFailed increases the failed count for the OutputValidation
// step and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (p *Pipelines) OutputValidationFailed(pipeline string, count int, message string) {
	p.Failed(OutputValidationStep, pipeline, count, message)
}

// OutputValidationPassed increases the passed count for the OutputValidation
// step and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (p *Pipelines) OutputValidationPassed(pipeline string, count int) {
	p.Passed(OutputValidationStep, pipeline, count)
}

// Passed increases the passed count for the specified step and pipeline by the
// given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) Passed(step PipelineStep, pipeline string, count int) {
	p.RLock()
	if m, ok := p.pending[pipeline]; ok {
		m.Lock()
		m.passed[step] += count
		m.Unlock()
		p.RUnlock()
		return
	}
	p.RUnlock()
	p.Lock()
	m, ok := p.pending[pipeline]
	if !ok {
		var err error
		m, err = p.newMetrics(pipeline)
		if err != nil {
			// The pipeline no longer exists.
			p.Unlock()
			return
		}
		p.pending[pipeline] = m
	}
	m.passed[step] += count
	p.Unlock()
}

// ReceiveFailed increases the failed count for the Receive step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) ReceiveFailed(pipeline string, count int, message string) {
	p.Failed(ReceiveStep, pipeline, count, message)
}

// ReceivePassed increases the passed count for the Receive step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (p *Pipelines) ReceivePassed(pipeline string, count int) {
	p.Passed(ReceiveStep, pipeline, count)
}

// TransformationFailed increases the failed count for the Transformation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (p *Pipelines) TransformationFailed(pipeline string, count int, message string) {
	p.Failed(TransformationStep, pipeline, count, message)
}

// TransformationPassed increases the passed count for the Transformation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (p *Pipelines) TransformationPassed(pipeline string, count int) {
	p.Passed(TransformationStep, pipeline, count)
}

// WaitStore waits for collected pipeline metrics to be stored in the database.
func (p *Pipelines) WaitStore() {
	p.Lock()
	tick := p.tick
	for {
		p.stored.Wait()
		if p.stored.tick == tick {
			break
		}
	}
	p.Unlock()
}

// aggregate aggregates metrics based on the provided time unit, which can be
// Hour, Day, or Month. It processes the metrics that are older than 60 minutes,
// 48 hours, or 30 days, respectively. timeslot represents the current timeslot.
func (p *Pipelines) aggregate(timeslot int32, unit time.Duration) {

	var interval int32  // Bucket width in minutes.
	var threshold int32 // Rows before this timeslot are aggregated.

	switch unit {
	case Hour:
		interval = 60
		threshold = timeslot + 1 - interval
	case Day:
		interval = 24 * 60
		threshold = timeslot + (60 - (timeslot % 60)) - 48*60
	case Month:
		interval = 30 * 24 * 60
		threshold = timeslot + (24*60 - (timeslot % (24 * 60))) - interval
	}

	query := `WITH aggregated AS (
	SELECT
		organization,
		workspace,
		connection,
		pipeline,
		target,
		timeslot - (timeslot % $1) AS slot,
		SUM(passed_0) AS passed_0,
		SUM(passed_1) AS passed_1,
		SUM(passed_2) AS passed_2,
		SUM(passed_3) AS passed_3,
		SUM(passed_4) AS passed_4,
		SUM(passed_5) AS passed_5,
		SUM(passed_6) AS passed_6,
		SUM(failed_0) AS failed_0,
		SUM(failed_1) AS failed_1,
		SUM(failed_2) AS failed_2,
		SUM(failed_3) AS failed_3,
		SUM(failed_4) AS failed_4,
		SUM(failed_5) AS failed_5,
		SUM(failed_6) AS failed_6,
		ARRAY_AGG(ctid) AS row_ctids
	FROM pipelines_metrics
	WHERE timeslot < $2 AND timeslot % $1 <> 0
	GROUP BY organization, workspace, connection, pipeline, target, slot
),
inserted AS (
	INSERT INTO pipelines_metrics (organization, workspace, connection, pipeline, target, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6)
	SELECT organization, workspace, connection, pipeline, target, slot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6
	FROM aggregated
	ON CONFLICT (pipeline, timeslot)
	DO UPDATE SET
		passed_0 = pipelines_metrics.passed_0 + EXCLUDED.passed_0,
		passed_1 = pipelines_metrics.passed_1 + EXCLUDED.passed_1,
		passed_2 = pipelines_metrics.passed_2 + EXCLUDED.passed_2,
		passed_3 = pipelines_metrics.passed_3 + EXCLUDED.passed_3,
		passed_4 = pipelines_metrics.passed_4 + EXCLUDED.passed_4,
		passed_5 = pipelines_metrics.passed_5 + EXCLUDED.passed_5,
		passed_6 = pipelines_metrics.passed_6 + EXCLUDED.passed_6,
		failed_0 = pipelines_metrics.failed_0 + EXCLUDED.failed_0,
		failed_1 = pipelines_metrics.failed_1 + EXCLUDED.failed_1,
		failed_2 = pipelines_metrics.failed_2 + EXCLUDED.failed_2,
		failed_3 = pipelines_metrics.failed_3 + EXCLUDED.failed_3,
		failed_4 = pipelines_metrics.failed_4 + EXCLUDED.failed_4,
		failed_5 = pipelines_metrics.failed_5 + EXCLUDED.failed_5,
		failed_6 = pipelines_metrics.failed_6 + EXCLUDED.failed_6
)
DELETE FROM pipelines_metrics
WHERE ctid = ANY (SELECT unnest(row_ctids) FROM aggregated)`

	var loggedMsg string

	bo := backoff.New(20)
	for bo.Next(p.metrics.close.ctx) {
		_, err := p.metrics.db.Exec(p.metrics.close.ctx, query, interval, threshold)
		if err == nil {
			break
		}
		if msg := err.Error(); msg != loggedMsg {
			slog.Error("core/metrics: failed to aggregate the metrics on pipeline", "error", msg)
			loggedMsg = msg
		}
	}

}

// start periodically stores pipeline metrics and aggregates older metrics into
// larger time slots.
func (p *Pipelines) start() {
	metrics := map[string]*pipelineMetrics{}
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	timeslot := TimeSlotFromTime(time.Now().UTC())

	for {
		var t time.Time
		isShuttingDown := false
		select {
		case t = <-ticker.C:
			t = t.UTC()
		case <-p.metrics.close.stop:
			isShuttingDown = true
		case <-p.metrics.close.ctx.Done():
			return
		}

		p.Lock()
		metrics, p.pending = p.pending, metrics
		p.tick++
		p.Unlock()

		p.store(timeslot, metrics)

		p.Lock()
		p.stored.tick = p.tick - 1
		p.stored.Broadcast()
		p.Unlock()

		if isShuttingDown {
			return
		}

		for _, m := range metrics {
			m.passed = [numSteps]int{}
			m.failed = [numSteps]int{}
			m.errors = m.errors[0:0]
		}

		timeslot = TimeSlotFromTime(t)

		var unit time.Duration
		switch t.Second() {
		case 0:
			unit = Hour
		case 1:
			unit = Day
		case 2:
			unit = Month
		}
		if unit > 0 && p.metrics.state.IsLeader() {
			p.aggregate(timeslot, unit)
		}
	}
}

// store persists collected pipeline metrics for the specified timeslot.
func (p *Pipelines) store(timeslot int32, metrics map[string]*pipelineMetrics) {

	var hasErrors bool

	b := &p.buf
	b.Reset()
	b.WriteString("WITH t(organization, workspace, connection, pipeline, target, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6) AS (\n\tVALUES ")
	i := 0
	for pipeline, m := range metrics {
		hasErrors = hasErrors || len(m.errors) > 0
		if m.passed == [numSteps]int{} && m.failed == [numSteps]int{} {
			if len(m.errors) == 0 {
				delete(metrics, pipeline)
			}
			continue
		}
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		b.WriteString(db.Quote(m.organization))
		b.WriteByte(',')
		b.WriteString(db.Quote(m.workspace))
		b.WriteByte(',')
		b.WriteString(db.Quote(m.connection))
		b.WriteByte(',')
		b.WriteString(db.Quote(pipeline))
		b.WriteByte(',')
		b.WriteString(db.Quote(m.target.String()))
		b.WriteString("::pipeline_target")
		b.WriteByte(',')
		b.WriteString(strconv.FormatInt(int64(timeslot), 10))
		b.WriteByte(',')
		for j := range 7 {
			b.WriteString(strconv.Itoa(m.passed[j]))
			b.WriteByte(',')
		}
		for j := range 7 {
			b.WriteString(strconv.Itoa(m.failed[j]))
			if j != 6 {
				b.WriteByte(',')
			}
		}
		b.WriteByte(')')
		i++
	}

	if i > 0 {

		b.WriteString("\n) INSERT INTO pipelines_metrics AS m " +
			`(organization, workspace, connection, pipeline, target, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6)` +
			` SELECT t.* FROM t WHERE EXISTS (SELECT 1 FROM organizations o WHERE o.id = t.organization)` +
			` ON CONFLICT (pipeline, timeslot) DO UPDATE SET ` +
			`passed_0 = m.passed_0 + EXCLUDED.passed_0, ` +
			`passed_1 = m.passed_1 + EXCLUDED.passed_1, ` +
			`passed_2 = m.passed_2 + EXCLUDED.passed_2, ` +
			`passed_3 = m.passed_3 + EXCLUDED.passed_3, ` +
			`passed_4 = m.passed_4 + EXCLUDED.passed_4, ` +
			`passed_5 = m.passed_5 + EXCLUDED.passed_5, ` +
			`passed_6 = m.passed_6 + EXCLUDED.passed_6, ` +
			`failed_0 = m.failed_0 + EXCLUDED.failed_0, ` +
			`failed_1 = m.failed_1 + EXCLUDED.failed_1, ` +
			`failed_2 = m.failed_2 + EXCLUDED.failed_2, ` +
			`failed_3 = m.failed_3 + EXCLUDED.failed_3, ` +
			`failed_4 = m.failed_4 + EXCLUDED.failed_4, ` +
			`failed_5 = m.failed_5 + EXCLUDED.failed_5, ` +
			`failed_6 = m.failed_6 + EXCLUDED.failed_6`)

		query := b.String()

		var loggedMsg string

		bo := backoff.New(20)
		for bo.Next(p.metrics.close.ctx) {
			_, err := p.metrics.db.Exec(p.metrics.close.ctx, query)
			if err == nil {
				break
			}
			if msg := err.Error(); msg != loggedMsg {
				slog.Error("core/metrics: failed to store the metrics on pipeline", "error", msg)
			}
		}

	}

	if !hasErrors {
		return
	}

	b.Reset()
	b.WriteString(`INSERT INTO pipelines_errors (pipeline, timeslot, step, count, message)` +
		` SELECT t.* FROM (VALUES `)
	i = 0
	for pipeline, m := range metrics {
		for _, err := range m.errors {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte('(')
			b.WriteString(db.Quote(pipeline))
			b.WriteByte(',')
			b.WriteString(strconv.Itoa(int(timeslot)))
			b.WriteByte(',')
			b.WriteString(strconv.Itoa(int(err.step)))
			b.WriteByte(',')
			b.WriteString(strconv.Itoa(err.count))
			b.WriteByte(',')
			b.WriteString(db.Quote(err.message))
			b.WriteByte(')')
			i++
		}
	}
	b.WriteString(`) AS t(pipeline, timeslot, step, count, message)` +
		` JOIN pipelines p ON p.id = t.pipeline`)
	query := b.String()

	var loggedMsg string

	bo := backoff.New(20)
	for bo.Next(p.metrics.close.ctx) {
		_, err := p.metrics.db.Exec(p.metrics.close.ctx, query)
		if err == nil {
			break
		}
		if msg := err.Error(); msg != loggedMsg {
			slog.Error("core/metrics: failed to store the errors on pipeline", "error", msg)
		}
	}
}

// newMetrics creates metrics for the pipeline identified by the provided ID.
// It returns an error only if the pipeline no longer exists.
func (p *Pipelines) newMetrics(pipeline string) (*pipelineMetrics, error) {
	pipelineState, ok := p.metrics.state.Pipeline(pipeline)
	if !ok {
		return nil, errors.New("pipeline not found")
	}
	connection := pipelineState.Connection()
	return &pipelineMetrics{
		organization: connection.Organization().ID,
		workspace:    connection.Workspace().ID,
		connection:   connection.ID,
		target:       pipelineState.Target,
	}, nil
}

// Errors returns the errors for the provided pipelines within the time range
// [start,end). The end time must not precede the start time, and both must be
// within [MinTime,MaxTime]. pipelines must not be empty. Returned errors are
// limited to [first, first+limit), where first >= 0 and 0 < limit <= 100.
func (p *Pipelines) Errors(ctx context.Context, start, end time.Time, pipelines []string, step *PipelineStep, first, limit int) ([]PipelineError, error) {

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

	rows, err := p.metrics.db.Query(ctx, query.String(), limit, first)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	i := 0
	errs := make([]PipelineError, limit)
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
// day before the day of the end date. selection specifies which metric series
// are returned.
//
// It returns [ErrMetricResultTooLarge] when an aggregated count cannot be
// represented by the result type.
func (p *Pipelines) MetricsPerDate(ctx context.Context, start, end time.Time, selection PipelineSelection) (PipelineMetrics, error) {
	return p.queryMetrics(ctx, start, end, Day, selection)
}

// MetricsPerTimeUnit returns metrics for the specified number of minutes,
// hours, or days based on the unit, which can be Minute, Hour, or Day, up to
// the current time. number must be in the following ranges: [1,60] for minutes,
// [1,48] for hours, and [1,30] for days. selection specifies which metric
// series are returned.
//
// It returns [ErrMetricResultTooLarge] when an aggregated count cannot be
// represented by the result type.
func (p *Pipelines) MetricsPerTimeUnit(ctx context.Context, number int, unit time.Duration, selection PipelineSelection) (PipelineMetrics, error) {
	now := time.Now().UTC()
	end := now.Truncate(unit).Add(unit)
	start := end.Add(-time.Duration(number) * unit)
	return p.queryMetrics(ctx, start, end, unit, selection)
}

// queryMetrics returns dense passed and failed count series for [start,end),
// grouped at the requested resolution by workspace, connection, or pipeline,
// as specified by selection. If selection specifies a target, only metrics for
// that target are returned.
//
// It returns [ErrMetricResultTooLarge] when an aggregated count cannot be
// represented by the result type.
func (p *Pipelines) queryMetrics(ctx context.Context, start, end time.Time, resolution time.Duration, selection PipelineSelection) (PipelineMetrics, error) {

	var query strings.Builder
	query.WriteString("SELECT ")
	switch {
	case selection.Workspaces != nil:
		query.WriteString("workspace, ")
	case selection.Connections != nil:
		query.WriteString("connection, ")
	case selection.Pipelines != nil:
		query.WriteString("pipeline, ")
	}
	query.WriteString("timeslot/$1 AS slot, SUM(passed_0::numeric)::text, SUM(passed_1::numeric)::text, SUM(passed_2::numeric)::text, SUM(passed_3::numeric)::text, SUM(passed_4::numeric)::text, SUM(passed_5::numeric)::text, SUM(passed_6::numeric)::text," +
		" SUM(failed_0::numeric)::text, SUM(failed_1::numeric)::text, SUM(failed_2::numeric)::text, SUM(failed_3::numeric)::text, SUM(failed_4::numeric)::text, SUM(failed_5::numeric)::text, SUM(failed_6::numeric)::text\n" +
		"FROM pipelines_metrics\nWHERE timeslot BETWEEN $2 AND $3")
	switch {
	case selection.Workspaces != nil:
		query.WriteString(" AND workspace IN (")
		writeQuotedValues(&query, selection.Workspaces)
		query.WriteByte(')')
	case selection.Connections != nil:
		query.WriteString(" AND connection IN (")
		writeQuotedValues(&query, selection.Connections)
		query.WriteByte(')')
	case selection.Pipelines != nil:
		query.WriteString(" AND pipeline IN (")
		writeQuotedValues(&query, selection.Pipelines)
		query.WriteByte(')')
	}
	if selection.Target != TargetNone {
		query.WriteString(" AND target = ")
		query.WriteString(db.Quote(selection.Target.String()))
	}
	query.WriteString("\nGROUP BY ")
	switch {
	case selection.Workspaces != nil:
		query.WriteString("workspace, slot\nORDER BY workspace, slot")
	case selection.Connections != nil:
		query.WriteString("connection, slot\nORDER BY connection, slot")
	case selection.Pipelines != nil:
		query.WriteString("pipeline, slot\nORDER BY pipeline, slot")
	}

	divisor := int32(resolution / time.Minute)
	tsStart := TimeSlotFromTime(start)
	tsEnd := TimeSlotFromTime(end) - 1

	rows, err := p.metrics.db.Query(ctx, query.String(), divisor, tsStart, tsEnd)
	if err != nil {
		return PipelineMetrics{}, err
	}
	defer rows.Close()

	metrics := PipelineMetrics{
		Start: start,
		End:   end,
	}

	number := int(end.Sub(start) / resolution)

	var currentID string
	var series *PipelineSeries

	for rows.Next() {
		var slot int32
		var id string
		var passed, failed [numSteps]int
		var passedValues, failedValues [numSteps]string
		err = rows.Scan(&id, &slot,
			&passedValues[0], &passedValues[1], &passedValues[2], &passedValues[3], &passedValues[4], &passedValues[5], &passedValues[6],
			&failedValues[0], &failedValues[1], &failedValues[2], &failedValues[3], &failedValues[4], &failedValues[5], &failedValues[6])
		if err != nil {
			return PipelineMetrics{}, err
		}
		for step := range numSteps {
			passed[step], err = parsePipelineMetricCount(passedValues[step])
			if err != nil {
				return PipelineMetrics{}, err
			}
			failed[step], err = parsePipelineMetricCount(failedValues[step])
			if err != nil {
				return PipelineMetrics{}, err
			}
		}
		i := int(slot - tsStart/divisor)
		if i < 0 || i >= number {
			return PipelineMetrics{}, fmt.Errorf("pipelines_metrics table contains timeslot %d that is out of range", slot)
		}
		if id != currentID {
			currentID = id
			metrics.Series = append(metrics.Series, PipelineSeries{})
			series = &metrics.Series[len(metrics.Series)-1]
			series.Passed = make([][numSteps]int, number)
			series.Failed = make([][numSteps]int, number)
			switch {
			case selection.Workspaces != nil:
				series.Workspace = id
			case selection.Connections != nil:
				series.Connection = id
			case selection.Pipelines != nil:
				series.Pipeline = id
			}
		}
		series.Passed[i] = passed
		series.Failed[i] = failed
	}
	if err := rows.Err(); err != nil {
		return PipelineMetrics{}, err
	}

	return metrics, nil
}

type pipelineError struct {
	step    PipelineStep
	count   int
	message string
}

// pipelineMetrics holds pipeline metrics collected during a given time period,
// pending their eventual write to the database.
type pipelineMetrics struct {
	sync.Mutex
	organization string
	workspace    string
	connection   string
	target       state.Target
	// TODO: Bound pipeline counts end-to-end or use types that keep pending,
	// persisted, and aggregated totals representable.
	passed [numSteps]int
	failed [numSteps]int
	errors []pipelineError
}

// parsePipelineMetricCount parses an exact PostgreSQL aggregate as an int. It
// returns [ErrMetricResultTooLarge] when the value is outside the int range and
// returns other parsing errors unchanged.
func parsePipelineMetricCount(value string) (int, error) {
	count, err := strconv.Atoi(value)
	if errors.Is(err, strconv.ErrRange) {
		return 0, ErrMetricResultTooLarge
	}
	return count, err
}

// writeQuotedValues writes the provided strings as comma-separated SQL
// literals.
func writeQuotedValues(b *strings.Builder, values []string) {
	for i, value := range values {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(db.Quote(value))
	}
}

// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package metrics collects and stores pipeline metrics in the database. It
// tracks both batch runs and events related to receiving or dispatching.
package metrics

import (
	"bytes"
	"context"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/backoff"
)

const (
	numSteps         = 6
	timeslotDuration = time.Minute
	flushInterval    = time.Second
	maxTimeslot      = int32(math.MaxInt64 / timeslotDuration) // 153722867
)

var (
	MinTime = TimeSlotToTime(0)           // 1970-01-01 00:00:00
	MaxTime = TimeSlotToTime(maxTimeslot) // 2262-04-11 23:47:00
)

// Step represents a step of a run.
type Step int

const (
	ReceiveStep Step = iota
	InputValidationStep
	FilterStep
	TransformationStep
	OutputValidationStep
	FinalizeStep
)

func (s Step) String() string {
	switch s {
	case ReceiveStep:
		return "Receive"
	case InputValidationStep:
		return "InputValidation"
	case FilterStep:
		return "Filter"
	case TransformationStep:
		return "Transformation"
	case OutputValidationStep:
		return "OutputValidation"
	case FinalizeStep:
		return "Finalize"
	}
	panic("core/metrics: invalid Step")
}

// Collector is a metrics collector.
type Collector struct {
	db      *db.DB
	state   *state.State
	mu      sync.RWMutex
	metrics map[int]*metrics
	tick    int
	buf     bytes.Buffer
	stored  struct {
		sync.Cond
		tick int // latest stored tick.
	}
	close struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		shutdown  chan struct{}
	}
}

// New returns a new metrics collector. This collector collects metrics with a
// resolution of one minute and saves the collected metrics to the database
// every minute.
func New(db *db.DB, state *state.State) *Collector {
	c := &Collector{
		db:      db,
		state:   state,
		metrics: map[int]*metrics{},
		tick:    1,
	}
	c.stored.L = &c.mu
	c.close.ctx, c.close.cancelCtx = context.WithCancel(context.Background())
	c.close.shutdown = make(chan struct{})
	go c.start()
	return c
}

// Close closes the collection and ensures that any remaining metrics are
// stored. If the provided context expires before the operation completes, the
// ongoing store operation is interrupted and the method returns without
// guaranteeing that all metrics have been saved.
func (c *Collector) Close(ctx context.Context) {
	close(c.close.shutdown)
	stop := context.AfterFunc(ctx, func() { c.close.cancelCtx() })
	defer stop()
}

// Failed increases the failed count for the specified step and pipeline by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) Failed(step Step, pipeline, count int, message string) {
	c.mu.RLock()
	if m, ok := c.metrics[pipeline]; ok {
		m.Lock()
		m.failed[step] += count
		if message != "" {
			m.errors = append(m.errors, pipelineError{step: step, count: count, message: message})
		}
		m.Unlock()
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()
	c.mu.Lock()
	m, ok := c.metrics[pipeline]
	if !ok {
		m = &metrics{}
		c.metrics[pipeline] = m
	}
	m.failed[step] += count
	if message != "" {
		m.errors = append(m.errors, pipelineError{step: step, count: count, message: message})
	}
	c.mu.Unlock()
}

// FilterFailed increases the failed count for the Filter step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FilterFailed(pipeline, count int) {
	c.Failed(FilterStep, pipeline, count, "")
}

// FilterPassed increases the passed count for the Filter step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FilterPassed(pipeline, count int) {
	c.Passed(FilterStep, pipeline, count)
}

// FinalizeFailed increases the failed count for the Finalize step and pipeline
// by the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FinalizeFailed(pipeline, count int, message string) {
	c.Failed(FinalizeStep, pipeline, count, message)
}

// FinalizePassed increases the passed count for the Finalize step and pipeline
// by the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FinalizePassed(pipeline, count int) {
	c.Passed(FinalizeStep, pipeline, count)
}

// InputValidationFailed increases the failed count for the InputValidation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (c *Collector) InputValidationFailed(pipeline, count int, message string) {
	c.Failed(InputValidationStep, pipeline, count, message)
}

// InputValidationPassed increases the passed count for the InputValidation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (c *Collector) InputValidationPassed(pipeline, count int) {
	c.Passed(InputValidationStep, pipeline, count)
}

// OutputValidationFailed increases the failed count for the OutputValidation
// step and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (c *Collector) OutputValidationFailed(pipeline, count int, message string) {
	c.Failed(OutputValidationStep, pipeline, count, message)
}

// OutputValidationPassed increases the passed count for the OutputValidation
// step and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (c *Collector) OutputValidationPassed(pipeline, count int) {
	c.Passed(OutputValidationStep, pipeline, count)
}

// Passed increases the passed count for the specified step and pipeline by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) Passed(step Step, pipeline, count int) {
	c.mu.RLock()
	if m, ok := c.metrics[pipeline]; ok {
		m.Lock()
		m.passed[step] += count
		m.Unlock()
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()
	c.mu.Lock()
	m, ok := c.metrics[pipeline]
	if !ok {
		m = &metrics{}
		c.metrics[pipeline] = m
	}
	m.passed[step] += count
	c.mu.Unlock()
}

// ReceiveFailed increases the failed count for the Receive step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) ReceiveFailed(pipeline, count int, message string) {
	c.Failed(ReceiveStep, pipeline, count, message)
}

// ReceivePassed increases the passed count for the Receive step and pipeline by
// the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) ReceivePassed(pipeline, count int) {
	c.Passed(ReceiveStep, pipeline, count)
}

// TransformationFailed increases the failed count for the Transformation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (c *Collector) TransformationFailed(pipeline, count int, message string) {
	c.Failed(TransformationStep, pipeline, count, message)
}

// TransformationPassed increases the passed count for the Transformation step
// and pipeline by the given count. It is safe to call concurrently from
// multiple goroutines.
func (c *Collector) TransformationPassed(pipeline, count int) {
	c.Passed(TransformationStep, pipeline, count)
}

// WaitStore waits for any collected metrics to be stored to the database before
// returning.
func (c *Collector) WaitStore() {
	c.mu.Lock()
	tick := c.tick
	for {
		c.stored.Wait()
		if c.stored.tick == tick {
			break
		}
	}
	c.mu.Unlock()
}

// aggregate aggregates metrics based on the provided time unit, which can be
// Hour, Day, or Month. It processes the metrics that are older than 60 minutes,
// 48 hours, or 30 days, respectively. timeslot represents the current timeslot.
func (c *Collector) aggregate(timeslot int32, unit time.Duration) {

	var interval int32
	var threshold int32

	switch unit {
	case Hour:
		interval = 60
		threshold = timeslot + 1 - interval
	case Day:
		interval = 48 * 60
		threshold = timeslot + (60 - (timeslot % 60)) - interval
	case Month:
		interval = 30 * 24 * 60
		threshold = timeslot + (24*60 - (timeslot % (24 * 60))) - interval
	}

	query := `WITH aggregated AS (
	SELECT
		pipeline,
		timeslot - (timeslot % $1) AS slot,
		SUM(passed_0) AS passed_0,
		SUM(passed_1) AS passed_1,
		SUM(passed_2) AS passed_2,
		SUM(passed_3) AS passed_3,
		SUM(passed_4) AS passed_4,
		SUM(passed_5) AS passed_5,
		SUM(failed_0) AS failed_0,
		SUM(failed_1) AS failed_1,
		SUM(failed_2) AS failed_2,
		SUM(failed_3) AS failed_3,
		SUM(failed_4) AS failed_4,
		SUM(failed_5) AS failed_5,
		ARRAY_AGG(ctid) AS row_ctids
	FROM pipelines_metrics
	WHERE timeslot < $2 AND timeslot % $1 <> 0
	GROUP BY pipeline, slot
),
inserted AS (
	INSERT INTO pipelines_metrics (pipeline, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)
	SELECT pipeline, slot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5
	FROM aggregated
	ON CONFLICT (pipeline, timeslot)
	DO UPDATE SET
		passed_0 = pipelines_metrics.passed_0 + EXCLUDED.passed_0,
		passed_1 = pipelines_metrics.passed_1 + EXCLUDED.passed_1,
		passed_2 = pipelines_metrics.passed_2 + EXCLUDED.passed_2,
		passed_3 = pipelines_metrics.passed_3 + EXCLUDED.passed_3,
		passed_4 = pipelines_metrics.passed_4 + EXCLUDED.passed_4,
		passed_5 = pipelines_metrics.passed_5 + EXCLUDED.passed_5,
		failed_0 = pipelines_metrics.failed_0 + EXCLUDED.failed_0,
		failed_1 = pipelines_metrics.failed_1 + EXCLUDED.failed_1,
		failed_2 = pipelines_metrics.failed_2 + EXCLUDED.failed_2,
		failed_3 = pipelines_metrics.failed_3 + EXCLUDED.failed_3,
		failed_4 = pipelines_metrics.failed_4 + EXCLUDED.failed_4,
		failed_5 = pipelines_metrics.failed_5 + EXCLUDED.failed_5
)
DELETE FROM pipelines_metrics
WHERE ctid = ANY (SELECT unnest(row_ctids) FROM aggregated)`

	var loggedMsg string

	bo := backoff.New(20)
	for bo.Next(c.close.ctx) {
		_, err := c.db.Exec(c.close.ctx, query, interval, threshold)
		if err == nil {
			break
		}
		if msg := err.Error(); msg != loggedMsg {
			slog.Error("core/metrics: failed to aggregate the metrics on pipeline", "err", msg)
			loggedMsg = msg
		}
	}

}

// start starts the metrics collector. It collects metrics every minute and
// aggregates them into 10-minute time slots for storage.
func (c *Collector) start() {

	var isShuttingDown bool

	metrics := map[int]*metrics{}

	// Wait the time to the next timeslot.
	now := time.Now().UTC()
	t := now.Add(time.Second).Truncate(time.Minute)
	time.Sleep(t.Sub(now))

	// Starts the ticker.
	ticker := time.NewTicker(flushInterval)
	timeslot := TimeSlotFromTime(t)

	for {

		c.mu.Lock()
		metrics, c.metrics = c.metrics, metrics
		c.tick++
		c.mu.Unlock()

		c.store(timeslot, metrics)

		c.mu.Lock()
		c.stored.tick = c.tick - 1
		c.stored.Broadcast()
		c.mu.Unlock()

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
		if unit > 0 && c.state.IsLeader() {
			c.aggregate(timeslot, unit)
		}

		select {
		case t = <-ticker.C:
		case <-c.close.shutdown:
			isShuttingDown = true
		case <-c.close.ctx.Done():
			return
		}

	}

}

// store stored any collected metrics in metrics for the specified timeslot to
// the database.
func (c *Collector) store(timeslot int32, metrics map[int]*metrics) {

	var hasErrors bool

	c.buf.Reset()
	c.buf.WriteString("WITH t AS (\n\tVALUES ")
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
			c.buf.WriteByte(',')
		}
		c.buf.WriteByte('(')
		c.buf.WriteString(strconv.Itoa(pipeline))
		c.buf.WriteByte(',')
		c.buf.WriteString(strconv.FormatInt(int64(timeslot), 10))
		c.buf.WriteByte(',')
		for j := 0; j < 6; j++ {
			c.buf.WriteString(strconv.Itoa(m.passed[j]))
			c.buf.WriteByte(',')
		}
		for j := 0; j < 6; j++ {
			c.buf.WriteString(strconv.Itoa(m.failed[j]))
			if j != 5 {
				c.buf.WriteByte(',')
			}
		}
		c.buf.WriteByte(')')
		i++
	}

	if i > 0 {

		c.buf.WriteString("\n) INSERT INTO pipelines_metrics AS m " +
			`(pipeline, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)` +
			` SELECT * FROM t ON CONFLICT (pipeline, timeslot) DO UPDATE SET ` +
			`passed_0 = m.passed_0 + EXCLUDED.passed_0, ` +
			`passed_1 = m.passed_1 + EXCLUDED.passed_1, ` +
			`passed_2 = m.passed_2 + EXCLUDED.passed_2, ` +
			`passed_3 = m.passed_3 + EXCLUDED.passed_3, ` +
			`passed_4 = m.passed_4 + EXCLUDED.passed_4, ` +
			`passed_5 = m.passed_5 + EXCLUDED.passed_5, ` +
			`failed_0 = m.failed_0 + EXCLUDED.failed_0, ` +
			`failed_1 = m.failed_1 + EXCLUDED.failed_1, ` +
			`failed_2 = m.failed_2 + EXCLUDED.failed_2, ` +
			`failed_3 = m.failed_3 + EXCLUDED.failed_3, ` +
			`failed_4 = m.failed_4 + EXCLUDED.failed_4, ` +
			`failed_5 = m.failed_5 + EXCLUDED.failed_5`)

		query := c.buf.String()

		var loggedMsg string

		bo := backoff.New(20)
		for bo.Next(c.close.ctx) {
			_, err := c.db.Exec(c.close.ctx, query)
			if err == nil {
				break
			}
			if msg := err.Error(); msg != loggedMsg {
				slog.Error("core/metrics: failed to store the metrics on pipeline", "err", msg)
			}
		}

	}

	if !hasErrors {
		return
	}

	c.buf.Reset()
	c.buf.WriteString("INSERT INTO pipelines_errors (pipeline, timeslot, step, count, message) VALUES ")
	i = 0
	for pipeline, m := range metrics {
		for _, err := range m.errors {
			if i > 0 {
				c.buf.WriteByte(',')
			}
			c.buf.WriteByte('(')
			c.buf.WriteString(strconv.Itoa(pipeline))
			c.buf.WriteByte(',')
			c.buf.WriteString(strconv.Itoa(int(timeslot)))
			c.buf.WriteByte(',')
			c.buf.WriteString(strconv.Itoa(int(err.step)))
			c.buf.WriteByte(',')
			c.buf.WriteString(strconv.Itoa(err.count))
			c.buf.WriteByte(',')
			c.buf.WriteString(db.Quote(err.message))
			c.buf.WriteByte(')')
			i++
		}
	}
	query := c.buf.String()

	var loggedMsg string

	bo := backoff.New(20)
	for bo.Next(c.close.ctx) {
		_, err := c.db.Exec(c.close.ctx, query)
		if err == nil {
			break
		}
		if msg := err.Error(); msg != loggedMsg {
			slog.Error("core/metrics: failed to store the errors on pipeline", "err", msg)
		}
	}
}

// TimeSlotFromTime returns the time slot for t that must be in UTC.
func TimeSlotFromTime(t time.Time) int32 {
	return int32(t.Unix() / int64(timeslotDuration.Seconds()))
}

// TimeSlotToTime converts a time slot back to a time.Time in UTC.
// It panics if ts is not in range [0,maxTimeslot].
func TimeSlotToTime(ts int32) time.Time {
	if ts < 0 || ts > maxTimeslot {
		panic("timeslot is out of range")
	}
	epoch := time.Unix(0, 0).UTC()
	return epoch.Add(time.Duration(ts) * timeslotDuration)
}

type pipelineError struct {
	step    Step
	count   int
	message string
}

// metrics holds the pipeline metrics that need to be stored to the database. It
// serves as a temporary storage for metrics collected during a given time
// period, pending their eventual write to the database.
type metrics struct {
	sync.Mutex
	passed [numSteps]int
	failed [numSteps]int
	errors []pipelineError
}

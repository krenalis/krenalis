//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package statistics collects and stores action statistics in the database.
// It tracks both batch executions and events related to receiving or
// dispatching.
package statistics

import (
	"bytes"
	"context"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/backoff"
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

// Step represents a step of an execution.
type Step int

const (
	ReceivingStep Step = iota
	InputValidationStep
	FilteringStep
	TransformationStep
	OutputValidationStep
	FinalizingStep
)

func (s Step) String() string {
	switch s {
	case ReceivingStep:
		return "Receiving"
	case InputValidationStep:
		return "InputValidation"
	case FilteringStep:
		return "Filtering"
	case TransformationStep:
		return "Transformation"
	case OutputValidationStep:
		return "OutputValidation"
	case FinalizingStep:
		return "Finalizing"
	}
	panic("apis/statistics: invalid Step")
}

// Statistics is a statistics collector.
type Statistics struct {
	db         *postgres.DB
	state      *state.State
	listener   uint8
	mu         sync.RWMutex
	diff       map[int]bool
	collectors map[int]*Collector
	stats      map[int]*statistics
	tick       int
	buf        bytes.Buffer
	stored     struct {
		sync.Cond
		tick int // latest stored tick.
	}
	close struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		shutdown  chan struct{}
	}
}

// New returns a new Statistics collector. This collector gathers collects
// statistics with a resolution of one minute and saves the collected
// statistics to the database every minute.
func New(db *postgres.DB, state *state.State) *Statistics {
	s := &Statistics{
		db:         db,
		state:      state,
		diff:       map[int]bool{},
		collectors: map[int]*Collector{},
		stats:      map[int]*statistics{},
		tick:       1,
	}
	s.stored.L = &s.mu
	s.close.ctx, s.close.cancelCtx = context.WithCancel(context.Background())
	s.close.shutdown = make(chan struct{})
	go s.start()
	return s
}

// Close closes the collection and ensures that any remaining statistics are
// stored. If the provided context expires before the operation completes, the
// ongoing store operation is interrupted and the method returns without
// guaranteeing that all statistics have been saved.
func (s *Statistics) Close(ctx context.Context) {
	close(s.close.shutdown)
	stop := context.AfterFunc(ctx, func() { s.close.cancelCtx() })
	defer stop()
}

// Collector return a collector for the specified action identifier. If a
// collector for the given action does not already exist, it creates and returns
// a new one.
func (s *Statistics) Collector(action int) *Collector {
	s.mu.Lock()
	c, ok := s.collectors[action]
	if !ok {
		c = &Collector{action: action, s: s}
		s.collectors[action] = c
		s.stats[action] = &statistics{}
		s.diff[action] = true
	}
	s.mu.Unlock()
	return c
}

// start starts the statistics collector. It collects statistics every minute
// and aggregates them into 10-minute time slots for storage.
func (s *Statistics) start() {

	var isShuttingDown bool

	stats := map[int]*statistics{}

	// Wait the time to the next timeslot.
	now := time.Now().UTC()
	t := now.Add(time.Second).Truncate(time.Minute)
	time.Sleep(t.Sub(now))

	// Starts the ticker.
	ticker := time.NewTicker(flushInterval)
	timeslot := TimeSlotFromTime(t)

	for {

		s.mu.Lock()
		if len(s.diff) > 0 {
			for action, added := range s.diff {
				if added {
					stats[action] = &statistics{}
				} else {
					delete(stats, action)
				}
			}
			clear(s.diff)
		}
		stats, s.stats = s.stats, stats
		s.tick++
		s.mu.Unlock()

		s.store(timeslot, stats)

		s.mu.Lock()
		s.stored.tick = s.tick - 1
		s.stored.Broadcast()
		s.mu.Unlock()

		if isShuttingDown {
			return
		}

		for _, st := range stats {
			st.passed = [numSteps]int{}
			st.failed = [numSteps]int{}
			st.errors = st.errors[0:0]
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
		if unit > 0 && s.state.IsLeader() {
			s.aggregate(timeslot, unit)
		}

		select {
		case t = <-ticker.C:
		case <-s.close.shutdown:
			isShuttingDown = true
		case <-s.close.ctx.Done():
			return
		}

	}

}

// aggregate aggregates statistics based on the provided time unit, which can be
// Hour, Day, or Month. It processes the statistics that are older than 60
// minutes, 48 hours, or 30 days, respectively. timeslot represents the current
// timeslot.
func (s *Statistics) aggregate(timeslot int32, unit time.Duration) {

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
		threshold = timeslot + (24*60 - (timeslot % 24 * 60)) - interval
	}

	query := `WITH aggregated AS (
	SELECT
		action,
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
	FROM actions_stats
	WHERE timeslot < $2 AND timeslot % $1 <> 0
	GROUP BY action, slot
),
inserted AS (
	INSERT INTO actions_stats (action, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)
	SELECT action, slot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5
	FROM aggregated
	ON CONFLICT (action, timeslot)
	DO UPDATE SET
		passed_0 = actions_stats.passed_0 + EXCLUDED.passed_0,
		passed_1 = actions_stats.passed_1 + EXCLUDED.passed_1,
		passed_2 = actions_stats.passed_2 + EXCLUDED.passed_2,
		passed_3 = actions_stats.passed_3 + EXCLUDED.passed_3,
		passed_4 = actions_stats.passed_4 + EXCLUDED.passed_4,
		passed_5 = actions_stats.passed_5 + EXCLUDED.passed_5,
		failed_0 = actions_stats.failed_0 + EXCLUDED.failed_0,
		failed_1 = actions_stats.failed_1 + EXCLUDED.failed_1,
		failed_2 = actions_stats.failed_2 + EXCLUDED.failed_2,
		failed_3 = actions_stats.failed_3 + EXCLUDED.failed_3,
		failed_4 = actions_stats.failed_4 + EXCLUDED.failed_4,
		failed_5 = actions_stats.failed_5 + EXCLUDED.failed_5
)
DELETE FROM actions_stats
WHERE ctid = ANY (SELECT unnest(row_ctids) FROM aggregated)`

	var loggedMsg string

	bo := backoff.New(20)
	for bo.Next(s.close.ctx) {
		_, err := s.db.Exec(s.close.ctx, query, interval, threshold)
		if err == nil {
			break
		}
		var a any
		a = err
		print(a)
		if msg := err.Error(); msg != loggedMsg {
			slog.Error("failed to aggregate the statistics on action", "err", msg)
		}
	}

}

// store stored any collected statistics in stats for the specified timeslot
// to the database.
func (s *Statistics) store(timeslot int32, stats map[int]*statistics) {

	var hasErrors bool

	s.buf.Reset()
	s.buf.WriteString("WITH t AS (\n\tVALUES ")
	i := 0
	for action, st := range stats {
		hasErrors = hasErrors || len(st.errors) > 0
		if st.passed == [numSteps]int{} && st.failed == [numSteps]int{} {
			continue
		}
		if i > 0 {
			s.buf.WriteByte(',')
		}
		s.buf.WriteByte('(')
		s.buf.WriteString(strconv.Itoa(action))
		s.buf.WriteByte(',')
		s.buf.WriteString(strconv.FormatInt(int64(timeslot), 10))
		s.buf.WriteByte(',')
		for j := 0; j < 6; j++ {
			s.buf.WriteString(strconv.Itoa(st.passed[j]))
			s.buf.WriteByte(',')
		}
		for j := 0; j < 6; j++ {
			s.buf.WriteString(strconv.Itoa(st.failed[j]))
			if j != 5 {
				s.buf.WriteByte(',')
			}
		}
		s.buf.WriteByte(')')
		i++
	}

	if i > 0 {

		s.buf.WriteString("\n) INSERT INTO actions_stats AS s " +
			`(action, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)` +
			` SELECT * FROM t ON CONFLICT (action, timeslot) DO UPDATE SET ` +
			`passed_0 = s.passed_0 + EXCLUDED.passed_0, ` +
			`passed_1 = s.passed_1 + EXCLUDED.passed_1, ` +
			`passed_2 = s.passed_2 + EXCLUDED.passed_2, ` +
			`passed_3 = s.passed_3 + EXCLUDED.passed_3, ` +
			`passed_4 = s.passed_4 + EXCLUDED.passed_4, ` +
			`passed_5 = s.passed_5 + EXCLUDED.passed_5, ` +
			`failed_0 = s.failed_0 + EXCLUDED.failed_0, ` +
			`failed_1 = s.failed_1 + EXCLUDED.failed_1, ` +
			`failed_2 = s.failed_2 + EXCLUDED.failed_2, ` +
			`failed_3 = s.failed_3 + EXCLUDED.failed_3, ` +
			`failed_4 = s.failed_4 + EXCLUDED.failed_4, ` +
			`failed_5 = s.failed_5 + EXCLUDED.failed_5`)

		query := s.buf.String()

		var loggedMsg string

		bo := backoff.New(20)
		for bo.Next(s.close.ctx) {
			_, err := s.db.Exec(s.close.ctx, query)
			if err == nil {
				break
			}
			if msg := err.Error(); msg != loggedMsg {
				slog.Error("failed to store the statistics on action", "err", msg)
			}
		}

	}

	if !hasErrors {
		return
	}

	s.buf.Reset()
	s.buf.WriteString("INSERT INTO actions_errors (action, timeslot, step, count, message) VALUES ")
	i = 0
	for action, st := range stats {
		for _, err := range st.errors {
			if i > 0 {
				s.buf.WriteByte(',')
			}
			s.buf.WriteByte('(')
			s.buf.WriteString(strconv.Itoa(action))
			s.buf.WriteByte(',')
			s.buf.WriteString(strconv.Itoa(int(timeslot)))
			s.buf.WriteByte(',')
			s.buf.WriteString(strconv.Itoa(int(err.step)))
			s.buf.WriteByte(',')
			s.buf.WriteString(strconv.Itoa(err.count))
			s.buf.WriteByte(',')
			s.buf.WriteString(postgres.QuoteValue(err.message))
			s.buf.WriteByte(')')
			i++
		}
	}
	query := s.buf.String()

	var loggedMsg string

	bo := backoff.New(20)
	for bo.Next(s.close.ctx) {
		_, err := s.db.Exec(s.close.ctx, query)
		if err == nil {
			break
		}
		if msg := err.Error(); msg != loggedMsg {
			slog.Error("failed to store the errors on action", "err", msg)
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

type actionError struct {
	step    Step
	count   int
	message string
}

// statistics holds the action statistics that need to be stored to the database.
// It serves as a temporary storage for statistics collected during a given
// time period, pending their eventual write to the database.
type statistics struct {
	sync.Mutex
	passed [numSteps]int
	failed [numSteps]int
	errors []actionError
}

// Collector collects the statistics for an action.
type Collector struct {
	action int
	s      *Statistics
}

// Close closes the collector and waits for any collected statistics to be
// stored to the database before returning.
func (c *Collector) Close() {
	c.s.mu.Lock()
	tick := c.s.tick
	for {
		c.s.stored.Wait()
		if c.s.stored.tick == tick {
			break
		}
	}
	delete(c.s.collectors, c.action)
	delete(c.s.stats, c.action)
	c.s.diff[c.action] = false
	c.s.mu.Unlock()
}

// FailedStep increases the failed count for the given step by the given count.
// It is safe to call concurrently from multiple goroutines.
func (c *Collector) FailedStep(step Step, count int, message string) {
	c.failedStep(step, count, message)
}

// FailedReceiving increases the failed count for the Receiving step by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FailedReceiving(count int, message string) {
	c.failedStep(ReceivingStep, count, message)
}

// FailedInputValidation increases the failed count for the InputValidation step
// by the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FailedInputValidation(count int, message string) {
	c.failedStep(InputValidationStep, count, message)
}

// FailedFiltering increases the failed count for the Filtering step by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FailedFiltering(count int, message string) {
	c.failedStep(FilteringStep, count, message)
}

// FailedTransformation increases the failed count for the Transformation step
// by the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FailedTransformation(count int, message string) {
	c.failedStep(TransformationStep, count, message)
}

// FailedOutputValidation increases the failed count for the OutputValidation
// step by the given count. It is safe to call concurrently from multiple
// goroutines.
func (c *Collector) FailedOutputValidation(count int, message string) {
	c.failedStep(OutputValidationStep, count, message)
}

// FailedFinalizing increases the failed count for the Finalizing step by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) FailedFinalizing(count int, message string) {
	c.failedStep(FinalizingStep, count, message)
}

// PassedReceiving increases the passed count for the Receiving step by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) PassedReceiving(count int) {
	c.passedStep(ReceivingStep, count)
}

// PassedInputValidation increases the passed count for the InputValidation step
// by the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) PassedInputValidation(count int) {
	c.passedStep(InputValidationStep, count)
}

// PassedFiltering increases the passed count for the Filtering step by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) PassedFiltering(count int) {
	c.passedStep(FilteringStep, count)
}

// PassedTransformation increases the passed count for the Transformation step
// by the given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) PassedTransformation(count int) {
	c.passedStep(TransformationStep, count)
}

// PassedOutputValidation increases the passed count for the OutputValidation
// step by the given count. It is safe to call concurrently from multiple
// goroutines.
func (c *Collector) PassedOutputValidation(count int) {
	c.passedStep(OutputValidationStep, count)
}

// PassedFinalizing increases the passed count for the Finalizing step by the
// given count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) PassedFinalizing(count int) {
	c.passedStep(FinalizingStep, count)
}

// failedStep increases the failed count for the specified step by the given
// count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) failedStep(step Step, count int, message string) {
	c.s.mu.RLock()
	st := c.s.stats[c.action]
	st.Lock()
	st.failed[step] += count
	st.errors = append(st.errors, actionError{step: step, count: count, message: message})
	st.Unlock()
	c.s.mu.RUnlock()
}

// passedStep increases the passed count for the specified step by the given
// count. It is safe to call concurrently from multiple goroutines.
func (c *Collector) passedStep(step Step, count int) {
	c.s.mu.RLock()
	st := c.s.stats[c.action]
	st.Lock()
	st.passed[step] += count
	st.Unlock()
	c.s.mu.RUnlock()
}

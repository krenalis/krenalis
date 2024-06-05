//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package statistics

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/backoff"

	"github.com/google/uuid"
)

const numSteps = 6

// timeslotBase is the time, in minutes, from the epoch, considered as the
// timeslot 0.
var timeslotBase int64 = 1701388800 / 60 // minutes from epoch to 2023-12-01 00:00:00 UTC

// ExecutionStep represents a step of an execution.
type ExecutionStep int

const (
	ReceivedStep ExecutionStep = iota
	InputValidatedStep
	FilteredStep
	TransformedStep
	OutputValidatedStep
	ConclusiveStep
)

// Collector is a statistics collector.
type Collector struct {
	db         *postgres.DB
	mu         sync.RWMutex
	executions map[int]*ExecutionCollector
	close      struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		shutdown  chan struct{}
		sync.WaitGroup
	}
}

// New returns a new statistics collector. It gathers statistics from the
// execution of actions with a one-minute resolution and saves them to the
// database every minute.
func New(db *postgres.DB) *Collector {

	stats := &Collector{
		db:         db,
		executions: map[int]*ExecutionCollector{},
	}

	stats.close.ctx, stats.close.cancelCtx = context.WithCancel(context.Background())
	stats.close.shutdown = make(chan struct{})

	go stats.process()

	return stats
}

// Execution returns the execution collector to collect statistics for the
// execution with identifier id.
func (c *Collector) Execution(id int) *ExecutionCollector {
	c.mu.Lock()
	defer c.mu.Unlock()
	action, ok := c.executions[id]
	if !ok {
		action = &ExecutionCollector{}
		c.executions[id] = action
	}
	return action
}

// Close closes the collector, ensuring any remaining statistics are stored. If
// the provided context expires before completion, ongoing store execution is
// interrupted and the function returns.
func (c *Collector) Close(ctx context.Context) {
	close(c.close.shutdown)
	stop := context.AfterFunc(ctx, func() { c.close.cancelCtx() })
	defer stop()
	c.close.Wait()
}

// collectedStats represents collected statistics that have to be stored.
type collectedStats struct {
	execution int
	passed    [numSteps]int
	failed    [numSteps]int
}

// process processes the collected statistics.
func (c *Collector) process() {

	now := time.Now().UTC()

	// Wait the time to the next timeslot.
	t := now.Add(time.Second).Truncate(time.Minute)
	time.Sleep(t.Sub(now))

	// Starts the ticker.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var isShuttingDown bool

	// Collect statistics each minute.
	for {
		timeslot := t.Unix()/60 - timeslotBase
		c.mu.Lock()
		if len(c.executions) > 0 {
			data := make([]collectedStats, 0, len(c.executions))
			for id, execution := range c.executions {
				d := collectedStats{}
				execution.mu.Lock()
				isZero := execution.passed == [numSteps]int{} && execution.failed == [numSteps]int{}
				if !isZero {
					d.passed = execution.passed
					d.failed = execution.failed
					execution.passed = [numSteps]int{}
					execution.failed = [numSteps]int{}
				}
				execution.mu.Unlock()
				if !isZero {
					d.execution = id
					data = append(data, d)
				}
			}
			if len(data) > 0 {
				c.close.Add(1)
				go c.saveStats(timeslot, data)
			}
		}
		c.mu.Unlock()
		if isShuttingDown {
			return
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

// saveStats saves the collected statistics data for the provided timeslot into
// the database.
func (c *Collector) saveStats(timeslot int64, data []collectedStats) {

	defer c.close.Done()

	var b strings.Builder
	b.WriteString("WITH t AS (\n\tVALUES ")
	for i, d := range data {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('(')
		b.WriteString(strconv.Itoa(d.execution))
		b.WriteByte(',')
		b.WriteString(strconv.FormatInt(timeslot, 10))
		b.WriteByte(',')
		for j := 0; j < 6; j++ {
			b.WriteString(strconv.Itoa(d.passed[j]))
			b.WriteByte(',')
		}
		for j := 0; j < 6; j++ {
			b.WriteString(strconv.Itoa(d.failed[j]))
			if j != 5 {
				b.WriteByte(',')
			}
		}
		b.WriteByte(')')
	}
	b.WriteString("\n) INSERT INTO actions_executions_stats AS s " +
		`(execution, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)` +
		` SELECT * FROM t ON CONFLICT (execution, timeslot) DO UPDATE SET ` +
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

	query := b.String()

	var errLogged string

	bo := backoff.New(20)
	for bo.Next(c.close.ctx) {
		_, err := c.db.Exec(c.close.ctx, query)
		if err != nil {
			if s := err.Error(); s != errLogged {
				select {
				case <-c.close.ctx.Done():
				default:
					slog.Error("failed to store the statistics on execution executions", "err", s)
				}
			}
			continue
		}
		break
	}

}

// ExecutionCollector collects the statistics for an execution.
type ExecutionCollector struct {
	mu     sync.Mutex
	passed [numSteps]int
	failed [numSteps]int
}

// Failed increases the failed count for the provided step.
func (stats *ExecutionCollector) Failed(step ExecutionStep, gid uuid.UUID, err error) {
	stats.mu.Lock()
	stats.failed[step]++
	stats.mu.Unlock()
}

// Passed increases the passed count for the provided step.
func (stats *ExecutionCollector) Passed(step ExecutionStep) {
	stats.mu.Lock()
	stats.passed[step]++
	stats.mu.Unlock()
}

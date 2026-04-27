// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

var periods = [...]int16{5, 15, 30, 60, 120, 180, 360, 480, 720, 1440}

type scIndex struct {
	i int8
	j int16
}

func periodIndex(period int16) int8 {
	for i, iv := range periods {
		if iv == period {
			return int8(i)
		}
	}
	panic("invalid period")
}

// pipelineScheduler is the pipeline scheduler.
type pipelineScheduler struct {
	core     *Core
	executor *pipelineExecutor
	ctx      context.Context    // context passes to the pipeline runs.
	cancel   context.CancelFunc // function to cancel the pipeline runs.
	wg       sync.WaitGroup     // waiting group that includes the schedulers and pipeline runs.
}

// newPipelineScheduler returns a new pipeline scheduler.
func newPipelineScheduler(core *Core) *pipelineScheduler {
	ps := &pipelineScheduler{
		core: core,
	}
	ps.ctx, ps.cancel = context.WithCancel(context.Background())
	core.state.Freeze()
	core.state.AddListener(ps.onCreatePipeline)
	core.state.AddListener(ps.onDeleteConnection)
	core.state.AddListener(ps.onDeleteOrganization)
	core.state.AddListener(ps.onDeletePipeline)
	core.state.AddListener(ps.onDeleteWorkspace)
	core.state.AddListener(ps.onElectLeader)
	core.state.AddListener(ps.onSetPipelineSchedulePeriod)
	core.state.Unfreeze()
	return ps
}

// Close closes the pipeline scheduler closing the executors and interrupting
// pipeline runs.
func (ps *pipelineScheduler) Close() {
	ps.core.state.Freeze()
	ps.core.state.Unfreeze()
	if ps.executor != nil {
		ps.executor.Close()
	}
	ps.cancel()
	ps.wg.Wait()
}

// onCreatePipeline is called when a pipeline is created.
func (ps *pipelineScheduler) onCreatePipeline(n state.CreatePipeline) {
	if ps.executor == nil {
		return
	}
	pipeline, _ := ps.core.state.Pipeline(n.ID)
	if pipeline.SchedulePeriod == 0 {
		return
	}
	ps.executor.AddPipeline(pipeline)
}

// onDeleteConnection is called when a connection is deleted from the state.
func (ps *pipelineScheduler) onDeleteConnection(n state.DeleteConnection) {
	if ps.executor == nil {
		return
	}
	var pipelines []int
	for _, pipeline := range n.Connection().Pipelines() {
		if pipeline.SchedulePeriod != 0 {
			pipelines = append(pipelines, pipeline.ID)
		}
	}
	if pipelines == nil {
		return
	}
	go func() {
		for _, pipeline := range pipelines {
			ps.executor.RemovePipeline(pipeline)
		}
	}()
}

// onDeleteOrganization is called when an organization is deleted from the
// state.
func (ps *pipelineScheduler) onDeleteOrganization(n state.DeleteOrganization) {
	if ps.executor == nil {
		return
	}
	var pipelines []int
	for _, workspace := range n.Organization().Workspaces() {
		for _, connection := range workspace.Connections() {
			for _, pipeline := range connection.Pipelines() {
				if pipeline.SchedulePeriod != 0 {
					pipelines = append(pipelines, pipeline.ID)
				}
			}
		}
	}
	if pipelines == nil {
		return
	}
	go func() {
		for _, pipeline := range pipelines {
			ps.executor.RemovePipeline(pipeline)
		}
	}()
}

// onDeleteWorkspace is called when a workspace is deleted from the state.
func (ps *pipelineScheduler) onDeleteWorkspace(n state.DeleteWorkspace) {
	if ps.executor == nil {
		return
	}
	var pipelines []int
	for _, connection := range n.Workspace().Connections() {
		for _, pipeline := range connection.Pipelines() {
			if pipeline.SchedulePeriod != 0 {
				pipelines = append(pipelines, pipeline.ID)
			}
		}
	}
	if pipelines == nil {
		return
	}
	go func() {
		for _, pipeline := range pipelines {
			ps.executor.RemovePipeline(pipeline)
		}
	}()
}

// onDeletePipeline is called when a pipeline is deleted from the state.
func (ps *pipelineScheduler) onDeletePipeline(n state.DeletePipeline) {
	if ps.executor == nil {
		return
	}
	go func() {
		ps.executor.RemovePipeline(n.ID)
	}()
}

// onElectLeader is called when a leader is elected.
func (ps *pipelineScheduler) onElectLeader(n state.ElectLeader) {
	if ps.executor != nil {
		if !ps.core.state.IsLeader() {
			go ps.executor.Close()
		}
		return
	}
	if !ps.core.state.IsLeader() {
		return
	}
	ps.executor = newPipelineExecutor(ps.core, &ps.wg, ps.ctx)
}

// onSetPipelineSchedulePeriod is called when the schedule period of a pipeline
// is set.
func (ps *pipelineScheduler) onSetPipelineSchedulePeriod(n state.SetPipelineSchedulePeriod) {
	if ps.executor == nil {
		return
	}
	pipeline, _ := ps.core.state.Pipeline(n.ID)
	ps.executor.SetPeriod(pipeline)
}

// pipelineExecutor handles the actual execution of pipeline runs.
// When a node becomes the leader, it starts its executor; the previous leader
// stops its own executor.
type pipelineExecutor struct {
	core      *Core
	mu        sync.Mutex // for the pipelines and indexes fields.
	pipelines [len(periods)]map[int16][]*state.Pipeline
	indexes   map[int]scIndex
	close     chan struct{}
}

// newPipelineExecutor returns a new pipeline executor. wg is the wait group
// that coordinates goroutines for the executor and pipeline runs. ctx is the
// context to pass to the pipeline runs.
func newPipelineExecutor(core *Core, wg *sync.WaitGroup, ctx context.Context) *pipelineExecutor {

	pe := &pipelineExecutor{
		core:    core,
		indexes: map[int]scIndex{},
		close:   make(chan struct{}),
	}
	for i := range len(pe.pipelines) {
		pe.pipelines[i] = map[int16][]*state.Pipeline{}
	}
	for _, pipeline := range pe.core.state.Pipelines() {
		if pipeline.SchedulePeriod == 0 {
			continue
		}
		i := periodIndex(pipeline.SchedulePeriod)
		j := pipeline.ScheduleStart % pipeline.SchedulePeriod
		pe.pipelines[i][j] = append(pe.pipelines[i][j], pipeline)
		pe.indexes[pipeline.ID] = scIndex{i, j}
	}

	wg.Go(func() {
		ticker := time.NewTicker(time.Minute)
		for {
			select {
			case t := <-ticker.C:
				minute := int16(t.Hour()*60 + t.Minute())
				for i, period := range periods {
					j := minute % period
					pe.mu.Lock()
					pipelines := pe.pipelines[i][j]
					pe.mu.Unlock()
					for _, pipeline := range pipelines {
						if !toExecute(pipeline) {
							continue
						}
						connection := pipeline.Connection()
						store, ok := pe.core.datastore.Store(connection.Workspace().ID)
						if !ok {
							continue
						}
						c := &Connection{core: pe.core, connection: connection, store: store}
						p := &Pipeline{core: pe.core, pipeline: pipeline, connection: c}
						wg.Go(func() {
							_, err := p.createRun(ctx, nil)
							if err != nil {
								if _, ok := err.(*errors.NotFoundError); ok {
									return
								}
								if _, ok := err.(*errors.UnprocessableError); ok {
									return
								}
								slog.Debug("core: cannot add run for pipeline", "pipeline", p.ID, "error", err)
							}
						})
					}
				}
			case <-pe.close:
				return
			}
		}
	})

	return pe
}

// Close stops the executor but does not interrupt any pipeline runs in
// progress.
func (pe *pipelineExecutor) Close() {
	close(pe.close)
}

// AddPipeline adds pipeline to the scheduler executor.
func (pe *pipelineExecutor) AddPipeline(pipeline *state.Pipeline) {
	i := periodIndex(pipeline.SchedulePeriod)
	j := pipeline.ScheduleStart % pipeline.SchedulePeriod
	pe.mu.Lock()
	pe.pipelines[i][j] = append(slices.Clone(pe.pipelines[i][j]), pipeline)
	pe.indexes[pipeline.ID] = scIndex{i, j}
	pe.mu.Unlock()
}

// RemovePipeline removes the pipeline with identifier id from the scheduler
// executor. If the pipeline does not exist it does nothing.
func (pe *pipelineExecutor) RemovePipeline(id int) {
	pe.mu.Lock()
	index, ok := pe.indexes[id]
	if !ok {
		pe.mu.Unlock()
		return
	}
	i, j := index.i, index.j
	pipelines := pe.pipelines[i][j]
	for k, pipeline := range pipelines {
		if pipeline.ID == id {
			pipelines = slices.Delete(pipelines, k, k+1)
			if len(pipelines) == 0 {
				delete(pe.pipelines[i], j)
			} else {
				pe.pipelines[i][j] = pipelines
			}
			break
		}
	}
	pe.mu.Unlock()
}

// SetPeriod sets the period of a pipeline.
func (pe *pipelineExecutor) SetPeriod(pipeline *state.Pipeline) {
	pe.mu.Lock()
	index, ok := pe.indexes[pipeline.ID]
	pe.mu.Unlock()
	if ok {
		if periods[index.i] == pipeline.SchedulePeriod {
			return
		}
		pe.RemovePipeline(pipeline.ID)
	}
	if pipeline.SchedulePeriod != 0 {
		pe.AddPipeline(pipeline)
	}
}

// toExecute reports whether pipeline can be executed.
func toExecute(pipeline *state.Pipeline) bool {
	if !pipeline.Enabled || pipeline.SchedulePeriod == 0 {
		return false
	}
	c := pipeline.Connection()
	if c.Workspace().Warehouse.Mode != state.Normal {
		return false
	}
	if _, ok := pipeline.Run(); ok {
		return false
	}
	return true
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/krenalis/krenalis/core/internal/collector/sender"
	"github.com/krenalis/krenalis/core/internal/connections"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/types"
)

// destinations is responsible for dispatching events to destination apps.
// Use the QueueEvent method to enqueue events for delivery.
type destinations struct {
	state       *state.State
	connections *connections.Connections
	provider    transformers.FunctionProvider
	metrics     *metrics.Collector

	// senders maps a connection ID to its sender.
	// No mutex is needed since all accesses occur while the state is frozen.
	senders map[int]*sender.Sender

	mu        sync.Mutex
	pipelines map[int]destinationPipelines // maps a destination connection ID to its pipelines; it is protected by mu.

	close struct {
		closed    atomic.Bool             // indicates if the writer has been closed
		ctx       context.Context         // context passes to iterators
		cancel    context.CancelCauseFunc // function to cancel iterators executions
		completed sync.Cond               // signal the completion of the current iteration
		iterators sync.WaitGroup          // waiting group for the iterators
	}
}

// newDestinations returns a new destinations instance.
func newDestinations(st *state.State, connections *connections.Connections, provider transformers.FunctionProvider, metrics *metrics.Collector) *destinations {

	d := destinations{
		state:       st,
		connections: connections,
		provider:    provider,
		metrics:     metrics,
		senders:     map[int]*sender.Sender{},
		pipelines:   map[int]destinationPipelines{},
	}
	d.close.ctx, d.close.cancel = context.WithCancelCause(context.Background())

	// Keeps all destination connections whose connector supports events.
	d.state.Freeze()
	d.state.AddListener(d.onCreateConnection)
	d.state.AddListener(d.onCreatePipeline)
	d.state.AddListener(d.onDeleteConnection)
	d.state.AddListener(d.onDeletePipeline)
	d.state.AddListener(d.onDeleteWorkspace)
	d.state.AddListener(d.onSetConnectionSettings)
	d.state.AddListener(d.onSetPipelineStatus)
	d.state.AddListener(d.onUpdatePipeline)
	for _, c := range st.Connections() {
		if c.Role != state.Destination {
			continue
		}
		if !c.Connector().DestinationTargets.Contains(state.TargetEvent) {
			continue
		}
		app := connections.Application(c)
		sender := sender.New(app, d.metrics)
		pipelines := make(destinationPipelines, 0, 1)
		// Keeps all pipelines active on the connection's events.
		for _, p := range c.Pipelines() {
			if !p.Enabled || p.Target != state.TargetEvent {
				continue
			}
			pipeline := d.createDestinationPipeline(p, sender)
			pipelines = append(pipelines, pipeline)
		}
		d.senders[c.ID] = sender
		d.pipelines[c.ID] = pipelines
	}
	d.state.Unfreeze()

	return &d
}

// QueueEvent enqueues the given event to the pipelines of the provided
// connection that are specified in the event.
func (d *destinations) QueueEvent(connection int, event streams.Event) {
	d.mu.Lock()
	pipelines := d.pipelines[connection]
	d.mu.Unlock()
	for _, id := range event.Destinations {
		if p, _ := pipelines.find(id); p != nil {
			p.QueueEvent(event)
		}
	}
}

// createDestinationPipeline creates a destination pipeline for the provided
// pipeline with the provided sender.
func (d *destinations) createDestinationPipeline(pipeline *state.Pipeline, sender *sender.Sender) *destinationPipeline {

	ctx, cancel := context.WithCancelCause(d.close.ctx)

	connection := pipeline.Connection()
	app := d.connections.Application(connection)
	eventTypeSchema, err := app.Schema(ctx, state.TargetEvent, pipeline.EventType)
	if err != nil {
		panic("TODO")
	}

	queue := &destinationPipelineQueue{
		metrics: d.metrics,
		sender:  sender,
		timer:   newStoppedTimer(),
		events:  make([]queuedEvent, 0, minQueuedEventSize),
	}
	queue.cond = sync.NewCond(&queue.mu)
	queue.close.ctx = ctx
	queue.close.cancel = cancel

	go func(connection, pipeline int) {
		done := queue.close.ctx.Done()
		for {
			select {
			case <-queue.timer.C:
				d.mu.Lock()
				pipelines, ok := d.pipelines[connection]
				d.mu.Unlock()
				if !ok {
					continue
				}
				if dp, _ := pipelines.find(pipeline); dp != nil {
					go dp.transform()
				}
			case <-done:
				return
			}
		}
	}(connection.ID, pipeline.ID)

	return newDestinationPipeline(pipeline, eventTypeSchema, d.provider, queue)
}

// onCreateConnection is called when a connection is created.
func (d *destinations) onCreateConnection(n state.CreateConnection) {
	if n.Role != state.Destination {
		return
	}
	c, _ := d.state.Connection(n.ID)
	connector := c.Connector()
	if !connector.DestinationTargets.Contains(state.TargetEvent) {
		return
	}
	app := d.connections.Application(c)
	d.senders[n.ID] = sender.New(app, d.metrics)
	pipelines := make([]*destinationPipeline, 0, 1)
	d.mu.Lock()
	d.pipelines[n.ID] = pipelines
	d.mu.Unlock()
}

// onCreatePipeline is called when a pipeline is created.
func (d *destinations) onCreatePipeline(n state.CreatePipeline) {
	if !n.Enabled || n.Target != state.TargetEvent {
		return
	}
	p, _ := d.state.Pipeline(n.ID)
	c := p.Connection()
	if c.Role != state.Destination {
		return
	}
	// No lock is needed for reading d.senders and d.pipelines since the state is frozen,
	// ensuring there are no concurrent writes.
	pipeline := d.createDestinationPipeline(p, d.senders[c.ID])
	pipelines := d.pipelines[c.ID].append(pipeline)
	d.mu.Lock()
	d.pipelines[c.ID] = pipelines
	d.mu.Unlock()
}

// onDeleteConnection is called when a connection is deleted.
func (d *destinations) onDeleteConnection(n state.DeleteConnection) {
	c := n.Connection()
	if c.Role != state.Destination {
		return
	}
	connector := c.Connector()
	if !connector.DestinationTargets.Contains(state.TargetEvent) {
		return
	}
	delete(d.senders, c.ID)
	pipelines := d.pipelines[c.ID] // No lock needed for reads while the state is frozen.
	d.mu.Lock()
	delete(d.pipelines, c.ID)
	d.mu.Unlock()
	go func() {
		for _, pipeline := range pipelines {
			pipeline.Close(errors.New("connection has been deleted"))
		}
	}()
}

// onDeletePipeline is called when a pipeline is deleted
func (d *destinations) onDeletePipeline(n state.DeletePipeline) {
	p := n.Pipeline()
	if !p.Enabled || p.Target != state.TargetEvent {
		return
	}
	c := p.Connection()
	if c.Role != state.Destination {
		return
	}
	pipelines := d.pipelines[c.ID] // No lock needed for reads while the state is frozen.
	dp, i := pipelines.find(p.ID)
	if dp == nil {
		panic("unexpected missing pipeline")
	}
	pipelines = pipelines.delete(i)
	d.mu.Lock()
	d.pipelines[c.ID] = pipelines
	d.mu.Unlock()
	go dp.Close(errors.New("pipeline has been deleted"))
}

// onDeleteWorkspace is called when a workspace is deleted.
func (d *destinations) onDeleteWorkspace(n state.DeleteWorkspace) {
	ws := n.Workspace()
	var pipelines []*destinationPipeline
	for _, c := range ws.Connections() {
		if c.Role != state.Destination {
			continue
		}
		connector := c.Connector()
		if !connector.DestinationTargets.Contains(state.TargetEvent) {
			continue
		}
		delete(d.senders, c.ID)
		pipelines = append(pipelines, d.pipelines[c.ID]...) // No lock needed for reads while the state is frozen.
		d.mu.Lock()
		delete(d.pipelines, c.ID)
		d.mu.Unlock()
	}
	if len(pipelines) > 0 {
		go func() {
			for _, pipeline := range pipelines {
				pipeline.Close(errors.New("workspace has been deleted"))
			}
		}()
	}
}

// onSetConnectionSettings is called when the settings of a connection is
// changed.
func (d *destinations) onSetConnectionSettings(n state.SetConnectionSettings) {
	sender, ok := d.senders[n.Connection]
	if !ok {
		return
	}
	connection, _ := d.state.Connection(n.Connection)
	app := d.connections.Application(connection)
	sender.SetApplication(app)
}

// onSetPipelineStatus is called when the status of a pipeline is set.
func (d *destinations) onSetPipelineStatus(n state.SetPipelineStatus) {
	p, _ := d.state.Pipeline(n.ID)
	if p.Target != state.TargetEvent {
		return
	}
	c := p.Connection()
	if c.Role != state.Destination {
		return
	}
	pipelines := d.pipelines[c.ID] // No lock needed for reads while the state is frozen.
	if n.Enabled {
		// Add the pipeline.
		pipeline := d.createDestinationPipeline(p, d.senders[c.ID])
		pipelines = pipelines.append(pipeline)
		d.mu.Lock()
		d.pipelines[c.ID] = pipelines
		d.mu.Unlock()
		return
	}
	// Remove the pipeline.
	dp, i := pipelines.find(p.ID)
	if dp == nil {
		panic("unexpected missing pipeline")
	}
	pipelines = pipelines.delete(i)
	d.mu.Lock()
	d.pipelines[c.ID] = pipelines
	d.mu.Unlock()
	go dp.Close(errors.New("pipeline has been disabled"))
}

// onUpdatePipeline is called when a pipeline is updated.
func (d *destinations) onUpdatePipeline(n state.UpdatePipeline) {
	p, _ := d.state.Pipeline(n.ID)
	if p.Target != state.TargetEvent {
		return
	}
	c := p.Connection()
	if c.Role != state.Destination {
		return
	}
	pipelines := d.pipelines[c.ID] // No lock needed for reads while the state is frozen.
	current, index := pipelines.find(p.ID)
	// Removes it if is not enabled but present.
	if !p.Enabled {
		if current != nil {
			pipelines = pipelines.delete(index)
			d.mu.Lock()
			d.pipelines[c.ID] = pipelines
			d.mu.Unlock()
			go current.Close(errors.New("pipeline has been disabled"))
		}
		return
	}
	// Adds it if it wasn't present.
	if current == nil {
		pipeline := d.createDestinationPipeline(p, d.senders[c.ID])
		pipelines = pipelines.append(pipeline)
		d.mu.Lock()
		d.pipelines[c.ID] = pipelines
		d.mu.Unlock()
		return
	}
	// If Filter, or Transformation has changed, replace the destination pipeline
	// with a new version.
	sameFilter := current.filter.Equal(p.Filter)
	sameSchema := types.Equal(current.schema, p.OutSchema)
	sameTransformation := current.transformation.Equal(p.Transformation)
	if sameFilter && sameSchema && sameTransformation {
		return
	}
	pipeline := *current
	if !sameFilter {
		pipeline.filter = p.Filter
	}
	if !sameSchema {
		pipeline.schema = p.OutSchema
	}
	if !sameTransformation {
		t := p.Transformation
		pipeline.transformation = t
		if t.Mapping == nil && t.Function == nil {
			pipeline.transformer = nil
		} else {
			pipeline.transformer, _ = transformers.New(p, d.provider, nil)
		}
	}
	pipelines = pipelines.replace(index, &pipeline)
	d.mu.Lock()
	d.pipelines[c.ID] = pipelines
	d.mu.Unlock()
}

type destinationPipelines []*destinationPipeline

func (pp destinationPipelines) append(pipelines ...*destinationPipeline) destinationPipelines {
	updated := make([]*destinationPipeline, len(pp)+len(pipelines))
	copy(updated, pp)
	copy(updated[len(pp):], pipelines)
	return updated
}

func (pp destinationPipelines) delete(index int) destinationPipelines {
	updated := make([]*destinationPipeline, len(pp)-1)
	copy(updated, pp[:index])
	copy(updated[index:], pp[index+1:])
	return updated
}

func (pp destinationPipelines) find(id int) (*destinationPipeline, int) {
	for i, pipeline := range pp {
		if pipeline.id == id {
			return pipeline, i
		}
	}
	return nil, -1
}

func (pp destinationPipelines) replace(i int, p *destinationPipeline) destinationPipelines {
	updates := make([]*destinationPipeline, len(pp))
	copy(updates, pp)
	updates[i] = p
	return updates
}

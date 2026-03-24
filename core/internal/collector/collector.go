// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/krenalis/krenalis/core/internal/connections"
	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/events"
	"github.com/krenalis/krenalis/core/internal/filters"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/prometheus"
	"github.com/krenalis/krenalis/tools/validation"

	"github.com/oschwald/maxminddb-golang/v2"
)

var errNoStreamConnection = errors.New("no stream connection")

// maxRequestSize is the maximum size inBatchRequests bytes of an event request body.
const maxRequestSize = 500 * 1024

// pipelineWorker represents a long-running pipeline event processor.
type pipelineWorker func()

// Errors handled by the HTTP server of the collector.
var (
	errMethodNotAllowed   = errors.New("method not allowed")
	errNotFound           = errors.NotFound("")
	errServiceUnavailable = errors.Unavailable("")
)

// ValidationError is the interface implemented by validation errors.
type ValidationError interface {
	error
	PropertyPath() string
}

// A Collector collects events, persists them in the database, and sends them to
// apps.
type Collector struct {
	db               *db.DB
	sc               streams.Connection
	state            *state.State
	datastore        *datastore.Datastore
	metrics          *metrics.Collector
	observers        Observers
	duplicated       sync.Map
	functionProvider transformers.FunctionProvider
	maxmindDB        *maxminddb.Reader
	eventWriters     sync.Map      // a map from workspace identifier to a *datastore.EventWriter value
	destinations     *destinations // destination connections used to send events
	identityWriters  sync.Map      // a map from pipeline identifier to a *identityWriter value
	workers          struct {
		cancelPipeline   map[int]context.CancelFunc // maps pipeline IDs to their worker cancel functions
		cancelConnection map[int]context.CancelFunc // maps connection IDs to their worker cancel functions
		sync.WaitGroup                              // wait group used to wait for all workers to exit
	}
	closed atomic.Bool
}

// New returns a new collector.
//
// maxMindDBPath is the path to the MaxMind db file, used for enriching the
// events with geolocation information; if not provided, the database file is
// not opened and the geolocation information are not automatically added by
// Krenalis.
func New(db *db.DB, sc streams.Connection, st *state.State, ds *datastore.Datastore, connections *connections.Connections,
	provider transformers.FunctionProvider, metrics *metrics.Collector, maxMindDBPath string) (*Collector, error) {

	var c = &Collector{
		db:               db,
		sc:               sc,
		state:            st,
		datastore:        ds,
		metrics:          metrics,
		functionProvider: provider,
		destinations:     newDestinations(st, connections, provider, metrics),
	}
	c.workers.cancelPipeline = map[int]context.CancelFunc{}
	c.workers.cancelConnection = map[int]context.CancelFunc{}

	st.Freeze()
	st.AddListener(c.onCreateConnection)
	st.AddListener(c.onCreatePipeline)
	st.AddListener(c.onCreateWorkspace)
	st.AddListener(c.onDeleteConnection)
	st.AddListener(c.onDeletePipeline)
	st.AddListener(c.onDeleteWorkspace)
	st.AddListener(c.onLinkConnection)
	st.AddListener(c.onSetPipelineStatus)
	st.AddListener(c.onUnlinkConnection)
	st.AddListener(c.onUpdatePipeline)
	for _, ws := range st.Workspaces() {
		c.addWorkspace(ws.ID)
	}
	for _, connection := range st.Connections() {
		if connection.LinkedConnections == nil {
			continue
		}
		switch connection.Role {
		case state.Source:
			// There is one worker per active source SDK and webhook pipeline.
			for _, p := range connection.Pipelines() {
				if p.Enabled {
					c.startPipelineWorker(p)
				}
			}
		case state.Destination:
			// There is one worker per destination app connection
			// that has at least one active pipeline sending events
			// and is linked to at least one source connection.
			if len(connection.LinkedConnections) == 0 {
				continue
			}
			for _, p := range connection.Pipelines() {
				if p.Enabled && p.Target == state.TargetEvent {
					c.startConnectionWorker(connection)
					break
				}
			}
		}
	}
	st.Unfreeze()

	if maxMindDBPath != "" {
		var err error
		c.maxmindDB, err = maxminddb.Open(maxMindDBPath)
		if err != nil {
			return nil, fmt.Errorf("cannot open maxmind DB at path %q: %s", maxMindDBPath, err)
		}
	}

	return c, nil
}

// Close closes the collector. When Close is called, no other calls to
// collector's methods should be in progress and no other shall be made.
// It panics if it has already been called.
func (c *Collector) Close(ctx context.Context) {
	if c.closed.Swap(true) {
		panic("core/events/collector already closed")
	}
	for _, cancel := range c.workers.cancelPipeline {
		cancel()
	}
	for _, cancel := range c.workers.cancelConnection {
		cancel()
	}
	c.workers.Wait()
	c.eventWriters.Range(func(_, ew any) bool {
		_ = ew.(*datastore.EventWriter).Close(ctx)
		return true
	})
}

// Observer returns the observer for the given workspace, or nil if the
// workspace does not exist.
// The bool result reports whether the workspace exists.
func (c *Collector) Observer(workspace int) (*Observer, bool) {
	return c.observers.Load(workspace)
}

// ServeHTTP serves both settings and event messages over HTTP.
// A message is a JSON stream of JSON objects where the first object is the
// message header.
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	var serveSettings = strings.HasPrefix(r.URL.Path, "/events/settings/")
	var err error
	if serveSettings {
		err = c.serveSettings(w, r)
	} else {
		// Serve events.
		if r.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				slog.Error("core/events/collector: an error occurred creating gzip reader", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer reader.Close()
			r.Body = http.MaxBytesReader(w, reader, maxRequestSize)
		}
		err = c.serveEvents(w, r)
	}
	if err != nil {
		w.Header().Set("Cache-Control", "no-store, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		if err, ok := err.(errors.ResponseWriterTo); ok {
			_ = err.WriteTo(w)
			return
		}
		switch err {
		case errMethodNotAllowed:
			if serveSettings {
				w.Header().Set("Allow", "GET, OPTIONS")
			} else {
				w.Header().Set("Allow", "POST, OPTIONS")
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		case errNoStreamConnection:
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		case errPayloadTooLarge:
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		case errReadBody:
			// connection already broken, cannot reply to the client
		default:
			if r.Context().Err() != nil {
				// The request context is done; no response should be written.
				return
			}
			if serveSettings {
				slog.Error("core/events/collector: an error occurred serving the settings", "error", err)
			} else {
				slog.Error("core/events/collector: an error occurred collecting an event", "error", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// addWorkspace adds a workspace to the collector.
// It is called from New and onCreateWorkspace.
func (c *Collector) addWorkspace(id int) {
	c.observers.Store(id, newObserver())
	store := c.datastore.Store(id)
	c.eventWriters.Store(id, store.NewEventWriter())
}

// connectionByKey returns the SDK or webhook connection for the key and true
// if found, or nil and false otherwise.
func (c *Collector) connectionByKey(key string) (*state.Connection, bool) {
	connection, ok := c.state.ConnectionByKey(key)
	if !ok || connection.Role != state.Source {
		return nil, false
	}
	return connection, true
}

// onCreateConnection is called when a connection is created.
func (c *Collector) onCreateConnection(n state.CreateConnection) {
	connection, _ := c.state.Connection(n.ID)
	if connection.Role != state.Source {
		return
	}
	for _, id := range connection.LinkedConnections {
		destination, _ := c.state.Connection(id)
		for _, p := range destination.Pipelines() {
			if p.Enabled && p.Target == state.TargetEvent {
				c.startConnectionWorker(destination)
				break
			}
		}
	}
}

// onCreatePipeline is called when a pipeline is created.
func (c *Collector) onCreatePipeline(n state.CreatePipeline) {
	p, _ := c.state.Pipeline(n.ID)
	if !p.Enabled {
		return
	}
	connection := p.Connection()
	if connection.LinkedConnections == nil {
		return
	}
	if connection.Role == state.Source {
		c.startPipelineWorker(p)
		return
	}
	if len(connection.LinkedConnections) == 0 || p.Target != state.TargetEvent {
		return
	}
	c.startConnectionWorker(connection)
}

// onCreateWorkspace is called when a workspace is created.
func (c *Collector) onCreateWorkspace(n state.CreateWorkspace) {
	c.addWorkspace(n.ID)
}

// onDeleteConnection is called when a connection is deleted.
func (c *Collector) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	if connection.LinkedConnections == nil {
		return
	}
	if connection.Role == state.Source {
		for _, p := range connection.Pipelines() {
			if p.Enabled {
				c.stopPipelineWorker(p)
			}
		}
		for _, id := range connection.LinkedConnections {
			destination, _ := c.state.Connection(id)
			if len(destination.LinkedConnections) == 0 {
				c.stopConnectionWorker(connection)
			}
		}
		return
	}
	c.stopConnectionWorker(connection)
}

// onDeletePipeline is called when a pipeline is deleted.
func (c *Collector) onDeletePipeline(n state.DeletePipeline) {
	p := n.Pipeline()
	if !p.Enabled {
		return
	}
	connection := p.Connection()
	if connection.LinkedConnections == nil {
		return
	}
	if connection.Role == state.Source {
		c.stopPipelineWorker(p)
		return
	}
	if len(connection.LinkedConnections) == 0 {
		return
	}
	for _, p := range connection.Pipelines() {
		if p.Enabled && p.Target == state.TargetEvent {
			return
		}
	}
	c.stopConnectionWorker(connection)
}

// onDeleteWorkspace is called when a workspace is deleted.
func (c *Collector) onDeleteWorkspace(n state.DeleteWorkspace) {
	ws := n.Workspace()
	c.observers.Delete(ws.ID)
	for _, connection := range ws.Connections() {
		if connection.LinkedConnections == nil {
			continue
		}
		if connection.Role == state.Source {
			for _, p := range connection.Pipelines() {
				if p.Enabled {
					c.stopPipelineWorker(p)
				}
			}
		} else if len(connection.LinkedConnections) > 0 {
			c.stopConnectionWorker(connection)
		}
	}
	ew, _ := c.eventWriters.LoadAndDelete(ws.ID)
	// Close using a canceled context to abort any in-flight flush.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = ew.(*datastore.EventWriter).Close(ctx)
}

// onLinkConnection is called when two unlinked connections are linked.
func (c *Collector) onLinkConnection(n state.LinkConnection) {
	connection, _ := c.state.Connection(n.Connections[1])
	if len(connection.LinkedConnections) > 1 {
		return
	}
	for _, p := range connection.Pipelines() {
		if p.Enabled && p.Target == state.TargetEvent {
			c.startConnectionWorker(connection)
			break
		}
	}
}

// onSetPipelineStatus is called when the status of a pipeline is set.
func (c *Collector) onSetPipelineStatus(n state.SetPipelineStatus) {
	p, _ := c.state.Pipeline(n.ID)
	connection := p.Connection()
	if connection.LinkedConnections == nil {
		return
	}
	if connection.Role == state.Source {
		if p.Enabled {
			c.startPipelineWorker(p)
		} else {
			c.stopPipelineWorker(p)
		}
		return
	}
	if len(connection.LinkedConnections) == 0 || p.Target != state.TargetEvent {
		return
	}
	for _, p := range connection.Pipelines() {
		if p.Enabled && p.Target == state.TargetEvent {
			c.startConnectionWorker(connection)
			return
		}
	}
	c.stopConnectionWorker(connection)
}

// onUnlinkConnection is called when two linked connections are unlinked.
func (c *Collector) onUnlinkConnection(n state.UnlinkConnection) {
	connection, _ := c.state.Connection(n.Connections[1])
	if len(connection.LinkedConnections) == 0 {
		c.stopConnectionWorker(connection)
	}
}

// onUpdatePipeline is called when a pipeline is updated.
func (c *Collector) onUpdatePipeline(n state.UpdatePipeline) {
	p, _ := c.state.Pipeline(n.ID)
	if p.Enabled {
		// The transformation might have changed.
		if w, ok := c.identityWriters.Load(p.ID); ok {
			var transformer *transformers.Transformer
			if p.Transformation.Mapping != nil || p.Transformation.Function != nil {
				transformer, _ = transformers.New(p, c.functionProvider, nil)
			}
			w.(*identityWriter).SetTransformer(transformer)
		}
	}
	connection := p.Connection()
	if connection.LinkedConnections == nil {
		return
	}
	if connection.Role == state.Source {
		if p.Enabled {
			c.startPipelineWorker(p)
		} else {
			c.stopPipelineWorker(p)
		}
		return
	}
	if len(connection.LinkedConnections) == 0 || p.Target != state.TargetEvent {
		return
	}
	// The pipeline may have been enabled and may no longer be.
	for _, p := range connection.Pipelines() {
		if p.Enabled && p.Target == state.TargetEvent {
			c.startConnectionWorker(connection)
			return
		}
	}
	c.stopConnectionWorker(connection)
}

// processIdentityEvents reads events from the pipeline, extracts identities,
// and persists them using the provided identity writer.
//
// It is called in its own goroutine and runs until the context is canceled.
func (c *Collector) processIdentityEvents(ctx context.Context, w *identityWriter, pipeline int) {
	stream, err := c.sc.Stream(ctx)
	if err != nil {
		return // ctx has been canceled or c.sc has been closed.
	}
	consumer := stream.Consume("pipeline-"+strconv.Itoa(pipeline), 1000)
	events := consumer.Events()
	done := ctx.Done()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				panic("consumer channel was closed before the worker terminated")
			}
			_ = w.Write(event)
		case <-done:
			consumer.Close()
			return
		}
	}
}

// processForwardedEvents reads events from the connection and forwards them to
// its destination pipelines.
//
// It is called in its own goroutine and runs until the context is canceled.
func (c *Collector) processForwardedEvents(ctx context.Context, destinations *destinations, connection int) {
	stream, err := c.sc.Stream(ctx)
	if err != nil {
		return // ctx has been canceled or c.sc has been closed.
	}
	consumer := stream.Consume("connection-"+strconv.Itoa(connection), 1000)
	events := consumer.Events()
	done := ctx.Done()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				panic("consumer channel was closed before the worker terminated")
			}
			destinations.QueueEvent(connection, event)
		case <-done:
			consumer.Close()
			return
		}
	}
}

// processWarehouseEvents processes events for the given pipeline and persists
// them to the data warehouse using the provided event writer.
//
// It is called in its own goroutine and runs until the context is canceled.
func (c *Collector) processWarehouseEvents(ctx context.Context, w *datastore.EventWriter, pipeline int) {
	stream, err := c.sc.Stream(ctx)
	if err != nil {
		return // ctx has been canceled or c.sc has been closed.
	}
	consumer := stream.Consume("pipeline-"+strconv.Itoa(pipeline), 1000)
	events := consumer.Events()
	done := ctx.Done()
	for {
		select {
		case event, ok := <-events:
			if !ok {
				panic("consumer channel was closed before the worker terminated")
			}
			_ = w.Write(ctx, event, pipeline)
		case <-done:
			consumer.Close()
			return
		}
	}
}

// serveEvents is called by the ServeHTTP method to serve an events request.
func (c *Collector) serveEvents(w http.ResponseWriter, r *http.Request) error {

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		w.WriteHeader(204)
		return nil
	}

	// Wait until the stream becomes available.
	// WaitUp returns false if the request context is canceled or if the stream
	// remains unavailable beyond a short, predefined timeout.
	//
	if !c.sc.WaitUp(r.Context()) {
		return errNoStreamConnection
	}

	var err error
	var dec *decoder
	var connection *state.Connection
	var usingWriteKey bool

	if auth, ok := r.Header["Authorization"]; ok {

		// Attempt to read and process the Authorization header.
		if len(auth) > 1 {
			return errors.BadRequest("request contains multiple Authorization headers")
		}
		token, found := validation.ParseBearer(auth[0])
		if !found {
			return errors.BadRequest(`Authorization header is invalid; use "Authorization: Bearer <KEY>" with an API key or an event write key`)
		}

		if token, found := strings.CutPrefix(token, "api_"); found {
			// Authenticate with the API key in the header.
			key, ok := c.state.AccessKeyByToken(token)
			if !ok || key.Type != state.AccessKeyTypeAPI {
				return errors.Unauthorized("API key in the Authorization header is invalid")
			}
			if header, ok := r.Header["Krenalis-Workspace"]; ok {
				if len(header) > 1 {
					return errors.BadRequest(`request contains multiple "Krenalis-Warehouse" headers`)
				}
				if key.Workspace > 0 {
					return errors.BadRequest(`"Krenalis-Workspace" header cannot be provided with a workspace restricted key`)
				}
				var id int64
				if header[0] != "" && header[0][0] != '+' {
					id, _ = strconv.ParseInt(header[0], 10, 32)
				}
				if id <= 0 {
					return errors.BadRequest(`"Krenalis-Workspace" header is invalid; use "Krenalis-Workspace: <WORKSPACE_ID>"`)
				}
				if _, ok = c.state.Workspace(int(id)); !ok {
					return errors.NotFound("workspace %d does not exist", id)
				}
				key.Workspace = int(id)
			}
			// Decode the request.
			dec, err = newDecoder(r)
			if err != nil {
				return err
			}
			if c.maxmindDB != nil {
				dec.SetMaxMindDB(c.maxmindDB)
			}
			// Read the connection.
			id, ok := dec.ConnectionId()
			if !ok {
				return errors.BadRequest("parameter 'connectionId' is required when using API key authentication")
			}
			if key.Workspace == 0 {
				connection, _ = c.state.Connection(id)
			} else {
				workspace, ok := c.state.Workspace(key.Workspace)
				if !ok {
					return errors.Unauthorized("API key in the Authorization header is invalid")
				}
				connection, _ = workspace.Connection(id)
			}
			if connection == nil {
				return errors.Unprocessable("ConnectionNotExist", "connection %d does not exist", id)
			}

		} else {
			// Authenticate with the event write key in the header.
			connection, _ = c.connectionByKey(token)
			if connection == nil {
				return errors.Unauthorized("event write key in the Authorization header is not valid")
			}
			usingWriteKey = true
		}

	}

	// Decode the request if it hasn't been decoded already.
	if dec == nil {
		dec, err = newDecoder(r)
		if err != nil {
			return err
		}
		if c.maxmindDB != nil {
			dec.SetMaxMindDB(c.maxmindDB)
		}
	}

	// Authenticate using the event write key in the body.
	if connection == nil {
		// Get the connection from the write key.
		writeKey := dec.WriteKey()
		if writeKey == "" {
			return errors.Unauthorized("Authorization header is missing")
		}
		connection, _ = c.connectionByKey(writeKey)
		if connection == nil {
			return errors.Unauthorized("the event write key in the request body does not exist")
		}
		usingWriteKey = true
	}

	if usingWriteKey {
		if _, ok := dec.ConnectionId(); ok {
			return errors.BadRequest("property 'connectionId' cannot be provided when using an event write key for authentication")
		}
	}

	ws := connection.Workspace()
	connector := connection.Connector()
	pipelines := connection.Pipelines()
	observer, _ := c.observers.Load(ws.ID)

	stream, err := c.sc.Stream(r.Context())
	if err != nil {
		return errNoStreamConnection
	}
	batch := stream.Batch()

	var topics []string
	var destinations []int

	var observedEvents []events.Event

	// Decode the events.
	for event, err := range dec.Events(connection.ID, connector.FallbackToRequestIP) {

		prometheus.Increment("Collector.serveEvents.decoded_events", 1)
		if err != nil {
			continue
		}

		_, duplicated := c.duplicated.LoadOrStore(event["messageId"].(string), nil)
		if duplicated {
			continue
		}

		if observer != nil {
			observedEvents = append(observedEvents, event)
		}

		topics = topics[0:0]
		destinations = destinations[0:0]

		// Store the events into the data warehouse.
		for _, p := range pipelines {
			if !p.Enabled || p.Target != state.TargetEvent {
				continue
			}
			c.metrics.ReceivePassed(p.ID, 1)
			if !filters.Applies(p.Filter, event) {
				c.metrics.FilterFailed(p.ID, 1)
				continue
			}
			c.metrics.FilterPassed(p.ID, 1)
			if _, ok := c.eventWriters.Load(ws.ID); ok {
				topics = append(topics, "pipeline-"+strconv.Itoa(p.ID))
			}
		}

		// Import the identities into the data warehouse.
		for _, p := range pipelines {
			if !p.Enabled || p.Target != state.TargetUser {
				continue
			}
			c.metrics.ReceivePassed(p.ID, 1)
			if !filters.Applies(p.Filter, event) {
				c.metrics.FilterFailed(p.ID, 1)
				continue
			}
			c.metrics.FilterPassed(p.ID, 1)
			if _, ok := c.identityWriters.Load(p.ID); ok {
				topics = append(topics, "pipeline-"+strconv.Itoa(p.ID))
			}
		}

		// Send the event to apps.
		for _, id := range connection.LinkedConnections {
			lc, ok := c.state.Connection(id)
			if !ok {
				continue
			}
			for _, p := range lc.Pipelines() {
				if !p.Enabled || p.Target != state.TargetEvent {
					continue
				}
				c.metrics.ReceivePassed(p.ID, 1)
				if !filters.Applies(p.Filter, event) {
					c.metrics.FilterFailed(p.ID, 1)
					continue
				}
				c.metrics.FilterPassed(p.ID, 1)
				destinations = append(destinations, p.ID)
			}
			if len(destinations) > 0 {
				topics = append(topics, "connection-"+strconv.Itoa(id))
			}
		}

		if len(topics) == 0 {
			continue
		}

		// Publish the event.
		err = batch.Publish(topics, event, destinations)
		if err != nil {
			return err
		}

	}

	err = batch.Done(r.Context())
	if err != nil {
		return err
	}

	for _, event := range observedEvents {
		observer.addEvent(event)
	}

	// Send a successful response to the client.
	prometheus.Increment("Collector.writeOK.calls", 1)
	w.Header().Set("Content-Type", "text/plain")

	return nil
}

// serveSettings is called by the ServeHTTP method to serve a settings request.
func (c *Collector) serveSettings(w http.ResponseWriter, r *http.Request) error {
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Max-Age", "900")
		w.Header().Set("Cache-Control", "public, max-age=900, immutable")
		w.WriteHeader(204)
		return nil
	}
	if r.Method != "GET" {
		return errMethodNotAllowed
	}
	writeKey, _ := strings.CutPrefix(r.URL.Path, "/events/settings/")
	if writeKey == "" || strings.Contains(writeKey, "/") {
		return errNotFound
	}
	connection, ok := c.connectionByKey(writeKey)
	if !ok || connection.Strategy == nil {
		return errors.Unauthorized("event write key in the path is invalid")
	}
	strategy := string(*connection.Strategy)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=10800")
	_ = json.Encode(w, map[string]any{
		"strategy": strategy,
		"integrations": map[string]any{
			"Krenalis": map[string]any{
				"apiKey": writeKey,
			},
		},
	})
	return nil
}

// startConnectionWorker starts a worker for the given connection if one does
// not already exist.
//
// It must be called with the state frozen.
func (c *Collector) startConnectionWorker(connection *state.Connection) {
	if _, ok := c.workers.cancelConnection[connection.ID]; ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.workers.cancelConnection[connection.ID] = cancel
	c.workers.Go(func() {
		c.processForwardedEvents(ctx, c.destinations, connection.ID)
	})
}

// startPipelineWorker starts a worker for the given pipeline if one does not
// already exist.
//
// It must be called with the state frozen.
func (c *Collector) startPipelineWorker(pipeline *state.Pipeline) {
	if _, ok := c.workers.cancelPipeline[pipeline.ID]; ok {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.workers.cancelPipeline[pipeline.ID] = cancel
	switch pipeline.Target {
	case state.TargetUser:
		// Import the identity into the data warehouse.
		iw := newIdentityWriter(c.datastore, pipeline, c.functionProvider, c.metrics)
		c.identityWriters.Store(pipeline.ID, iw)
		c.workers.Go(func() {
			c.processIdentityEvents(ctx, iw, pipeline.ID)
		})
	case state.TargetEvent:
		// Store the event into the data warehouse.
		ws := pipeline.Connection().Workspace()
		ew, _ := c.eventWriters.Load(ws.ID)
		c.workers.Go(func() {
			c.processWarehouseEvents(ctx, ew.(*datastore.EventWriter), pipeline.ID)
		})
	default:
		panic("unreachable")
	}
}

// stopConnectionWorker stops the worker associated with the given connection.
// If no worker exists, it does nothing.
// It must be called with the state frozen.
func (c *Collector) stopConnectionWorker(connection *state.Connection) {
	if cancel, ok := c.workers.cancelConnection[connection.ID]; ok {
		cancel()
		delete(c.workers.cancelConnection, connection.ID)
	}
}

// stopPipelineWorker stops the worker associated with the given pipeline.
// If no worker exists, it does nothing.
// It must be called with the state frozen.
func (c *Collector) stopPipelineWorker(pipeline *state.Pipeline) {
	cancel, ok := c.workers.cancelPipeline[pipeline.ID]
	if !ok {
		return
	}
	cancel()
	delete(c.workers.cancelPipeline, pipeline.ID)
	if pipeline.Target == state.TargetUser {
		iw, _ := c.identityWriters.LoadAndDelete(pipeline.ID)
		go func() {
			_ = iw.(*identityWriter).Close(context.Background())
		}()
	}
}

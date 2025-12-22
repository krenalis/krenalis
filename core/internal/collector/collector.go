// Copyright 2025 Open2b. All rights reserved.
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

	"github.com/meergo/meergo/core/internal/connections"
	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/filters"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/tools/errors"
	"github.com/meergo/meergo/tools/json"
	meergoMetrics "github.com/meergo/meergo/tools/metrics"
	"github.com/meergo/meergo/tools/validation"

	"github.com/oschwald/maxminddb-golang/v2"
)

// maxRequestSize is the maximum size inBatchRequests bytes of an event request body.
const maxRequestSize = 500 * 1024

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
	closed           atomic.Bool
}

// New returns a new collector.
//
// maxMindDBPath is the path to the MaxMind db file, used for enriching the
// events with geolocation information; if not provided, the database file is
// not opened and the geolocation information are not automatically added by
// Meergo.
func New(db *db.DB, st *state.State, ds *datastore.Datastore, connections *connections.Connections,
	provider transformers.FunctionProvider, metrics *metrics.Collector, maxMindDBPath string) (*Collector, error) {
	var c = &Collector{
		db:               db,
		state:            st,
		datastore:        ds,
		metrics:          metrics,
		functionProvider: provider,
		destinations:     newDestinations(st, connections, provider, metrics),
	}
	st.Freeze()
	st.AddListener(c.onCreatePipeline)
	st.AddListener(c.onCreateWorkspace)
	st.AddListener(c.onDeleteConnection)
	st.AddListener(c.onDeletePipeline)
	st.AddListener(c.onDeleteWorkspace)
	st.AddListener(c.onSetPipelineStatus)
	st.AddListener(c.onUpdatePipeline)
	for _, ws := range st.Workspaces() {
		c.observers.Store(ws.ID, newObserver())
		store := ds.Store(ws.ID)
		c.eventWriters.Store(ws.ID, store.NewEventWriter(c.eventAck))
	}
	for _, pipeline := range st.Pipelines() {
		// Create an identity writer for each active SDK pipeline.
		if pipeline.Target == state.TargetUser {
			if !pipeline.Enabled {
				continue
			}
			if t := pipeline.Connection().Connector().Type; t != state.SDK && t != state.Webhook {
				continue
			}
			iw := newIdentityWriter(c.datastore, pipeline, provider, metrics)
			c.identityWriters.Store(pipeline.ID, iw)
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
func (c *Collector) Close() {
	if c.closed.Swap(true) {
		panic("core/events/collector already closed")
	}
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
				slog.Error("core/events/collector: an error occurred creating gzip reader", "err", err)
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
		case errPayloadTooLarge:
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		case errReadBody:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		default:
			if serveSettings {
				slog.Error("core/events/collector: an error occurred serving the settings", "err", err)
			} else {
				slog.Error("core/events/collector: an error occurred collecting an event", "err", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// connectionByKey returns the SDK or webhook connection for the key and true
// if found, or nil and false otherwise.
func (c *Collector) connectionByKey(key string) (*state.Connection, bool) {
	conn, ok := c.state.ConnectionByKey(key)
	if !ok || conn.Role != state.Source {
		return nil, false
	}
	t := conn.Connector().Type
	if t == state.SDK || t == state.Webhook {
		return conn, true
	}
	return nil, false
}

// eventAck acknowledges when an event is written to the data warehouse.
func (c *Collector) eventAck(evs []datastore.AckEvent, err error) {
	meergoMetrics.Increment("Collector.eventAck.calls", 1)
	for _, event := range evs {
		if err != nil {
			c.metrics.FinalizeFailed(event.Pipeline, 1, err.Error())
			return
		}
		c.metrics.FinalizePassed(event.Pipeline, 1)
	}
}

// importEventsPipeline returns the pipeline of the source connection that
// imports events into the data warehouse, if there is one and if it is enabled;
// otherwise, it returns nil and false.
func (c *Collector) importEventsPipeline(connection *state.Connection) (*state.Pipeline, bool) {
	for _, p := range connection.Pipelines() {
		if p.Enabled && p.Target == state.TargetEvent {
			return p, true
		}
	}
	return nil, false
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
			"Meergo": map[string]any{
				"apiKey": writeKey,
			},
		},
	})
	return nil
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
				return errors.Unauthorized(`the API key in the Authorization header does not exist`)
			}
			if header, ok := r.Header["Meergo-Workspace"]; ok {
				if len(header) > 1 {
					return errors.BadRequest(`request contains multiple "Meergo-Warehouse" headers`)
				}
				if key.Workspace > 0 {
					return errors.BadRequest(`"Meergo-Workspace" header cannot be provided with a workspace restricted key`)
				}
				var id int64
				if header[0] != "" && header[0][0] != '+' {
					id, _ = strconv.ParseInt(header[0], 10, 32)
				}
				if id <= 0 {
					return errors.BadRequest(`"Meergo-Workspace" header is invalid; use "Meergo-Workspace: <WORKSPACE_ID>"`)
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
					return errors.Unauthorized("the API key in the Authorization header does not exist")
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
				return errors.Unauthorized("the event write key in the Authorization header does not exist")
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
	observer, _ := c.observers.Load(ws.ID)

	var eventErr error

	// Decode the events.
	for event, err := range dec.Events(connection.ID, connector.FallbackToRequestIP) {

		meergoMetrics.Increment("Collector.serveEvents.decoded_events", 1)
		if err != nil {
			if eventErr == nil {
				eventErr = err
			}
			continue
		}
		if observer != nil {
			observer.addEvent(event)
		}

		_, duplicated := c.duplicated.LoadOrStore(event["messageId"].(string), nil)
		if duplicated {
			continue
		}

		// Store the events into the data warehouse.
		if pipeline, ok := c.importEventsPipeline(connection); ok {
			c.metrics.ReceivePassed(pipeline.ID, 1)
			if !filters.Applies(pipeline.Filter, event) {
				c.metrics.FilterFailed(pipeline.ID, 1)
				continue
			}
			c.metrics.FilterPassed(pipeline.ID, 1)
			ew, ok := c.eventWriters.Load(ws.ID)
			if !ok {
				continue
			}
			err = ew.(*datastore.EventWriter).Write(event, pipeline.ID)
			if err != nil {
				c.metrics.FinalizeFailed(pipeline.ID, 1, err.Error())
				if eventErr == nil {
					eventErr = errServiceUnavailable
				}
				continue
			}
		}

		// Import the identities into the data warehouse.
		for _, pipeline := range connection.Pipelines() {
			if pipeline.Target != state.TargetUser || !pipeline.Enabled {
				continue
			}
			c.metrics.ReceivePassed(pipeline.ID, 1)
			if !filters.Applies(pipeline.Filter, event) {
				c.metrics.FilterFailed(pipeline.ID, 1)
				meergoMetrics.Increment("Collector.serveEvents.discarded_identities", 1)
				continue
			}
			c.metrics.FilterPassed(pipeline.ID, 1)
			if w, ok := c.identityWriters.Load(pipeline.ID); ok {
				err = w.(*identityWriter).Write(event)
				if err != nil {
					c.metrics.FinalizeFailed(pipeline.ID, 1, err.Error())
					if eventErr == nil {
						eventErr = errServiceUnavailable
					}
					continue
				}
			}
		}

		// Send the event to apps.
		for _, id := range connection.LinkedConnections {
			c.destinations.QueueEvent(id, event)
		}

	}

	if eventErr != nil {
		return eventErr
	}

	// Send a successful response to the client.
	meergoMetrics.Increment("Collector.writeOK.calls", 1)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", "21")
	_, _ = io.WriteString(w, "{\n  \"success\": true\n}")

	return nil
}

// onCreatePipeline is called when a pipeline is created.
func (c *Collector) onCreatePipeline(n state.CreatePipeline) {
	pipeline, _ := c.state.Pipeline(n.ID)
	if pipeline.Target != state.TargetUser || !pipeline.Enabled {
		return
	}
	if t := pipeline.Connection().Connector().Type; t != state.SDK && t != state.Webhook {
		return
	}
	iw := newIdentityWriter(c.datastore, pipeline, c.functionProvider, c.metrics)
	c.identityWriters.Store(pipeline.ID, iw)
}

// onCreateWorkspace is called when a workspace is created.
func (c *Collector) onCreateWorkspace(n state.CreateWorkspace) {
	c.observers.Store(n.ID, newObserver())
	store := c.datastore.Store(n.ID)
	c.eventWriters.Store(n.ID, store.NewEventWriter(c.eventAck))
}

// onDeleteConnection is called when a connection is deleted.
func (c *Collector) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	if t := connection.Connector().Type; t != state.SDK && t != state.Webhook {
		return
	}
	for _, pipeline := range connection.Pipelines() {
		if pipeline.Target != state.TargetUser || !pipeline.Enabled {
			continue
		}
		iw, ok := c.identityWriters.LoadAndDelete(pipeline.ID)
		if !ok {
			continue
		}
		_ = iw.(*identityWriter).Close(context.Background())
	}
}

// onDeleteWorkspace is called when a workspace is deleted.
func (c *Collector) onDeleteWorkspace(n state.DeleteWorkspace) {
	c.observers.Delete(n.ID)
	c.eventWriters.Delete(n.ID)
}

// onDeletePipeline is called when a pipeline is deleted.
func (c *Collector) onDeletePipeline(n state.DeletePipeline) {
	iw, ok := c.identityWriters.LoadAndDelete(n.ID)
	if !ok {
		return
	}
	_ = iw.(*identityWriter).Close(context.Background())
	// TODO(marco): should the ongoing transformations be interrupted?
}

// onSetPipelineStatus is called when the status of a pipeline is set.
func (c *Collector) onSetPipelineStatus(n state.SetPipelineStatus) {
	pipeline, _ := c.state.Pipeline(n.ID)
	if pipeline.Target != state.TargetUser {
		return
	}
	connection := pipeline.Connection()
	if t := connection.Connector().Type; t != state.SDK && t != state.Webhook {
		return
	}
	if pipeline.Enabled {
		iw := newIdentityWriter(c.datastore, pipeline, c.functionProvider, c.metrics)
		c.identityWriters.Store(pipeline.ID, iw)
		return
	}
	p, _ := c.identityWriters.LoadAndDelete(n.ID)
	p.(*identityWriter).Close(context.Background())
}

// onUpdatePipeline is called when a pipeline is updated.
func (c *Collector) onUpdatePipeline(n state.UpdatePipeline) {
	pipeline, _ := c.state.Pipeline(n.ID)
	if pipeline.Target != state.TargetUser {
		return
	}
	connection := pipeline.Connection()
	if t := connection.Connector().Type; t != state.SDK && t != state.Webhook {
		return
	}
	if !pipeline.Enabled {
		if iw, ok := c.identityWriters.LoadAndDelete(pipeline.ID); ok {
			iw.(*identityWriter).Close(context.Background())
		}
		return
	}
	w, ok := c.identityWriters.Load(pipeline.ID)
	if !ok {
		iw := newIdentityWriter(c.datastore, pipeline, c.functionProvider, c.metrics)
		c.identityWriters.Store(pipeline.ID, iw)
		return
	}
	// The transformation might have changed.
	var transformer *transformers.Transformer
	if pipeline.Transformation.Mapping != nil || pipeline.Transformation.Function != nil {
		transformer, _ = transformers.New(pipeline, c.functionProvider, nil)
	}
	w.(*identityWriter).SetTransformer(transformer)
	// TODO(marco): how does changing the warehouse mode affect the source pipeline?
}

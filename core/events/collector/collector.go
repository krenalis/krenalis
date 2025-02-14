//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package collector

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/events/dispatcher"
	"github.com/meergo/meergo/core/filters"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/json"
	meergoMetrics "github.com/meergo/meergo/metrics"

	"github.com/google/uuid"
	"github.com/oschwald/maxminddb-golang"
)

// maxRequestSize is the maximum size inBatchRequests bytes of an event request body.
const maxRequestSize = 500 * 1024

const maxmindDBPath = "GeoLite2-City.mmdb"

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

// actionIdentityWriter represents an identity writer for an action.
type actionIdentityWriter struct {
	id          int //action identifier
	writer      *datastore.EventIdentityWriter
	mu          sync.Mutex // for transformer, records, and timer.
	transformer *transformers.Transformer
	identities  []events.Event
	timer       *time.Timer
}

// newActionIdentityWriter returns a new actionIdentityWriter.
func newActionIdentityWriter(ds *datastore.Datastore, action *state.Action, provider transformers.Provider, ack datastore.EventIdentityWriterAckFunc) *actionIdentityWriter {
	sa := &actionIdentityWriter{id: action.ID}
	ws := action.Connection().Workspace()
	store := ds.Store(ws.ID)
	sa.writer, _ = store.NewEventIdentityWriter(action.ID, ack)
	if t := action.Transformation; t.Mapping != nil || t.Function != nil {
		sa.transformer, _ = transformers.New(action, provider, nil)
	}
	return sa
}

// Close closes sa.
func (sa *actionIdentityWriter) Close(ctx context.Context) error {
	if sa.timer != nil {
		sa.timer.Stop()
		sa.timer = nil
	}
	return sa.writer.Close(ctx)
}

// A Collector collects events, persists them in the database and sends them to
// the dispatcher.
type Collector struct {
	db                  *postgres.DB
	state               *state.State
	datastore           *datastore.Datastore
	operationStore      events.OperationStore
	metrics             *metrics.Collector
	observer            *Observer
	duplicated          sync.Map
	transformerProvider transformers.Provider
	dispatcher          *dispatcher.Dispatcher
	maxmindDB           *maxminddb.Reader
	eventWriters        sync.Map // a map from workspace identifier to a *datastore.EventWriter value
	actions             sync.Map // a map from action identifier to a *actionIdentityWriter value
	closed              atomic.Bool
}

// New returns a new event collector. It receives HTTP requests from mobile,
// server and website sources and sends them to the dispatcher.
func New(db *postgres.DB, st *state.State, ds *datastore.Datastore, opStore events.OperationStore, provider transformers.Provider, dispatcher *dispatcher.Dispatcher, metrics *metrics.Collector) (*Collector, error) {
	var c = &Collector{
		db:                  db,
		state:               st,
		datastore:           ds,
		operationStore:      opStore,
		metrics:             metrics,
		observer:            newObserver(db),
		transformerProvider: provider,
		dispatcher:          dispatcher,
	}
	st.Freeze()
	st.AddListener(c.onCreateAction)
	st.AddListener(c.onCreateWorkspace)
	st.AddListener(c.onDeleteAction)
	st.AddListener(c.onDeleteConnection)
	st.AddListener(c.onDeleteWorkspace)
	st.AddListener(c.onSetActionStatus)
	st.AddListener(c.onUpdateAction)
	for _, ws := range st.Workspaces() {
		store := ds.Store(ws.ID)
		c.eventWriters.Store(ws.ID, store.NewEventWriter(c.eventAck))
	}
	for _, action := range st.Actions() {
		if action.Target != state.Users || !action.Enabled {
			continue
		}
		if action.Connection().Keys == nil {
			continue
		}
		sa := newActionIdentityWriter(c.datastore, action, provider, c.identityAck)
		c.actions.Store(action.ID, sa)
	}
	st.Unfreeze()

	var err error
	c.maxmindDB, err = maxminddb.Open(maxmindDBPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot open maxmind DB at path %q: %s", maxmindDBPath, err)
	}

	c.reloadEvents(context.Background())

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

// Observer returns the observer for the collected events.
func (c *Collector) Observer() *Observer {
	return c.observer
}

// ServeHTTP serves both settings and event messages over HTTP.
// A message is a JSON stream of JSON objects where the first object is the
// message header.
func (c *Collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()
	var serveSettings = strings.HasPrefix(r.URL.Path, "/events/settings/")
	var err error
	if serveSettings {
		err = c.serveSettings(w, r)
	} else {
		// Serve events.
		if r.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				slog.Error("collector: an error occurred creating gzip reader", "err", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer reader.Close()
			r.Body = http.MaxBytesReader(w, reader, maxRequestSize)
		}
		err = c.serveEvents(w, r)
	}
	if err != nil {
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
				slog.Error("collector: an error occurred serving the settings", "err", err)
			} else {
				slog.Error("collector: an error occurred collecting an event", "err", err)
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// identityAck acknowledges when identities are written to the data warehouse.
func (c *Collector) identityAck(action int, ids []string, err error) {
	doneEvents := make([]events.DoneEvent, len(ids))
	for i, id := range ids {
		doneEvents[i].Action = action
		doneEvents[i].ID = id
	}
	c.operationStore.Done(doneEvents...)
	if err != nil {
		c.metrics.FinalizeFailed(action, len(ids), err.Error())
		return
	}
	c.metrics.FinalizePassed(action, len(ids))
}

// connectionByKey returns an enable source mobile, server or website connection
// given its key and true, if exists, otherwise returns nil and false.
func (c *Collector) connectionByKey(key string) (*state.Connection, bool) {
	conn, ok := c.state.ConnectionByKey(key)
	if ok && conn.Role == state.Source {
		switch conn.Connector().Type {
		case state.Mobile, state.Server, state.Website:
			return conn, true
		}
	}
	return nil, false
}

// eventAck acknowledges when an event is written to the data warehouse.
func (c *Collector) eventAck(evs []datastore.AckEvent, err error) {
	meergoMetrics.Increment("Collector.eventAck.calls", 1)
	doneEvents := make([]events.DoneEvent, len(evs))
	for i, event := range evs {
		doneEvents[i] = events.DoneEvent(event)
	}
	c.operationStore.Done(doneEvents...)
	for _, event := range evs {
		if err != nil {
			c.metrics.FinalizeFailed(event.Action, 1, err.Error())
			return
		}
		c.metrics.FinalizePassed(event.Action, 1)
	}
}

// eventDestinations returns the destination actions to which events from source
// can be dispatched.
func (c *Collector) eventDestinations(connection *state.Connection) []*state.Action {
	var actions []*state.Action
	for _, id := range connection.LinkedConnections {
		c, ok := c.state.Connection(id)
		if !ok {
			continue
		}
		for _, action := range c.Actions() {
			if action.Enabled && action.Target == state.Events {
				actions = append(actions, action)
			}
		}
	}
	return actions
}

// importEventsAction returns the action of the source connection that imports
// events into the data warehouse, if there is one and if it is enabled;
// otherwise, it returns nil and false.
func (c *Collector) importEventsAction(connection *state.Connection) (*state.Action, bool) {
	for _, a := range connection.Actions() {
		if a.Enabled && a.Target == state.Events {
			return a, true
		}
	}
	return nil, false
}

// reloadEvents reloads the operations from the operation store.
func (c *Collector) reloadEvents(ctx context.Context) {
	operations, errf := c.operationStore.Pending(ctx)
	for op := range operations {
		for _, actionID := range op.Actions {
			action, ok := c.state.Action(actionID)
			if !ok {
				continue
			}
			if action.Target == state.Users {
				// Import the user identities into the data warehouse
				_ = c.writeIdentity(action, op.Event)
				continue
			}
			connection := action.Connection()
			if connection.Role == state.Source {
				// Store the events into the data warehouse.
				ew, ok := c.eventWriters.Load(connection.Workspace().ID)
				if !ok {
					continue
				}
				_ = ew.(*datastore.EventWriter).Write(op.Event, action.ID)
				continue
			}
			// Send the events to destinations.
			_ = c.dispatcher.Dispatch(op.Event, action)
		}
	}
	if err := errf(); err != nil {
		slog.Error("error occurred reloading events", "err", err)
	}
}

// serveSettings is called by the ServeHTTP method to serve a settings request.
func (c *Collector) serveSettings(w http.ResponseWriter, r *http.Request) error {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Max-Age", "900")
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
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Header().Set("Cache-Control", "max-age=31536000")
		w.WriteHeader(http.StatusNotFound)
		// Do not modify the returned body, as it is used by the JavaScript SDK
		// to present an appropriate error message in the console.
		_, _ = io.WriteString(w, `error: invalid collect key`)
		return nil
	}
	strategy := string(*connection.Strategy)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=10800")
	w.Header().Set("Access-Control-Allow-Origin", origin)
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

	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")
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
		token, ok := strings.CutPrefix(auth[0], "Bearer ")
		if !ok || token == "" {
			return errors.BadRequest(`Authorization header is invalid; use "Authorization: Bearer <KEY>" with an API key or an event write key`)
		}

		if len(token) == 43 {
			// Authenticate with the API key in the header.
			key, ok := c.state.APIKeyByToken(token)
			if !ok {
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
			dec, err = newDecoder(r, c.skip)
			if err != nil {
				return err
			}
			// Read the connection.
			id, ok := dec.Connection()
			if !ok {
				return errors.BadRequest("parameter 'connection' is required when using API key authentication")
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
		dec, err = newDecoder(r, c.skip)
		if err != nil {
			return err
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
		if _, ok := dec.Connection(); ok {
			return errors.BadRequest("property 'connection' cannot be provided when using an event write key for authentication")
		}
	}

	connectionType := connection.Connector().Type
	ws := connection.Workspace()

	var eventErr error

	var operations []events.PendingOperation

	// Decode the events.
	for event, err := range dec.Events(connection.ID, connectionType) {
		meergoMetrics.Increment("Collector.serveEvents.decoded_events", 1)

		if err != nil {
			if eventErr == nil {
				eventErr = err
			}
			continue
		}

		c.observer.addEvent(event)

		var actions []int

		// Store the events into the data warehouse.
		if action, ok := c.importEventsAction(connection); ok {
			c.metrics.ReceivePassed(action.ID, 1)
			if !filters.Applies(action.Filter, event) {
				c.metrics.FilterFailed(action.ID, 1)
				continue
			}
			c.metrics.FilterPassed(action.ID, 1)
			ew, ok := c.eventWriters.Load(ws.ID)
			if !ok {
				continue
			}
			err = ew.(*datastore.EventWriter).Write(event, action.ID)
			if err != nil {
				c.metrics.FinalizeFailed(action.ID, 1, err.Error())
				if eventErr == nil {
					eventErr = errServiceUnavailable
				}
				continue
			} else {
				actions = append(actions, action.ID)
			}
		}

		// Import the user identities into the data warehouse
		eventContext := event["context"].(map[string]any)
		if event["type"] == "identify" || eventContext["traits"] != nil {
			for _, action := range connection.Actions() {
				if action.Target != state.Users || !action.Enabled {
					continue
				}
				c.metrics.ReceivePassed(action.ID, 1)
				if !filters.Applies(action.Filter, event) {
					c.metrics.FilterFailed(action.ID, 1)
					meergoMetrics.Increment("Collector.serveEvents.discarded_user_identitites", 1)
					continue
				}
				c.metrics.FilterPassed(action.ID, 1)
				err = c.writeIdentity(action, event)
				if err != nil {
					c.metrics.FinalizeFailed(action.ID, 1, err.Error())
					if eventErr == nil {
						eventErr = errServiceUnavailable
					}
					continue
				}
				actions = append(actions, action.ID)
			}
		}

		// Send the events to destinations.
		for _, action := range c.eventDestinations(connection) {
			c.metrics.ReceivePassed(action.ID, 1)
			if !filters.Applies(action.Filter, event) {
				c.metrics.FilterFailed(action.ID, 1)
				meergoMetrics.Increment("Collector.serveEvents.events_to_dispatched.filter_failed", 1)
				continue
			}
			c.metrics.FilterPassed(action.ID, 1)
			meergoMetrics.Increment("Collector.serveEvents.events_to_dispatched.filter_passed", 1)
			err = c.dispatcher.Dispatch(event, action)
			if err != nil {
				c.metrics.FinalizeFailed(action.ID, 1, err.Error())
				if eventErr == nil {
					eventErr = errServiceUnavailable
				}
				continue
			}
			c.metrics.FinalizePassed(action.ID, 1)
			actions = append(actions, action.ID)
		}

		if actions != nil {
			operations = append(operations, events.PendingOperation{Event: event, Actions: actions})
		}

	}

	// Persist the events.
	if len(operations) > 0 {
		err = c.operationStore.Store(context.Background(), operations)
		if err != nil && eventErr == nil {
			eventErr = errServiceUnavailable
		}
	}

	if eventErr != nil {
		return eventErr
	}

	// Send a successful response to the client.
	writeOK(w, origin)

	return nil
}

// skip reports if the event with the provided identifier should be skipped
// because is duplicated.
func (c *Collector) skip(id uuid.UUID) bool {
	_, skip := c.duplicated.LoadOrStore(id, nil)
	return skip
}

// onCreateAction is called when an action is created.
func (c *Collector) onCreateAction(n state.CreateAction) {
	action, _ := c.state.Action(n.ID)
	if action.Target != state.Users || !action.Enabled {
		return
	}
	if action.Connection().Keys == nil {
		return
	}
	go func() {
		sa := newActionIdentityWriter(c.datastore, action, c.transformerProvider, c.identityAck)
		c.actions.Store(action.ID, sa)
	}()
}

// onCreateWorkspace is called when a workspace is created.
func (c *Collector) onCreateWorkspace(n state.CreateWorkspace) {
	store := c.datastore.Store(n.ID)
	c.eventWriters.Store(n.ID, store.NewEventWriter(c.eventAck))
}

// onDeleteAction is called when an action is deleted.
func (c *Collector) onDeleteAction(n state.DeleteAction) {
	sa, ok := c.actions.LoadAndDelete(n.ID)
	if !ok {
		return
	}
	_ = sa.(*actionIdentityWriter).Close(context.Background())
	// TODO(marco): should the ongoing transformations be interrupted?
}

// onDeleteConnection is called when a connection is deleted.
func (c *Collector) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	if connection.Keys == nil {
		return
	}
	for _, action := range connection.Actions() {
		if action.Target != state.Users || !action.Enabled {
			continue
		}
		sa, ok := c.actions.LoadAndDelete(action.ID)
		if !ok {
			continue
		}
		_ = sa.(*actionIdentityWriter).Close(context.Background())
	}
}

// onDeleteWorkspace is called when a workspace is deleted.
func (c *Collector) onDeleteWorkspace(n state.DeleteWorkspace) {
	c.eventWriters.Delete(n.ID)
}

// onSetActionStatus is called when the status of an action is set.
func (c *Collector) onSetActionStatus(n state.SetActionStatus) {
	action, _ := c.state.Action(n.ID)
	if action.Target != state.Users {
		return
	}
	connection := action.Connection()
	if connection.Keys == nil {
		return
	}
	if action.Enabled {
		go func() {
			sa := newActionIdentityWriter(c.datastore, action, c.transformerProvider, c.identityAck)
			c.actions.Store(action.ID, sa)
		}()
		return
	}
	if a, ok := c.actions.LoadAndDelete(n.ID); ok {
		_ = a.(*actionIdentityWriter).Close(context.Background())
	}
}

// onUpdateAction is called when an action is updated.
func (c *Collector) onUpdateAction(n state.UpdateAction) {
	action, _ := c.state.Action(n.ID)
	if action.Target != state.Users {
		return
	}
	connection := action.Connection()
	if connection.Keys == nil {
		return
	}
	if !action.Enabled {
		if a, ok := c.actions.LoadAndDelete(action.ID); ok {
			_ = a.(*actionIdentityWriter).Close(context.Background())
		}
		return
	}
	a, ok := c.actions.Load(action.ID)
	if !ok {
		go func() {
			sa := newActionIdentityWriter(c.datastore, action, c.transformerProvider, c.identityAck)
			c.actions.Store(action.ID, sa)
		}()
		return
	}
	// The transformation might have changed.
	sa := a.(*actionIdentityWriter)
	sa.mu.Lock()
	if action.Transformation.Mapping == nil && action.Transformation.Function == nil {
		sa.transformer = nil
	} else {
		sa.transformer, _ = transformers.New(action, c.transformerProvider, nil)
	}
	sa.mu.Unlock()
	// TODO(marco): il cambio del warehouse mode come influisce sulla source action?
}

// Send a successful response to the client.
func writeOK(w http.ResponseWriter, origin string) {
	meergoMetrics.Increment("Collector.writeOK.calls", 1)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", "21")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	_, _ = io.WriteString(w, "{\n  \"success\": true\n}")
}

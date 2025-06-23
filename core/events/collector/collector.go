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

	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/filters"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/json"
	meergoMetrics "github.com/meergo/meergo/metrics"

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

// A Collector collects events, persists them in the database, and sends them to
// apps.
type Collector struct {
	db               *db.DB
	state            *state.State
	datastore        *datastore.Datastore
	metrics          *metrics.Collector
	observer         *Observer
	duplicated       sync.Map
	functionProvider transformers.FunctionProvider
	maxmindDB        *maxminddb.Reader
	eventWriters     sync.Map      // a map from workspace identifier to a *datastore.EventWriter value
	destinations     *destinations // destination connections used to send events
	identityWriters  sync.Map      // a map from action identifier to a *identityWriter value
	closed           atomic.Bool
}

// New returns a new collector.
func New(db *db.DB, st *state.State, ds *datastore.Datastore, connectors *connectors.Connectors, provider transformers.FunctionProvider, metrics *metrics.Collector) (*Collector, error) {
	var c = &Collector{
		db:               db,
		state:            st,
		datastore:        ds,
		metrics:          metrics,
		observer:         newObserver(db),
		functionProvider: provider,
		destinations:     newDestinations(st, connectors, provider, metrics),
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
		// Create an identity writer for each active SDK action.
		if action.Target == state.TargetUser {
			if !action.Enabled {
				continue
			}
			if action.Connection().Connector().Type != state.SDK {
				continue
			}
			iw := newIdentityWriter(c.datastore, action, provider, metrics)
			c.identityWriters.Store(action.ID, iw)
		}
	}
	st.Unfreeze()

	var err error
	c.maxmindDB, err = maxminddb.Open(maxmindDBPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot open maxmind DB at path %q: %s", maxmindDBPath, err)
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

// connectionByKey returns a source SDK connection given its key and true, if
// exists, otherwise returns nil and false.
func (c *Collector) connectionByKey(key string) (*state.Connection, bool) {
	conn, ok := c.state.ConnectionByKey(key)
	if ok && conn.Role == state.Source && conn.Connector().Type == state.SDK {
		return conn, true
	}
	return nil, false
}

// eventAck acknowledges when an event is written to the data warehouse.
func (c *Collector) eventAck(evs []datastore.AckEvent, err error) {
	meergoMetrics.Increment("Collector.eventAck.calls", 1)
	for _, event := range evs {
		if err != nil {
			c.metrics.FinalizeFailed(event.Action, 1, err.Error())
			return
		}
		c.metrics.FinalizePassed(event.Action, 1)
	}
}

// importEventsAction returns the action of the source connection that imports
// events into the data warehouse, if there is one and if it is enabled;
// otherwise, it returns nil and false.
func (c *Collector) importEventsAction(connection *state.Connection) (*state.Action, bool) {
	for _, a := range connection.Actions() {
		if a.Enabled && a.Target == state.TargetEvent {
			return a, true
		}
	}
	return nil, false
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
			dec, err = newDecoder(r)
			if err != nil {
				return err
			}
			if c.maxmindDB != nil {
				dec.SetMaxMindDB(c.maxmindDB)
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
		if _, ok := dec.Connection(); ok {
			return errors.BadRequest("property 'connection' cannot be provided when using an event write key for authentication")
		}
	}

	connectionType := connection.Connector().Type
	ws := connection.Workspace()

	var eventErr error

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

		_, duplicated := c.duplicated.LoadOrStore(event["id"].(string), nil)
		if duplicated {
			continue
		}

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
			}
		}

		// Import the user identities into the data warehouse
		eventContext := event["context"].(map[string]any)
		if event["type"] == "identify" || eventContext["traits"] != nil {
			for _, action := range connection.Actions() {
				if action.Target != state.TargetUser || !action.Enabled {
					continue
				}
				c.metrics.ReceivePassed(action.ID, 1)
				if !filters.Applies(action.Filter, event) {
					c.metrics.FilterFailed(action.ID, 1)
					meergoMetrics.Increment("Collector.serveEvents.discarded_user_identitites", 1)
					continue
				}
				c.metrics.FilterPassed(action.ID, 1)
				if w, ok := c.identityWriters.Load(action.ID); ok {
					err = w.(*identityWriter).Write(event)
					if err != nil {
						c.metrics.FinalizeFailed(action.ID, 1, err.Error())
						if eventErr == nil {
							eventErr = errServiceUnavailable
						}
						continue
					}
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
	writeOK(w, origin)

	return nil
}

// onCreateAction is called when an action is created.
func (c *Collector) onCreateAction(n state.CreateAction) {
	action, _ := c.state.Action(n.ID)
	if action.Target != state.TargetUser || !action.Enabled {
		return
	}
	if action.Connection().Connector().Type != state.SDK {
		return
	}
	go func() {
		iw := newIdentityWriter(c.datastore, action, c.functionProvider, c.metrics)
		c.identityWriters.Store(action.ID, iw)
	}()
}

// onCreateWorkspace is called when a workspace is created.
func (c *Collector) onCreateWorkspace(n state.CreateWorkspace) {
	store := c.datastore.Store(n.ID)
	c.eventWriters.Store(n.ID, store.NewEventWriter(c.eventAck))
}

// onDeleteAction is called when an action is deleted.
func (c *Collector) onDeleteAction(n state.DeleteAction) {
	iw, ok := c.identityWriters.LoadAndDelete(n.ID)
	if !ok {
		return
	}
	_ = iw.(*identityWriter).Close(context.Background())
	// TODO(marco): should the ongoing transformations be interrupted?
}

// onDeleteConnection is called when a connection is deleted.
func (c *Collector) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	if connection.Connector().Type != state.SDK {
		return
	}
	for _, action := range connection.Actions() {
		if action.Target != state.TargetUser || !action.Enabled {
			continue
		}
		iw, ok := c.identityWriters.LoadAndDelete(action.ID)
		if !ok {
			continue
		}
		_ = iw.(*identityWriter).Close(context.Background())
	}
}

// onDeleteWorkspace is called when a workspace is deleted.
func (c *Collector) onDeleteWorkspace(n state.DeleteWorkspace) {
	c.eventWriters.Delete(n.ID)
}

// onSetActionStatus is called when the status of an action is set.
func (c *Collector) onSetActionStatus(n state.SetActionStatus) {
	action, _ := c.state.Action(n.ID)
	if action.Target != state.TargetUser {
		return
	}
	connection := action.Connection()
	if connection.Connector().Type != state.SDK {
		return
	}
	if action.Enabled {
		go func() {
			iw := newIdentityWriter(c.datastore, action, c.functionProvider, c.metrics)
			c.identityWriters.Store(action.ID, iw)
		}()
		return
	}
	if a, ok := c.identityWriters.LoadAndDelete(n.ID); ok {
		_ = a.(*identityWriter).Close(context.Background())
	}
}

// onUpdateAction is called when an action is updated.
func (c *Collector) onUpdateAction(n state.UpdateAction) {
	action, _ := c.state.Action(n.ID)
	if action.Target != state.TargetUser {
		return
	}
	connection := action.Connection()
	if connection.Connector().Type != state.SDK {
		return
	}
	if !action.Enabled {
		if iw, ok := c.identityWriters.LoadAndDelete(action.ID); ok {
			_ = iw.(*identityWriter).Close(context.Background())
		}
		return
	}
	w, ok := c.identityWriters.Load(action.ID)
	if !ok {
		go func() {
			iw := newIdentityWriter(c.datastore, action, c.functionProvider, c.metrics)
			c.identityWriters.Store(action.ID, iw)
		}()
		return
	}
	// The transformation might have changed.
	var transformer *transformers.Transformer
	if action.Transformation.Mapping != nil || action.Transformation.Function != nil {
		transformer, _ = transformers.New(action, c.functionProvider, nil)
	}
	w.(*identityWriter).SetTransformer(transformer)
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

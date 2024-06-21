//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package collector

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/culture"
	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/events/dispatcher"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/statistics"
	"github.com/open2b/chichi/apis/transformers"

	"github.com/google/uuid"
	"github.com/mssola/useragent"
	"github.com/oschwald/maxminddb-golang"
	"github.com/relvacode/iso8601"
)

// maxRequestSize is the maximum size inBatchRequests bytes of an event request body.
const maxRequestSize = 500 * 1024

const maxmindDBPath = "GeoLite2-City.mmdb"

// Errors handled by the HTTP server of the collector.
var (
	errBadRequest         = errors.New("bad request")
	errMethodNotAllowed   = errors.New("method not allowed")
	errNotFound           = errors.New("not found")
	errServiceUnavailable = errors.New("service unavailable")
	errUnauthorized       = errors.New("unauthorized")
)

// ValidationError is the interface implemented by validation errors.
type ValidationError interface {
	error
	PropertyPath() string
}

type batchEvents struct {
	Batch    []*collectedEvent `json:"batch"`
	Context  *events.Context   `json:"context,omitempty"`
	SentAt   string            `json:"sentAt,omitempty"`
	WriteKey string            `json:"writeKey,omitempty"`
}

// collectedEvent represents an event as collected from a client.
type collectedEvent struct {
	header *events.Header

	id [20]byte

	AnonymousId  string          `json:"anonymousId,omitempty"`
	Category     string          `json:"category,omitempty"`
	Context      events.Context  `json:"context,omitempty"`
	Event        string          `json:"event,omitempty"`
	GroupId      string          `json:"groupId,omitempty"`
	Integrations json.RawMessage `json:"integrations,omitempty"`
	MessageId    string          `json:"messageId,omitempty"`
	Name         string          `json:"name,omitempty"`
	receivedAt   time.Time
	SentAt       string `json:"sentAt,omitempty"`
	sentAt       time.Time
	Timestamp    string `json:"timestamp,omitempty"`
	timestamp    time.Time
	Traits       map[string]any `json:"traits,omitempty"`
	Type         *string        `json:"type"`
	UserId       string         `json:"userId,omitempty"`
	PreviousId   string         `json:"previousId,omitempty"`
	Properties   map[string]any `json:"properties,omitempty"`

	WriteKey string `json:"writeKey,omitempty"`
}

// A Collector collects events, persists them in the database and sends them to
// the dispatcher.
type Collector struct {
	db                  *postgres.DB
	state               *state.State
	datastore           *datastore.Datastore
	statistics          *statistics.Collector
	observer            *Observer
	messageIds          sync.Map
	transformerProvider transformers.Provider
	dispatcher          *dispatcher.Dispatcher
	maxmindDB           *maxminddb.Reader
}

// New returns a new event collector. It receives HTTP requests from mobile,
// server and website sources and sends them to the dispatcher.
func New(db *postgres.DB, st *state.State, ds *datastore.Datastore, provider transformers.Provider, dispatcher *dispatcher.Dispatcher, stats *statistics.Collector) (*Collector, error) {
	var collector = Collector{
		db:                  db,
		state:               st,
		datastore:           ds,
		statistics:          stats,
		observer:            newObserver(db),
		messageIds:          sync.Map{},
		transformerProvider: provider,
		dispatcher:          dispatcher,
	}
	var err error
	collector.maxmindDB, err = maxminddb.Open(maxmindDBPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot open maxmind DB at path %q: %s", maxmindDBPath, err)
	}
	return &collector, nil
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
	var serveSettings = strings.HasPrefix(r.URL.Path, "/api/v1/projects/")
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
		switch err {
		case errBadRequest:
			http.Error(w, "Bad batchRequest", http.StatusBadRequest)
		case errMethodNotAllowed:
			if serveSettings {
				w.Header().Set("Allow", "GET, OPTIONS")
			} else {
				w.Header().Set("Allow", "POST, OPTIONS")
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		case errNotFound:
			http.Error(w, "Not Found", http.StatusNotFound)
		case errServiceUnavailable:
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		case errUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

// actions returns the app destination actions that are enabled, have the Events
// target, and their connection is enabled.
func (c *Collector) actions() []*state.Action {
	var actions []*state.Action
	for _, action := range c.state.Actions() {
		if !action.Enabled || action.Target != state.Events {
			continue
		}
		c := action.Connection()
		if !c.Enabled || c.Role != state.Destination || c.Connector().Type != state.AppType {
			continue
		}
		actions = append(actions, action)
	}
	return actions
}

// canCollectEvents reports whether the provided connection can collect events.
// It can collect events if it is enabled and has an enabled action, or is
// enabled and has an enabled event destination with an enabled action on
// events.
func (c *Collector) canCollectEvents(connection *state.Connection) bool {
	return connection.Enabled && (c.hasImportEventsAction(connection) ||
		c.hasImportUsersAction(connection) || c.hasEventDestinations(connection))
}

// connectionByKey returns an enable source mobile, server or website connection
// given its key and true, if exists, otherwise returns nil and false.
func (c *Collector) connectionByKey(key string) (*state.Connection, bool) {
	conn, ok := c.state.ConnectionByKey(key)
	if ok && conn.Enabled && conn.Role == state.Source {
		switch conn.Connector().Type {
		case state.MobileType, state.ServerType, state.WebsiteType:
			return conn, true
		}
	}
	return nil, false
}

// HasEventDestinations reports whether source has an enabled event destination
// with an enabled action on events.
func (c *Collector) hasEventDestinations(source *state.Connection) bool {
	for _, id := range source.EventConnections {
		c, ok := c.state.Connection(id)
		if !ok || !c.Enabled {
			continue
		}
		for _, action := range c.Actions() {
			if action.Enabled && action.Target == state.Events {
				return true
			}
		}
	}
	return false
}

// HasImportEventsAction reports whether source has an enabled action that
// import the events.
func (c *Collector) hasImportEventsAction(source *state.Connection) bool {
	for _, a := range source.Actions() {
		if a.Enabled && a.Target == state.Events {
			return true
		}
	}
	return false
}

// HasImportUsersAction reports whether source has an enabled action that
// import the users.
func (c *Collector) hasImportUsersAction(source *state.Connection) bool {
	for _, a := range source.Actions() {
		if a.Enabled && a.Target == state.Users {
			return true
		}
	}
	return false
}

// importUserIdentities imports user identities from the given events batch
// collected on the source connection.
//
// If the data warehouse is in inspection mode, it returns the
// datastore.ErrInspectionMode error. If it is in maintenance mode, it returns
// the datastore.ErrMaintenanceMode error.
func (c *Collector) importUserIdentities(source *state.Connection, events []*events.Event) error {
	for _, action := range source.Actions() {
		if !action.Enabled {
			continue
		}
		if action.Target != state.Users {
			continue
		}
		connection := action.Connection()
		ws := connection.Workspace()
		store := c.datastore.Store(ws.ID)
		stats := c.statistics.Action(action.ID)

		// Instantiate an event identity writer for writing the user identities.
		ctx := context.Background()
		iw, err := store.EventIdentityWriter(action.ID, func(ids []string, err error) {
			if err != nil {
				stats.FailedCount(statistics.Finalizing, len(ids), err.Error())
				return
			}
			stats.PassedCount(statistics.Finalizing, len(ids))
		})
		if err != nil {
			return err
		}
		defer iw.Close(ctx)

		// Import the user identities from the events batch.
		for _, event := range events {
			if *event.Type != "identify" {
				continue
			}
			mapEvent := event.ToMap()
			ok, err := filterApplies(action.Filter, mapEvent)
			if err != nil || !ok {
				continue
			}
			var properties map[string]any
			// If the action specifies mappings, apply them to the event and
			// obtain the properties.
			if m := action.Transformation.Mapping; m != nil {
				transformation := state.Transformation{Mapping: m}
				transformer := transformers.New(action.InSchema, action.OutSchema, transformation, action.ID, c.transformerProvider, nil)
				properties, err = transformer.Transform(ctx, mapEvent)
				if err != nil {
					if _, ok := err.(ValidationError); ok {
						stats.Passed(statistics.Transformation)
						stats.Failed(statistics.OutputValidation, err.Error())
						continue
					}
					stats.Failed(statistics.Transformation, err.Error())
					continue
				}
				stats.Passed(statistics.Transformation)
				stats.Passed(statistics.OutputValidation)
			}
			// Discard anonymous events with no properties.
			if event.UserId == "" && len(properties) == 0 {
				continue
			}
			// Write the user identity on the data warehouse.
			err = iw.Write(datastore.Identity{
				ID:             event.UserId,
				AnonymousID:    event.AnonymousId,
				Properties:     properties,
				LastChangeTime: event.Timestamp,
			}, event.MessageId)
			if err != nil {
				return iw.Close(ctx)
			}
		}
		err = iw.Close(ctx)
		if err != nil {
			return err
		}
		// Run the Identity Resolution.
		err = store.RunIdentityResolution(ctx)
		if err != nil {
			if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
				return err
			}
			return fmt.Errorf("cannot run the Identity Resolution: %s", err)
		}
	}
	return nil
}

// persistEvents persists events in the database.
func (c *Collector) persistEvents(events []*collectedEvent) <-chan error {
	ack := make(chan error, 1)
	go func() {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		for _, event := range events {
			header := event.header
			remoteAddr, _, _ := net.SplitHostPort(header.RemoteAddr)
			_ = enc.Encode(event)
			payload := b.Bytes()
			_, err := c.db.Exec(context.Background(), "INSERT INTO event_payloads (id, connection, received_at, remote_addr, user_agent, payload) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING",
				event.id[:], header.Connection, header.ReceivedAt, remoteAddr, header.Headers.Get("User-Agent"), payload)
			if err != nil {
				ack <- err
				return
			}
			b.Reset()
		}
		ack <- nil
	}()
	return ack
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
	path, ok := strings.CutPrefix(r.URL.Path, "/api/v1/projects/")
	if !ok {
		return errNotFound
	}
	writeKey, ok := strings.CutSuffix(path, "/settings")
	if !ok || strings.Contains(writeKey, "/") {
		return errNotFound
	}
	source, ok := c.connectionByKey(writeKey)
	if !ok || source.Strategy == nil {
		w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
		w.Header().Set("Cache-Control", "max-age=31536000")
		w.WriteHeader(http.StatusNotFound)
		// Do not modify the returned body, as it is used by the JavaScript SDK
		// to present an appropriate error message in the console.
		_, _ = io.WriteString(w, `error: invalid write key`)
		return nil
	}
	strategy := string(*source.Strategy)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600, stale-while-revalidate=10800")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	keyName, ok := os.LookupEnv("CHICHI_EVENT_SETTINGS_NAME")
	if !ok {
		keyName = "Chichi"
	}
	_ = enc.Encode(map[string]any{
		"strategy": strategy,
		"integrations": map[string]any{
			keyName: map[string]any{
				"apiKey": writeKey,
			},
		},
	})
	return nil
}

// serveEvents is called by the ServeHTTP method to serve an events request.
func (c *Collector) serveEvents(w http.ResponseWriter, r *http.Request) error {

	date := time.Now().UTC()
	if y := date.Year(); y < 1 || y > 9999 {
		slog.Error(fmt.Sprintf("detected a critical clock synchronization issue. Clock is %s", date.Format(time.RFC3339Nano)))
		os.Exit(1)
	}

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
	if r.Method != "POST" {
		return errMethodNotAllowed
	}

	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	switch method {
	case "batch", "b":
		method = "batch"
	case "alias", "group", "identify", "page", "screen", "track":
	default:
		return errNotFound
	}

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || (mt != "application/json" && mt != "text/plain") || len(params) > 1 {
		return errBadRequest
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		return errBadRequest
	}

	// Validate the content length.
	if cl := r.Header.Get("Content-Length"); cl != "" {
		length, _ := strconv.Atoi(cl)
		if length < 1 || length > maxRequestSize {
			return errBadRequest
		}
	}

	header := &events.Header{
		ReceivedAt: date,
		RemoteAddr: r.RemoteAddr,
		Method:     r.Method,
		Proto:      r.Proto,
		URL:        r.URL.String(),
		Headers:    collectHeader(r),
	}

	// Parse the request's body.
	evs, err := parse(r.Body, method == "batch")
	if err != nil {
		return errBadRequest
	}

	// Validate the write key.
	var connection *state.Connection
	{
		writeKey := evs.WriteKey
		if method != "batch" {
			writeKey = evs.Batch[0].WriteKey
		}
		if writeKey == "" {
			if key, _, ok := r.BasicAuth(); ok {
				writeKey = key
			}
		}
		if writeKey != "" {
			connection, _ = c.connectionByKey(writeKey)
		}
		if connection == nil {
			writeOK(w, origin)
			return nil
		}
	}
	header.Connection = connection.ID

	// Assign an identifier to each event concatenating the connection with the message ID.
	var id bytes.Buffer
	for _, event := range evs.Batch {
		id.WriteString(strconv.Itoa(connection.ID))
		id.WriteRune(':')
		id.WriteString(event.MessageId)
		h := sha1.New()
		_, _ = id.WriteTo(h)
		copy(event.id[:], h.Sum(nil))
		id.Reset()
	}

	// Discard duplicated events.
	evs.Batch = c.removeDuplicatedEvents(evs.Batch)
	if len(evs.Batch) == 0 {
		writeOK(w, origin)
		return nil
	}

	for i := 0; i < len(evs.Batch); i++ {
		event := evs.Batch[i]
		if event.Type == nil && method != "batch" {
			event.Type = &method
		}
		if event.SentAt == "" {
			event.SentAt = evs.SentAt
		}
		event.header = header
		mergeContexts(&event.Context, evs.Context)
		err = validateEvent(method, event)
		c.observer.addEvent(header.Connection, event, err)
		if err != nil {
			// Remove the invalid event.
			evs.Batch = slices.Delete(evs.Batch, i, i+1)
			c.setEventAsReceived(event)
			i--
			continue
		}
	}
	if len(evs.Batch) == 0 {
		return nil
	}

	if !c.canCollectEvents(connection) {
		c.setEventsAsReceived(evs.Batch)
		writeOK(w, origin)
		return nil
	}

	// Add the events to the database.
	ack := c.persistEvents(evs.Batch)

	// Enrich the events.
	for _, event := range evs.Batch {
		c.enrichEvent(event)
	}

	// Wait for the events to be logged.
	if err = <-ack; err != nil {
		return err
	}

	// Set the events as received.
	c.setEventsAsReceived(evs.Batch)

	// Send a successful response to the client.
	writeOK(w, origin)

	batch := make([]*events.Event, len(evs.Batch))
	for i, event := range evs.Batch {
		batch[i] = &events.Event{
			Header:       event.header,
			Id:           event.id,
			AnonymousId:  event.AnonymousId,
			Category:     event.Category,
			Context:      event.Context,
			Event:        event.Event,
			GroupId:      event.GroupId,
			Integrations: event.Integrations,
			MessageId:    event.MessageId,
			Name:         event.Name,
			ReceivedAt:   event.receivedAt,
			SentAt:       event.sentAt,
			Timestamp:    event.timestamp,
			Traits:       event.Traits,
			Type:         event.Type,
			UserId:       event.UserId,
			PreviousId:   event.PreviousId,
			Properties:   event.Properties,
		}
	}

	if c.hasImportEventsAction(connection) {
		// Store the events into the data warehouse.
		err = c.storeEvents(connection.Workspace().ID, batch)
		if err != nil {
			if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
				err = errServiceUnavailable
			}
			return err
		}
	}

	if c.hasImportUsersAction(connection) {
		// Import the user identities.
		err = c.importUserIdentities(connection, batch)
		if err != nil {
			if err == datastore.ErrInspectionMode || err == datastore.ErrMaintenanceMode {
				err = errServiceUnavailable
			}
			return err
		}
	}

	if c.hasEventDestinations(connection) {
		// Send the events to the dispatcher.
		for _, event := range batch {
			for _, action := range c.actions() {
				eventAsMap := event.ToMap()
				ok, err := filterApplies(action.Filter, eventAsMap)
				if err != nil || !ok {
					continue
				}
				c.dispatcher.Dispatch(event, action)
			}
		}
	}

	return nil
}

// validateEvent validates the given method and event and returns an error if
// they are not valid. method can be "alias", "identify", "group", "page",
// "screen", "track", or "batch".
func validateEvent(method string, event *collectedEvent) error {

	// Type.
	if event.Type == nil {
		if method == "batch" {
			return errors.New("missing event type")
		}
		event.Type = &method
	} else if method != "batch" && *event.Type != method {
		return errors.New("invalid event type")
	}
	typ := *event.Type
	var (
		isAlias    = typ == "alias"
		isIdentify = typ == "identify"
		isGroup    = typ == "group"
		isPage     = typ == "page"
		isScreen   = typ == "screen"
		isTrack    = typ == "track"
	)

	// AnonymousId and UserId.
	if event.AnonymousId == "" && event.UserId == "" {
		if isIdentify || isAlias {
			return errors.New("missing event userId")
		}
		return errors.New("missing event anonymousId")
	}

	// Category.
	if event.Category != "" && !isPage {
		return errors.New("unexpected event category")
	}

	// EventI.
	if event.Event == "" && isTrack {
		return errors.New("missing event name")
	}
	if event.Event != "" && !isTrack {
		return errors.New("unexpected event name")
	}

	// GroupId.
	if event.GroupId == "" && isGroup {
		return errors.New("missing event group")
	}
	if event.GroupId != "" && !isGroup {
		return errors.New("unexpected event group")
	}

	// Name.
	if event.Name != "" && !isScreen && !isPage {
		return errors.New("unexpected screen or page name")
	}

	// PreviousId.
	if event.PreviousId == "" && isAlias {
		return errors.New("missing event previousId")
	}
	if event.PreviousId != "" && !isAlias {
		return errors.New("unexpected event previousId")
	}

	// Properties.
	if event.Properties != nil && !isPage && !isScreen && !isTrack {
		return errors.New("unexpected event properties")
	}

	// Traits.
	if event.Traits != nil {
		if !isIdentify && !isGroup {
			return errors.New("unexpected event traits")
		}
	}

	return nil
}

// mergeContexts merges defaultCtx into ctx.
func mergeContexts(ctx, defaultCtx *events.Context) {
	if defaultCtx == nil {
		return
	}
	// App.
	if ctx.App.Name == "" {
		ctx.App.Name = defaultCtx.App.Name
	}
	if ctx.App.Version == "" {
		ctx.App.Name = defaultCtx.App.Version
	}
	if ctx.App.Build == "" {
		ctx.App.Build = defaultCtx.App.Build
	}
	if ctx.App.Namespace == "" {
		ctx.App.Namespace = defaultCtx.App.Namespace
	}
	// Campaign.
	if ctx.Campaign.Name == "" {
		ctx.Campaign.Name = defaultCtx.Campaign.Name
	}
	if ctx.Campaign.Source == "" {
		ctx.Campaign.Source = defaultCtx.Campaign.Source
	}
	if ctx.Campaign.Medium == "" {
		ctx.Campaign.Medium = defaultCtx.Campaign.Medium
	}
	if ctx.Campaign.Term == "" {
		ctx.Campaign.Term = defaultCtx.Campaign.Term
	}
	if ctx.Campaign.Content == "" {
		ctx.Campaign.Content = defaultCtx.Campaign.Content
	}
	// Device.
	if ctx.Device.Id == "" {
		ctx.Device.Id = defaultCtx.Device.Id
	}
	if ctx.Device.AdvertisingId == "" {
		ctx.Device.AdvertisingId = defaultCtx.Device.AdvertisingId
	}
	if !ctx.Device.AdTrackingEnabled {
		ctx.Device.AdTrackingEnabled = defaultCtx.Device.AdTrackingEnabled
	}
	if ctx.Device.Manufacturer == "" {
		ctx.Device.Manufacturer = defaultCtx.Device.Manufacturer
	}
	if ctx.Device.Model == "" {
		ctx.Device.Model = defaultCtx.Device.Model
	}
	if ctx.Device.Name == "" {
		ctx.Device.Name = defaultCtx.Device.Name
	}
	if ctx.Device.Type == "" {
		ctx.Device.Type = defaultCtx.Device.Type
	}
	if ctx.Device.Token == "" {
		ctx.Device.Token = defaultCtx.Device.Token
	}
	// Direct.
	if !ctx.Direct {
		ctx.Direct = defaultCtx.Direct
	}
	// IP.
	if ctx.IP == "" {
		ctx.IP = defaultCtx.IP
	}
	// Library.
	if ctx.Library.Name == "" {
		ctx.Library.Name = defaultCtx.Library.Name
	}
	if ctx.Library.Version == "" {
		ctx.Library.Version = defaultCtx.Library.Version
	}
	// Locale.
	if ctx.Locale == "" {
		ctx.Locale = defaultCtx.Locale
	}
	// Location.
	if ctx.Location.City == "" {
		ctx.Location.City = defaultCtx.Location.City
	}
	if ctx.Location.Country == "" {
		ctx.Location.Country = defaultCtx.Location.Country
	}
	if ctx.Location.Latitude == 0 {
		ctx.Location.Latitude = defaultCtx.Location.Latitude
	}
	if ctx.Location.Longitude == 0 {
		ctx.Location.Longitude = defaultCtx.Location.Longitude
	}
	if ctx.Location.Speed == 0 {
		ctx.Location.Speed = defaultCtx.Location.Speed
	}
	// Network.
	if !ctx.Network.Bluetooth {
		ctx.Network.Bluetooth = defaultCtx.Network.Bluetooth
	}
	if ctx.Network.Carrier == "" {
		ctx.Network.Carrier = defaultCtx.Network.Carrier
	}
	if !ctx.Network.Cellular {
		ctx.Network.Cellular = defaultCtx.Network.Cellular
	}
	if !ctx.Network.WiFi {
		ctx.Network.WiFi = defaultCtx.Network.WiFi
	}
	// OS.
	if ctx.OS.Name == "" {
		ctx.OS.Name = defaultCtx.OS.Name
	}
	if ctx.OS.Version == "" {
		ctx.OS.Version = defaultCtx.OS.Version
	}
	// Page.
	if ctx.Page.Path == "" {
		ctx.Page.Path = defaultCtx.Page.Path
	}
	if ctx.Page.Referrer == "" {
		ctx.Page.Referrer = defaultCtx.Page.Referrer
	}
	if ctx.Page.Search == "" {
		ctx.Page.Search = defaultCtx.Page.Search
	}
	if ctx.Page.Title == "" {
		ctx.Page.Title = defaultCtx.Page.Title
	}
	if ctx.Page.URL == "" {
		ctx.Page.URL = defaultCtx.Page.URL
	}
	// Referrer.
	if ctx.Referrer.Id == "" {
		ctx.Referrer.Id = defaultCtx.Referrer.Id
	}
	if ctx.Referrer.Type == "" {
		ctx.Referrer.Type = defaultCtx.Referrer.Type
	}
	// Screen.
	if ctx.Screen.Width == 0 {
		ctx.Screen.Width = defaultCtx.Screen.Width
	}
	if ctx.Screen.Height == 0 {
		ctx.Screen.Height = defaultCtx.Screen.Height
	}
	if ctx.Screen.Density == 0 {
		ctx.Screen.Density = defaultCtx.Screen.Density
	}
	// SessionId.
	if ctx.SessionId == 0 {
		ctx.SessionId = defaultCtx.SessionId
	}
	// SessionStart.
	if !ctx.SessionStart {
		ctx.SessionStart = defaultCtx.SessionStart
	}
	// GroupId.
	if ctx.GroupId == "" {
		ctx.GroupId = defaultCtx.GroupId
	}
	// Timezone.
	if ctx.Timezone == "" {
		ctx.Timezone = defaultCtx.Timezone
	}
	// Traits.
	if ctx.Traits == nil {
		ctx.Traits = defaultCtx.Traits
	}
	// UserAgent.
	if ctx.UserAgent == "" {
		ctx.UserAgent = defaultCtx.UserAgent
	}
}

// enrichEvent enriches the given event.
func (c *Collector) enrichEvent(event *collectedEvent) {

	// AnonymousId.
	if event.AnonymousId == "" {
		event.AnonymousId = uuid.NewString()
	}

	// Browser and OS.
	if event.Context.UserAgent == "" {
		event.Context.Browser.Name = "None"
		event.Context.OS.Name = "None"
	} else {
		ua := useragent.New(event.Context.UserAgent)
		browserName, browserVersion := ua.Browser()
		switch browserName {
		default:
			event.Context.Browser.Name = "Other"
			if len(browserName) <= 25 {
				event.Context.Browser.Other = browserName
			}
		case "Chrome":
			event.Context.Browser.Name = "Chrome"
		case "Safari":
			event.Context.Browser.Name = "Safari"
		case "Edge":
			event.Context.Browser.Name = "Edge"
		case "Firefox":
			event.Context.Browser.Name = "Firefox"
		case "Samsung Internet":
			event.Context.Browser.Name = "Samsung Internet"
		case "Opera":
			event.Context.Browser.Name = "Opera"
		}
		if event.Context.Browser.Name != "Other" || event.Context.Browser.Other != "" {
			if strings.Contains(browserVersion, ".") {
				parts := strings.SplitN(browserVersion, ".", 3)
				if len(parts) == 3 {
					browserVersion = parts[0] + "." + parts[1]
				}
				if utf8.RuneCountInString(browserVersion) > 10 {
					browserVersion = parts[0]
				}
			}
			if utf8.RuneCountInString(browserVersion) <= 10 {
				event.Context.Browser.Version = browserVersion
			}
		}
		osInfo := ua.OSInfo()
		switch osInfo.Name {
		case "Mac OS X":
			event.Context.OS.Name = "macOS"
		case "Android", "Windows", "iOS", "Linux", "ChromeOS":
			event.Context.OS.Name = osInfo.Name
		default:
			event.Context.OS.Name = "Other"
		}
		if utf8.RuneCountInString(osInfo.Version) <= 10 {
			event.Context.OS.Version = osInfo.Version
		}
	}

	// IP.
	var requestIP net.IP
	if event.Context.IP == "" {
		ip, _, _ := net.SplitHostPort(event.header.RemoteAddr)
		requestIP = net.ParseIP(ip).To16()
	} else {
		requestIP = net.ParseIP(event.Context.IP).To16()
	}
	event.Context.IP = requestIP.String()

	// Locale.
	if event.Context.Locale != "" {
		event.Context.Locale = culture.Locale(event.Context.Locale).Name()
	}

	// Location.
	if loc := event.Context.Location; loc.Country == "" && loc.City == "" {
		if c.maxmindDB != nil {
			var record struct {
				City struct {
					Names struct {
						EN string `maxminddb:"en"`
					} `maxminddb:"names"`
				} `maxminddb:"city"`
				Country struct {
					IsoCode string `maxminddb:"iso_code"`
				} `maxminddb:"country"`
				Location struct {
					Latitude  float64 `maxminddb:"latitude"`
					Longitude float64 `maxminddb:"longitude"`
				} `maxminddb:"location"`
			}
			err := c.maxmindDB.Lookup(requestIP, &record)
			if err == nil {
				country := culture.Country(record.Country.IsoCode)
				if country != nil {
					event.Context.Location.Country = country.Code()
				}
				event.Context.Location.City = record.City.Names.EN
				event.Context.Location.Latitude = record.Location.Latitude
				event.Context.Location.Longitude = record.Location.Longitude
			}
		}
	} else if loc.Country != "" {
		c := culture.Country(loc.Country)
		event.Context.Location.Country = c.Code()
	}

	// Page.
	if event.Context.Page.URL != "" {
		u, _ := url.Parse(event.Context.Page.URL)
		event.Context.Page.URL = u.String()
		event.Context.Page.Path = u.Path
		event.Context.Page.Search = u.RawQuery
		if event.Context.Page.Referrer != "" {
			u, _ := url.Parse(event.Context.Page.Referrer)
			event.Context.Page.Referrer = u.String()
		}
	}

	// Screen.
	if w, h := event.Context.Screen.Width, event.Context.Screen.Height; w <= 0 || w > math.MaxInt16 || h <= 0 || h > math.MaxInt16 {
		event.Context.Screen.Width = 0
		event.Context.Screen.Height = 0
	}
	if d := event.Context.Screen.Density; d < 0 || d >= 9.995 {
		event.Context.Screen.Density = 0
	}

	// UserAgent.
	event.Context.UserAgent = event.header.Headers.Get("User-Agent")

	// MessageId.
	if event.MessageId == "" {
		event.MessageId = uuid.NewString()
	}

	// ReceivedAt.
	event.receivedAt = event.header.ReceivedAt

	// SentAt.
	var err error
	event.sentAt, err = iso8601.ParseString(event.SentAt)
	if err != nil {
		event.sentAt = event.header.ReceivedAt
	} else {
		event.sentAt = event.sentAt.UTC()
		// The iso8601.ParseString function returns years >= 0.
		if y := event.sentAt.Year(); y < 1 || y > 9999 {
			event.sentAt = event.header.ReceivedAt
		}
	}

	// Timestamp.
	event.timestamp, err = iso8601.ParseString(event.Timestamp)
	if err != nil {
		event.timestamp = event.header.ReceivedAt
	} else {
		skew := event.header.ReceivedAt.Sub(event.sentAt)
		event.timestamp = event.timestamp.UTC().Add(skew)
		// The iso8601.ParseString function returns years >= 0.
		if y := event.timestamp.Year(); y < 1 || y > 9999 {
			event.timestamp = event.header.ReceivedAt
		}
	}

	// Traits.
	if t := *event.Type; (t == "identify" || t == "group") && event.Traits == nil {
		event.Traits = map[string]any{}
	}

	// Context.Traits.
	if t := *event.Type; t != "identify" && t != "group" && event.Context.Traits == nil {
		event.Context.Traits = map[string]any{}
	}

	// Properties.
	if t := *event.Type; (t == "page" || t == "screen" || t == "track") && event.Properties == nil {
		event.Properties = map[string]any{}
	}

}

// removeDuplicatedEvents removes duplicated events returning the modified
// slice.
func (c *Collector) removeDuplicatedEvents(events []*collectedEvent) []*collectedEvent {
	for i := 0; i < len(events); i++ {
		if _, ok := c.messageIds.Load(events[i].id); ok {
			events = slices.Delete(events, i, i+1)
			i--
		}
	}
	return events
}

// setEventAsReceived sets the provided event as received.
func (c *Collector) setEventAsReceived(event *collectedEvent) {
	c.messageIds.Store(event.id, nil)
}

// setEventsAsReceived sets the provided events as received.
func (c *Collector) setEventsAsReceived(events []*collectedEvent) {
	for _, event := range events {
		c.messageIds.Store(event.id, nil)
	}
}

// storeEvents store the events in the data warehouse.
//
// If the data warehouse is in inspection mode, it returns the
// datastore.ErrInspectionMode error. If it is in maintenance mode, it returns
// the datastore.ErrMaintenanceMode error.
func (c *Collector) storeEvents(workspace int, events []*events.Event) error {

	store := c.datastore.Store(workspace)
	if store == nil {
		return nil
	}

	var traits bytes.Buffer
	traitsEnc := json.NewEncoder(&traits)
	traitsEnc.SetEscapeHTML(false)
	var properties bytes.Buffer
	propertiesEnc := json.NewEncoder(&properties)
	propertiesEnc.SetEscapeHTML(false)

	rows := make([][]any, len(events))

	for i, e := range events {

		var err error

		// Set traits.
		traits.Reset()
		if *e.Type == "identify" || *e.Type == "group" {
			err = traitsEnc.Encode(e.Traits)
		} else {
			err = traitsEnc.Encode(e.Context.Traits)
		}
		if err != nil {
			slog.Error("cannot marshal event", "err", err)
			continue
		}
		traits.Truncate(traits.Len() - 1) // remove the new line added by json.Encode

		// Set properties.
		properties.Reset()
		if e.Properties == nil {
			properties.WriteString("{}")
		} else {
			err = propertiesEnc.Encode(e.Properties)
			if err != nil {
				slog.Error("cannot marshal event", "err", err)
				continue
			}
			properties.Truncate(properties.Len() - 1) // remove the new line added by json.Encode
		}

		// Set groupId.
		groupId := e.GroupId
		if *e.Type != "group" {
			groupId = e.Context.GroupId
		}

		rows[i] = []any{
			e.AnonymousId,                                     // anonymous_id
			e.Category,                                        // category
			e.Context.App.Name,                                // context_app_name
			e.Context.App.Version,                             // context_app_version
			e.Context.App.Build,                               // context_app_build
			e.Context.App.Namespace,                           // context_app_namespace
			e.Context.Browser.Name,                            // context_browser_name
			e.Context.Browser.Other,                           // context_browser_other
			e.Context.Browser.Version,                         // context_browser_version
			e.Context.Campaign.Name,                           // context_campaign_name
			e.Context.Campaign.Source,                         // context_campaign_source
			e.Context.Campaign.Medium,                         // context_campaign_medium
			e.Context.Campaign.Term,                           // context_campaign_term
			e.Context.Campaign.Content,                        // context_campaign_content
			e.Context.Device.Id,                               // context_device_id
			e.Context.Device.AdvertisingId,                    // context_device_advertising_id
			e.Context.Device.AdTrackingEnabled,                // context_device_ad_tracking_enabled
			e.Context.Device.Manufacturer,                     // context_device_manufacturer
			e.Context.Device.Model,                            // context_device_model
			e.Context.Device.Name,                             // context_device_name
			e.Context.Device.Type,                             // context_device_type
			e.Context.Device.Token,                            // context_device_token
			e.Context.IP,                                      // context_ip
			e.Context.Library.Name,                            // context_library_name
			e.Context.Library.Version,                         // context_library_version
			e.Context.Locale,                                  // context_locale
			e.Context.Location.City,                           // context_location_city
			e.Context.Location.Country,                        // context_location_country
			e.Context.Location.Latitude,                       // context_location_latitude
			e.Context.Location.Longitude,                      // context_location_longitude
			e.Context.Location.Speed,                          // context_location_speed
			e.Context.Network.Bluetooth,                       // context_network_bluetooth
			e.Context.Network.Carrier,                         // context_network_carrier
			e.Context.Network.Cellular,                        // context_network_cellular
			e.Context.Network.WiFi,                            // context_network_wifi
			e.Context.OS.Name,                                 // context_os_name
			e.Context.OS.Version,                              // context_os_version
			e.Context.Page.Path,                               // context_page_path
			e.Context.Page.Referrer,                           // context_page_referrer
			e.Context.Page.Search,                             // context_page_search
			e.Context.Page.Title,                              // context_page_title
			e.Context.Page.URL,                                // context_page_url
			e.Context.Referrer.Id,                             // context_referrer_id
			e.Context.Referrer.Type,                           // context_referrer_type
			int16(e.Context.Screen.Width),                     // context_screen_width
			int16(e.Context.Screen.Height),                    // context_screen_height
			e.Context.Screen.Density,                          // context_screen_density
			e.Context.SessionId,                               // context_session_id
			e.Context.SessionStart,                            // context_session_start
			e.Context.Timezone,                                // context_timezone
			e.Context.UserAgent,                               // context_user_agent
			e.Event,                                           // event
			groupId,                                           // group_id
			e.MessageId,                                       // message_id
			e.Name,                                            // name
			json.RawMessage(slices.Clone(properties.Bytes())), // properties
			e.ReceivedAt,                                      // received_at
			e.SentAt,                                          // sent_at
			e.Header.Connection,                               // source
			e.Timestamp,                                       // timestamp
			json.RawMessage(slices.Clone(traits.Bytes())),     // traits
			*e.Type,  // type
			e.UserId, // user_id
		}

	}

	return store.AddEvents(rows)
}

// collectHeader returns selected headers of r.
func collectHeader(r *http.Request) http.Header {
	h := make(http.Header)
	for k, v := range r.Header {
		switch k {
		case
			"Content-Encoding",
			"Content-Length",
			"Content-Type",
			"User-Agent":
			h[k] = v
		}
	}
	h.Add("Host", r.Host)
	return h
}

// Send a successful response to the client.
func writeOK(w http.ResponseWriter, origin string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", "21")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	_, _ = io.WriteString(w, "{\n  \"success\": true\n}")
}

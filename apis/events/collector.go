//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package events

import (
	"bytes"
	"context"
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
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/culture"
	"chichi/apis/datastore"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/state"
	"chichi/apis/transformers"

	"github.com/google/uuid"
	"github.com/mssola/useragent"
	"github.com/oschwald/geoip2-golang"
	"github.com/relvacode/iso8601"
	"github.com/segmentio/ksuid"
	"golang.org/x/exp/maps"
	"golang.org/x/text/unicode/norm"
)

// maxRequestSize is the maximum size inBatchRequests bytes of an event request body.
const maxRequestSize = 500 * 1024

// Errors handled by the HTTP server of the collector.
var (
	errUnauthorized = errors.New("unauthorized")
	errBadRequest   = errors.New("bad request")
	errNotFound     = errors.New("not found")
)

type batchEvents struct {
	Batch    []*collectedEvent `json:"batch"`
	Context  *eventContext     `json:"context,omitempty"`
	WriteKey string            `json:"writeKey,omitempty"`
}

// A collector collects events, store them in the event log and sends them to
// the processor.
type collector struct {
	state       *eventsState
	datastore   *datastore.Datastore
	eventLog    *eventsLog
	events      chan *collectedEvent
	observer    *Observer
	transformer transformers.Function
	geoLiteDB   *geoip2.Reader
}

// newCollector returns a new event collector. It receives HTTP requests from
// mobile, server and website sources and sends them to the eventsLog.
func newCollector(st *eventsState, ds *datastore.Datastore, eventLog *eventsLog, transformer transformers.Function, observer *Observer) (*collector, error) {
	var collector = collector{
		state:       st,
		datastore:   ds,
		eventLog:    eventLog,
		events:      make(chan *collectedEvent, 1000),
		observer:    observer,
		transformer: transformer,
	}
	var err error
	collector.geoLiteDB, err = geoip2.Open(geoLite2Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot open GeoLite at path %q: %s", geoLite2Path, err)
	}
	return &collector, nil
}

// Events returns the events channel.
func (c *collector) Events() <-chan *collectedEvent {
	return c.events
}

// ServeHTTP serves event messages from HTTP.
// A message is a JSON stream of JSON objects where the first object is the
// message header.
func (c *collector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := c.serveHTTP(w, r)
	if err != nil {
		switch err {
		case errBadRequest:
			http.Error(w, "Bad batchRequest", http.StatusBadRequest)
		case errNotFound:
			http.Error(w, "Invalid path or identifier", http.StatusNotFound)
		case errUnauthorized:
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		default:
			slog.Error("error occurred collecting an event", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// importUserTraits imports the user traits from the given events batch
// collected on the source connection.
func (c *collector) importUserTraits(ctx context.Context, source *state.Connection, eventsBatch []*collectedEvent) error {
	anonIdents := source.Workspace().AnonymousIdentifiers.Mapping
	for _, action := range source.Actions() {
		if !action.Enabled {
			continue
		}
		if action.Target != state.Users {
			continue
		}
		ws := action.Connection().Workspace()
		store := c.datastore.Store(ws.ID)

		// Instantiate an IdentitiesWriter for writing the users identities.
		ack := func(err error, ids []string) {
			if err != nil {
				slog.Warn("cannot import users traits", "action", action.ID, "ids", ids, "err", err)
				return
			}
			slog.Warn("users traits imported successfully", "action", action.ID, "ids", ids)
		}
		iw := store.IdentitiesWriter(ctx, action.OutSchema, action.Connection().ID, true, ack)
		defer iw.Close(ctx)

		// Import the user traits for this event, if provided.
		for _, event := range eventsBatch {
			if len(event.Traits) == 0 && len(event.Context.Traits) == 0 {
				continue
			}
			// TODO(Gianluca): shall we normalize the user properties before
			// transformation?
			transformation := state.Transformation{
				Mapping:  action.Transformation.Mapping,
				Function: action.Transformation.Function,
			}
			if len(anonIdents) > 0 {
				transformation.Mapping = map[string]string{}
				maps.Copy(transformation.Mapping, action.Transformation.Mapping)
				maps.Copy(transformation.Mapping, anonIdents)
			}
			transformer, err := transformers.New(action.InSchema, action.OutSchema, transformation, action.ID, c.transformer, nil)
			if err != nil {
				return err
			}
			// Transform the event.
			properties, err := transformer.Transform(ctx, event.MapEvent())
			if err != nil {
				return err
			}
			// Write the user identity on the data warehouse.
			ok := iw.Write(ctx, warehouses.Identity{
				ID:          event.UserId,
				Properties:  properties,
				AnonymousID: event.AnonymousId,
				Timestamp:   event.timestamp,
			})
			if !ok {
				return iw.Close(ctx)
			}
		}
		err := iw.Close(ctx)
		if err != nil {
			return err
		}
		// Resolve and sync the users.
		err = store.ResolveSyncUsers(ctx)
		if err != nil {
			return fmt.Errorf("cannot resolve and sync users: %s", err)
		}
	}
	return nil
}

// serveHTTP is called by the ServeHTTP method to serve an event request.
func (c *collector) serveHTTP(w http.ResponseWriter, r *http.Request) error {

	ctx := r.Context()
	date := time.Now().UTC()

	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	switch method {
	case "batch", "track", "page", "screen", "identify", "group", "alias":
	default:
		return errNotFound
	}

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" || len(params) > 1 {
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

	header := &EventHeader{
		ReceivedAt: date,
		RemoteAddr: r.RemoteAddr,
		Method:     r.Method,
		Proto:      r.Proto,
		URL:        r.URL.String(),
		Headers:    collectHeader(r),
	}

	// Read the body and check that is not be longer than maxRequestSize bytes and,
	// it is a streaming of JSON objects, otherwise return the errBadRequest error.
	lr := &io.LimitedReader{R: r.Body, N: maxRequestSize + 1}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r.Body)
	if err != nil {
		return err
	}
	if lr.N == 0 {
		return errBadRequest
	}
	b := buf.Bytes()

	// Read the events.
	nr := norm.NFC.Reader(bytes.NewReader(b))
	dec := json.NewDecoder(nr)
	dec.UseNumber()

	var events batchEvents
	if method == "batch" {
		err = dec.Decode(&events)
	} else {
		events.Batch = make([]*collectedEvent, 1)
		err = dec.Decode(&events.Batch[0])
	}
	if err != nil {
		return errBadRequest
	}
	if len(events.Batch) == 0 {
		return errBadRequest
	}

	// Validate the write key.
	var source *state.Connection
	{
		writeKey := events.WriteKey
		if method != "batch" {
			writeKey = events.Batch[0].WriteKey
		}
		if writeKey == "" {
			if key, _, ok := r.BasicAuth(); ok {
				writeKey = key
			}
		}
		if writeKey != "" {
			source, _ = c.state.ConnectionByKey(writeKey)
		}
		if source == nil {
			// Send a successful response to the client.
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "21")
			_, _ = io.WriteString(w, "{\n  \"success\": true\n}")
			return nil
		}
	}
	header.source = source.ID

	for i := 0; i < len(events.Batch); i++ {
		event := events.Batch[i]
		if event.Type == nil && method != "batch" {
			event.Type = &method
		}
		event.header = header
		mergeContexts(&event.Context, events.Context)
		err = validateEvent(method, event)
		c.observer.addEvent(header.source, event, err)
		if err != nil {
			// Remove the invalid event.
			events.Batch = slices.Delete(events.Batch, i, i+1)
			i--
			continue
		}
		event.id = ksuid.New()
	}
	if len(events.Batch) == 0 {
		return nil
	}

	if !c.state.HasEnabledActions(source) {
		// Send a successful response to the client.
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "21")
		_, _ = io.WriteString(w, "{\n  \"success\": true\n}")
		return nil
	}

	// Append the events sources to the events log.
	ack := c.eventLog.Append(events.Batch)

	// Enrich the events.
	for _, event := range events.Batch {
		c.enrichEvent(event)
	}

	// Wait for the events to be logged.
	if err = <-ack; err != nil {
		return err
	}

	// Send a successful response to the client.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", "21")
	_, _ = io.WriteString(w, "{\n  \"success\": true\n}")

	// Store the events into the data warehouse.
	c.storeEvents(source.Workspace().ID, events.Batch)

	// In case there are user traits in some events, import the user from the
	// events batch.
	importUserTraits := false
	for _, event := range events.Batch {
		if len(event.Traits) > 0 || len(event.Context.Traits) > 0 {
			importUserTraits = true
			break
		}
	}
	if importUserTraits {
		err := c.importUserTraits(ctx, source, events.Batch)
		if err != nil {
			return err
		}
	}

	// Send the events to the next stage.
	for _, event := range events.Batch {
		c.events <- event
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

	// Event.
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
func mergeContexts(ctx, defaultCtx *eventContext) {
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
func (c *collector) enrichEvent(event *collectedEvent) {

	// Source.
	event.source = event.header.source

	// AnonymousId.
	if event.AnonymousId == "" {
		event.AnonymousId = uuid.NewString()
	}

	// Browser and OS.
	if event.Context.UserAgent == "" {
		event.Context.browser.Name = "None"
		event.Context.OS.Name = "None"
	} else {
		ua := useragent.New(event.Context.UserAgent)
		browserName, browserVersion := ua.Browser()
		switch browserName {
		default:
			event.Context.browser.Name = "Other"
			if len(browserName) <= 25 {
				event.Context.browser.Other = browserName
			}
		case "Chrome":
			event.Context.browser.Name = "Chrome"
		case "Safari":
			event.Context.browser.Name = "Safari"
		case "Edge":
			event.Context.browser.Name = "Edge"
		case "Firefox":
			event.Context.browser.Name = "Firefox"
		case "Samsung Internet":
			event.Context.browser.Name = "Samsung Internet"
		case "Opera":
			event.Context.browser.Name = "Opera"
		}
		if event.Context.browser.Name != "Other" || event.Context.browser.Other != "" {
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
				event.Context.browser.Version = browserVersion
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
		if c.geoLiteDB != nil {
			city, err := c.geoLiteDB.City(requestIP)
			if err == nil {
				country := culture.Country(city.Country.IsoCode)
				if country != nil {
					event.Context.Location.Country = country.Code()
				}
				event.Context.Location.City = city.City.Names["en"]
				event.Context.Location.Latitude = city.Location.Latitude
				event.Context.Location.Longitude = city.Location.Longitude
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
	}
	event.sentAt = event.sentAt.UTC()

	// Timestamp.
	event.timestamp, err = iso8601.ParseString(event.Timestamp)
	if err != nil {
		event.timestamp = event.sentAt
	}
	skew := event.header.ReceivedAt.Sub(event.sentAt)
	event.timestamp = event.timestamp.UTC().Add(skew)

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

	return
}

// storeEvents store the events in the data warehouse.
func (c *collector) storeEvents(workspace int, events []*collectedEvent) {

	store := c.datastore.Store(workspace)
	if store == nil {
		return
	}

	var traits bytes.Buffer
	traitsEnc := json.NewEncoder(&traits)
	traitsEnc.SetEscapeHTML(false)
	var properties bytes.Buffer
	propertiesEnc := json.NewEncoder(&properties)
	propertiesEnc.SetEscapeHTML(false)

	rows := make([]map[string]any, len(events))

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

		rows[i] = map[string]any{
			"gid":         0, // TODO: set the correct GID. See https://github.com/open2b/chichi/issues/483.
			"anonymousId": e.AnonymousId,
			"category":    e.Category,
			"context": map[string]any{
				"app": map[string]any{
					"name":      e.Context.App.Name,
					"version":   e.Context.App.Version,
					"build":     e.Context.App.Build,
					"namespace": e.Context.App.Namespace,
				},
				"browser": map[string]any{
					"name":    e.Context.browser.Name,
					"other":   e.Context.browser.Other,
					"version": e.Context.browser.Version,
				},
				"campaign": map[string]any{
					"name":    e.Context.Campaign.Name,
					"source":  e.Context.Campaign.Source,
					"medium":  e.Context.Campaign.Medium,
					"term":    e.Context.Campaign.Term,
					"content": e.Context.Campaign.Content,
				},
				"device": map[string]any{
					"id":                e.Context.Device.Id,
					"advertisingId":     e.Context.Device.AdvertisingId,
					"adTrackingEnabled": e.Context.Device.AdTrackingEnabled,
					"manufacturer":      e.Context.Device.Manufacturer,
					"model":             e.Context.Device.Model,
					"name":              e.Context.Device.Name,
					"type":              e.Context.Device.Type,
					"token":             e.Context.Device.Token,
				},
				"ip": e.Context.IP,
				"library": map[string]any{
					"name":    e.Context.Library.Name,
					"version": e.Context.Library.Version,
				},
				"locale": e.Context.Locale,
				"location": map[string]any{
					"city":      e.Context.Location.City,
					"country":   e.Context.Location.Country,
					"latitude":  e.Context.Location.Latitude,
					"longitude": e.Context.Location.Longitude,
					"speed":     e.Context.Location.Speed,
				},
				"network": map[string]any{
					"bluetooth": e.Context.Network.Bluetooth,
					"carrier":   e.Context.Network.Carrier,
					"cellular":  e.Context.Network.Cellular,
					"wifi":      e.Context.Network.WiFi,
				},
				"os": map[string]any{
					"name":    e.Context.OS.Name,
					"version": e.Context.OS.Version,
				},
				"page": map[string]any{
					"path":     e.Context.Page.Path,
					"referrer": e.Context.Page.Referrer,
					"search":   e.Context.Page.Search,
					"title":    e.Context.Page.Title,
					"url":      e.Context.Page.URL,
				},
				"referrer": map[string]any{
					"id":   e.Context.Referrer.Id,
					"type": e.Context.Referrer.Type,
				},
				"screen": map[string]any{
					"width":   int16(e.Context.Screen.Width),
					"height":  int16(e.Context.Screen.Height),
					"density": e.Context.Screen.Density,
				},
				"session": map[string]any{
					"id":    e.Context.SessionId,
					"start": e.Context.SessionStart,
				},
				"timezone":  e.Context.Timezone,
				"userAgent": e.Context.UserAgent,
			},
			"event":      e.Event,
			"groupId":    groupId,
			"messageId":  e.MessageId,
			"name":       e.Name,
			"properties": json.RawMessage(properties.Bytes()),
			"receivedAt": e.receivedAt,
			"sentAt":     e.sentAt,
			"source":     e.source,
			"timestamp":  e.timestamp,
			"traits":     json.RawMessage(traits.Bytes()),
			"type":       *e.Type,
			"userId":     e.UserId,
		}

	}

	store.AddEvents(rows)

	return
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

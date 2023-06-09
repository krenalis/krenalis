//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package events

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/state"

	"github.com/google/uuid"
	"github.com/mssola/useragent"
	"github.com/open2b/nuts/culture"
	"github.com/oschwald/geoip2-golang"
	"github.com/relvacode/iso8601"
	"github.com/segmentio/ksuid"
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

// collectedHeader represents the header of a batch request.
type collectedHeader struct {
	ReceivedAt time.Time   `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
	source     int
	sourceType state.ConnectorType
	server     int
}

type batchEvents struct {
	Batch   []*collectedEvent `json:"batch"`
	Context *eventContext     `json:"context,omitempty"`
	SentAt  string            `json:"sentAt,omitempty"`
}

type eventContext struct {
	App struct {
		Name      string `json:"name,omitempty"`
		Version   string `json:"version,omitempty"`
		Build     string `json:"build,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"app,omitempty"`
	Campaign struct {
		Name    string `json:"name,omitempty"`
		Source  string `json:"source,omitempty"`
		Medium  string `json:"medium,omitempty"`
		Term    string `json:"term,omitempty"`
		Content string `json:"content,omitempty"`
	} `json:"campaign,omitempty"`
	Device struct {
		ID            string `json:"id,omitempty"`
		Name          string `json:"name,omitempty"`
		Manufacturer  string `json:"manufacturer,omitempty"`
		Model         string `json:"model,omitempty"`
		Type          string `json:"type,omitempty"`
		Version       string `json:"version,omitempty"`
		AdvertisingID string `json:"advertisingId,omitempty"`
	} `json:"device,omitempty"`
	Direct  bool   `json:"direct,omitempty"`
	IP      string `json:"ip,omitempty"`
	Library struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"library,omitempty"`
	Locale   string `json:"locale,omitempty"`
	Location struct {
		City      string  `json:"city,omitempty"`
		Country   string  `json:"country,omitempty"`
		Region    string  `json:"region,omitempty"`
		Latitude  float64 `json:"latitude,omitempty"`
		Longitude float64 `json:"longitude,omitempty"`
		Speed     float64 `json:"speed,omitempty"`
	} `json:"location,omitempty"`
	Network struct {
		Cellular  bool   `json:"cellular,omitempty"`
		WiFi      bool   `json:"wifi,omitempty"`
		Bluetooth bool   `json:"bluetooth,omitempty"`
		Carrier   string `json:"carrier,omitempty"`
	} `json:"network,omitempty"`
	OS struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"os,omitempty"`
	Page struct {
		URL      string `json:"url,omitempty"`
		Path     string `json:"path,omitempty"`
		Search   string `json:"search,omitempty"`
		Hash     string `json:"hash,omitempty"`
		Title    string `json:"title,omitempty"`
		Referrer string `json:"referrer,omitempty"`
	} `json:"page,omitempty"`
	Referrer struct {
		Type string `json:"type,omitempty"`
		Name string `json:"name,omitempty"`
		URL  string `json:"url,omitempty"`
		Link string `json:"link,omitempty"`
	} `json:"referrer,omitempty"`
	Screen struct {
		Density float64 `json:"density,omitempty"`
		Width   int     `json:"width,omitempty"`
		Height  int     `json:"height,omitempty"`
	} `json:"screen,omitempty"`
	Timezone  string          `json:"timezone,omitempty"`
	Traits    json.RawMessage `json:"traits,omitempty"`
	UserAgent string          `json:"userAgent,omitempty"`
}

type collectedEvent struct {
	header *collectedHeader

	AnonymousID  string          `json:"anonymousId,omitempty"`
	Context      eventContext    `json:"context,omitempty"`
	Event        string          `json:"event,omitempty"`
	GroupID      string          `json:"groupId,omitempty"`
	Integrations json.RawMessage `json:"integrations,omitempty"`
	MessageID    string          `json:"messageId,omitempty"`
	Name         string          `json:"name,omitempty"`
	PreviousID   string          `json:"previousId,omitempty"`
	Properties   json.RawMessage `json:"properties,omitempty"`
	Timestamp    string          `json:"timestamp,omitempty"`
	Traits       json.RawMessage `json:"traits,omitempty"`
	Type         *string         `json:"type"`
	UserID       string          `json:"userId,omitempty"`

	browser struct {
		name    string
		other   string
		version string
	}
	date   string
	domain string
	id     ksuid.KSUID
	ip     string
	page   struct {
		url      string
		path     string
		search   string
		hash     string
		title    string
		referrer string
	}
	properties string
	receivedAt time.Time
	screen     struct {
		density int16
		width   int16
		height  int16
	}
	sentAt    time.Time
	source    int32
	timestamp time.Time
	userAgent string
}

// A collector collects events, store them in the event log and sends them to
// the processor.
type collector struct {
	state     *eventsState
	eventLog  *eventsLog
	events    chan *collectedEvent
	observer  *Observer
	warehouse *warehouses
	geoLiteDB *geoip2.Reader
}

// newCollector returns a new event collector. It receives HTTP requests from
// mobile, server and website sources and sends them to the eventsLog.
func newCollector(st *eventsState, eventLog *eventsLog, observer *Observer, warehouse *warehouses) (*collector, error) {
	var collector = collector{
		state:     st,
		eventLog:  eventLog,
		events:    make(chan *collectedEvent, 1000),
		observer:  observer,
		warehouse: warehouse,
	}
	var err error
	collector.geoLiteDB, err = geoip2.Open(geoLite2Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot open GeoLite at path %q: %s", geoLite2Path, err)
	}
	return &collector, nil
}

// Events returns the collected events channel.
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
			log.Printf("[error] %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// serveHTTP is called by the ServeHTTP method to serve an event request.
func (c *collector) serveHTTP(w http.ResponseWriter, r *http.Request) error {

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

	// Authenticate the request.
	auth, _, ok := r.BasicAuth()
	if !ok || auth == "" {
		return errUnauthorized
	}
	key, src, ok := strings.Cut(auth, "-")
	if !ok && len(key) <= 10 {
		src, key = key, ""
	}

	// Validate the source connection.
	var source *state.Connection
	if src != "" {
		sourceID, _ := strconv.Atoi(src)
		if sourceID < 1 || sourceID > math.MaxInt32 {
			return errBadRequest
		}
		source, ok = c.state.Source(sourceID)
		if !ok {
			return errNotFound
		}
		sourceType := source.Connector().Type
		if sourceType != state.MobileType && sourceType != state.WebsiteType {
			return errNotFound
		}
		if !c.state.HasEnabledActions(sourceID) {
			return errNotFound
		}
	}

	// Validate the server key.
	var serverID int
	var server *state.Connection
	if key != "" {
		server, ok = c.state.ServerByKey(key)
		if !ok {
			return errUnauthorized
		}
		if source != nil && server.Workspace().ID != source.Workspace().ID {
			return errUnauthorized
		}
		serverID = server.ID
		if !c.state.HasEnabledActions(serverID) {
			return errNotFound
		}
	}

	var sourceID int
	var sourceType state.ConnectorType
	if source != nil {
		sourceID = source.ID
		sourceType = source.Connector().Type
	}
	header := &collectedHeader{
		ReceivedAt: date,
		RemoteAddr: r.RemoteAddr,
		Method:     r.Method,
		Proto:      r.Proto,
		URL:        r.URL.String(),
		Headers:    r.Header,
		source:     sourceID,
		sourceType: sourceType,
		server:     serverID,
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

	for i := 0; i < len(events.Batch); i++ {
		event := events.Batch[i]
		if event.Type == nil && method != "batch" {
			event.Type = &method
		}
		event.header = header
		mergeContexts(&event.Context, events.Context)
		err = validateEvent(method, event)
		c.observer.AddEvent(sourceID, serverID, 0, event, err)
		if err != nil {
			// Remove the invalid event.
			copy(events.Batch[:i], events.Batch[i+1:])
			events.Batch = events.Batch[:len(events.Batch)-1]
			i--
			continue
		}
		event.id = ksuid.New()
	}
	if len(events.Batch) == 0 {
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

	// Sent a successful response to the client.
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", "21")
	_, _ = io.WriteString(w, "{\n  \"success\": true\n}")

	// Send the events to the data warehouse.
	c.warehouse.Add(events.Batch)

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

	// Event.
	if event.Event == "" && isTrack {
		return errors.New("missing event name")
	}
	if event.Event != "" && !isTrack {
		return errors.New("unexpected event name")
	}

	// GroupID.
	if event.GroupID == "" && isGroup {
		return errors.New("missing event group")
	}
	if event.GroupID != "" && !isGroup {
		return errors.New("unexpected event group")
	}

	// Name.
	if event.Name != "" && !isScreen && !isPage {
		return errors.New("unexpected screen or page name")
	}

	// PreviousID.
	if event.PreviousID == "" && isAlias {
		return errors.New("missing event previousId")
	}
	if event.PreviousID != "" && !isAlias {
		return errors.New("unexpected event previousId")
	}

	// Properties.
	if event.Properties != nil {
		if event.Properties[0] != '{' {
			return errors.New("properties is not a JSON object")
		}
		if !isPage && !isScreen && !isTrack {
			return errors.New("unexpected event properties")
		}
	}

	// Traits.
	if event.Traits != nil {
		if event.Traits[0] != '{' {
			return errors.New("traits is not a JSON object")
		}
		if !isIdentify && !isGroup {
			return errors.New("unexpected event traits")
		}
	}

	// AnonymousID and UserID.
	if event.AnonymousID == "" && event.UserID == "" {
		if isIdentify || isAlias {
			return errors.New("missing event userId")
		}
		return errors.New("missing event anonymousId")
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
	if ctx.Device.ID == "" {
		ctx.Device.ID = defaultCtx.Device.ID
	}
	if ctx.Device.Name == "" {
		ctx.Device.Name = defaultCtx.Device.Name
	}
	if ctx.Device.Manufacturer == "" {
		ctx.Device.Manufacturer = defaultCtx.Device.Manufacturer
	}
	if ctx.Device.Model == "" {
		ctx.Device.Model = defaultCtx.Device.Model
	}
	if ctx.Device.Type == "" {
		ctx.Device.Type = defaultCtx.Device.Type
	}
	if ctx.Device.Version == "" {
		ctx.Device.Version = defaultCtx.Device.Version
	}
	if ctx.Device.AdvertisingID == "" {
		ctx.Device.AdvertisingID = defaultCtx.Device.AdvertisingID
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
	if ctx.Location.Region == "" {
		ctx.Location.Region = defaultCtx.Location.Region
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
	if !ctx.Network.Cellular {
		ctx.Network.Cellular = defaultCtx.Network.Cellular
	}
	if !ctx.Network.WiFi {
		ctx.Network.WiFi = defaultCtx.Network.WiFi
	}
	if !ctx.Network.Bluetooth {
		ctx.Network.Bluetooth = defaultCtx.Network.Bluetooth
	}
	if ctx.Network.Carrier == "" {
		ctx.Network.Carrier = defaultCtx.Network.Carrier
	}
	// OS.
	if ctx.OS.Name == "" {
		ctx.OS.Name = defaultCtx.OS.Name
	}
	if ctx.OS.Version == "" {
		ctx.OS.Version = defaultCtx.OS.Version
	}
	// Page.
	if ctx.Page.URL == "" {
		ctx.Page.URL = defaultCtx.Page.URL
	}
	if ctx.Page.Path == "" {
		ctx.Page.Path = defaultCtx.Page.Path
	}
	if ctx.Page.Search == "" {
		ctx.Page.Search = defaultCtx.Page.Search
	}
	if ctx.Page.Hash == "" {
		ctx.Page.Hash = defaultCtx.Page.Hash
	}
	if ctx.Page.Title == "" {
		ctx.Page.Title = defaultCtx.Page.Title
	}
	if ctx.Page.Referrer == "" {
		ctx.Page.Referrer = defaultCtx.Page.Referrer
	}
	// Referrer.
	if ctx.Referrer.Type == "" {
		ctx.Referrer.Type = defaultCtx.Referrer.Type
	}
	if ctx.Referrer.Name == "" {
		ctx.Referrer.Name = defaultCtx.Referrer.Name
	}
	if ctx.Referrer.URL == "" {
		ctx.Referrer.URL = defaultCtx.Referrer.URL
	}
	if ctx.Referrer.Link == "" {
		ctx.Referrer.Link = defaultCtx.Referrer.Link
	}
	// Screen.
	if ctx.Screen.Density == 0 {
		ctx.Screen.Density = defaultCtx.Screen.Density
	}
	if ctx.Screen.Width == 0 {
		ctx.Screen.Width = defaultCtx.Screen.Width
	}
	if ctx.Screen.Height == 0 {
		ctx.Screen.Height = defaultCtx.Screen.Height
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

	now := time.Now().UTC()

	// MessageID.
	if event.MessageID == "" {
		event.MessageID = uuid.NewString()
	}

	// AnonymousID.
	if event.AnonymousID == "" {
		event.AnonymousID = uuid.NewString()
	}

	// Properties.
	if len(event.Properties) > 0 && !bytes.Equal(event.Properties, emptyProperties) {
		// Decode the properties.
		dec := json.NewDecoder(bytes.NewReader(event.Properties))
		dec.UseNumber()
		event.Properties = nil
		var properties map[string]any
		err := dec.Decode(&properties)
		if err == nil {
			// Encode the properties.
			var b strings.Builder
			enc := json.NewEncoder(&b)
			enc.SetIndent("", "")
			enc.SetEscapeHTML(false)
			err = enc.Encode(properties)
			if err == nil {
				s := b.String()
				event.properties = s[:len(s)-1] // remove the new line.
			}
		}
	}
	if len(event.properties) == 0 {
		event.properties = "{}"
	}

	// Locale.
	if event.Context.Locale != "" {
		event.Context.Locale = culture.Locale(event.Context.Locale).Name()
	}

	// IP.
	var requestIP net.IP
	if event.Context.IP == "" {
		ip, _, _ := net.SplitHostPort(event.header.RemoteAddr)
		requestIP = net.ParseIP(ip).To16()
	} else {
		requestIP = net.ParseIP(event.Context.IP).To16()
	}
	event.ip = requestIP.String()

	// page.
	if event.header.sourceType != state.MobileType {
		u, _ := url.Parse(event.Context.Page.URL)
		event.page.url = u.String()
		event.page.path = u.Path
		event.page.search = u.RawQuery
		event.page.hash = u.Fragment
		event.page.title = event.Context.Page.Title
		if event.Context.Page.Referrer != "" {
			u, _ := url.Parse(event.Context.Page.Referrer)
			event.page.referrer = u.String()
		}
	}

	// Screen.
	if d := event.Context.Screen.Density; 0 < d && d < 10 {
		event.screen.density = int16(math.Round(d * 100))
	}
	if w, h := event.Context.Screen.Width, event.Context.Screen.Height; (0 < w && w <= math.MaxInt16) && (0 < h && h <= math.MaxInt16) {
		event.screen.width = int16(w)
		event.screen.height = int16(h)
	}

	// Timestamp and date.
	if event.Timestamp == "" {
		event.timestamp = event.header.ReceivedAt
	} else {
		event.timestamp, _ = iso8601.ParseString(event.Timestamp)
		event.timestamp = event.timestamp.UTC()
		if event.header.server > 0 {
			if event.timestamp.After(now) {
				event.timestamp = event.header.ReceivedAt
			}
		} else {
			if t := event.timestamp; t.Add(-15*time.Minute).Before(now) || t.After(now) {
				event.timestamp = event.header.ReceivedAt
			}
		}
	}
	event.date = event.timestamp.Format(time.DateOnly)

	// ReceivedAt.
	event.receivedAt = event.header.ReceivedAt

	// Location.
	if loc := event.Context.Location; loc.Country == "" && loc.Region == "" && loc.City == "" {
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

	// UserAgent, DeviceType and Browser.
	if event.header.sourceType != state.MobileType {
		event.userAgent = event.header.Headers.Get("User-Agent")
		ua := useragent.New(event.userAgent)
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
		browserName, browserVersion := ua.Browser()
		switch browserName {
		default:
			event.browser.name = "Other"
			if len(browserName) <= 25 {
				event.browser.other = browserName
			}
		case "Chrome":
			event.browser.name = "Chrome"
		case "Safari":
			event.browser.name = "Safari"
		case "Edge":
			event.browser.name = "Edge"
		case "Firefox":
			event.browser.name = "Firefox"
		case "Samsung Internet":
			event.browser.name = "Samsung Internet"
		case "Opera":
			event.browser.name = "Opera"
		}
		if event.browser.name != "Other" || event.browser.other != "" {
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
				event.browser.version = browserVersion
			}
		}
	}

	return
}

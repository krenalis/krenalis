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
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"math/rand"
	"mime"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/events/collector"
	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector"

	"github.com/google/uuid"
	"github.com/mssola/useragent"
	"github.com/open2b/nuts/culture"
	"github.com/oschwald/geoip2-golang"
	"github.com/relvacode/iso8601"
)

const (
	flushQueueTimeout = 1 * time.Second // interval to flushEvents the queue
	geoLite2Path      = "GeoLite2-City.mmdb"
	maxEventSize      = 32 * 1024
	maxEventsQueueLen = 10000
)

// eventDateLayout is the layout used for dates in events.
const eventDateLayout = "2006-01-02T15:04:05.999Z"

// emptyProperties represents an empty event properties.
var emptyProperties = []byte("{}")

type Event struct {
	AnonymousId string
	City        string
	Country     string
	DeviceType  string
	Event       string
	IP          string
	Language    string
	OSName      string
	OSVersion   string
	Properties  json.RawMessage
	Referrer    string
	Screen      struct {
		Density float64
		Width   int
		Height  int
	}
	SentAt    string
	Target    string
	Text      string
	Timestamp string
	Title     string
	URL       string
	UserId    string
	UTM       struct {
		Source   string
		Medium   string
		Campaign string
		Term     string
		Content  string
	}
	browser struct {
		name    string
		other   string
		version string
	}
	location struct {
		city    string
		country struct {
			code string
			name string
		}
		latitude  float64
		longitude float64
		timezone  string
	}
	date   string
	domain string
	ip     string
	os     struct {
		name    string
		version string
	}
	page struct {
		path     string
		referrer string
		search   string
		title    string
		url      string
	}
	properties string
	receivedAt time.Time
	screen     struct {
		density uint16
		width   uint16
		height  uint16
	}
	sentAt    time.Time
	source    int32
	timestamp time.Time
	userAgent string

	// workspace, data and err are used during event processing.
	workspace int
	data      []byte
	err       error
}

// Processor processes events received from source streams and sent them to
// their data warehouses.
type Processor struct {
	sync.Mutex // for the streams field.
	db         *postgres.DB
	ctx        context.Context
	state      *state.State
	streams    map[int]*processorStream
	queues     map[int]*queue
	observer   *observer
	geoLiteDB  *geoip2.Reader
}

// processorStream represents a stream used by the processor.
type processorStream struct {
	id        int
	workspace int
	stream    connector.StreamConnection
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewProcessor returns a new processor.
func NewProcessor(ctx context.Context, db *postgres.DB, st *state.State,
	defaultStream connector.StreamConnection) (*Processor, error) {

	processor := Processor{
		db:       db,
		ctx:      ctx,
		state:    st,
		streams:  map[int]*processorStream{},
		queues:   map[int]*queue{},
		observer: newObserver(db),
	}

	for _, c := range st.Connections() {
		if processor.isSuitableStream(c) {
			processor.replaceStream(nil, c)
		}
	}

	var err error
	processor.geoLiteDB, err = geoip2.Open(geoLite2Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("cannot open GeoLite at path %q: %s", geoLite2Path, err)
	}

	// Process the events from the default stream.
	go processor.process(&processorStream{
		stream: defaultStream,
		ctx:    ctx,
	})

	st.AddListener(processor.onAddConnection)
	st.AddListener(processor.onDeleteConnection)
	st.AddListener(processor.onDeleteWorkspace)
	st.AddListener(processor.onSetConnectionSettings)
	st.AddListener(processor.onSetConnectionStatus)
	st.AddListener(processor.onSetWarehouseSettings)

	return &processor, nil
}

// isSuitableStream reports whether c is a stream from which the processor must
// read events.
func (processor *Processor) isSuitableStream(c *state.Connection) bool {
	if c == nil || !c.Enabled || c.Role != state.SourceRole {
		return false
	}
	if typ := c.Connector().Type; typ != state.StreamType {
		return false
	}
	if c.Workspace().Warehouse == nil {
		return false
	}
	return true
}

// onAddConnection is called when a connection is added.
func (processor *Processor) onAddConnection(n state.AddConnectionNotification) {
	c, _ := processor.state.Connection(n.ID)
	if processor.isSuitableStream(c) {
		go processor.replaceStream(nil, c)
	}
}

// onDeleteConnection is called when a connection is deleted.
func (processor *Processor) onDeleteConnection(n state.DeleteConnectionNotification) {
	if old, ok := processor.streams[n.ID]; ok {
		go processor.replaceStream(old, nil)
	}
}

// onDeleteWorkspace is called when a workspace is deleted.
func (processor *Processor) onDeleteWorkspace(n state.DeleteWorkspaceNotification) {
	for _, s := range processor.streams {
		if s.workspace == n.ID {
			go processor.replaceStream(s, nil)
		}
	}
}

// onSetConnectionSettings is called when the settings of a connection are
// changed.
func (processor *Processor) onSetConnectionSettings(n state.SetConnectionSettingsNotification) {
	if old, ok := processor.streams[n.Connection]; ok {
		new, _ := processor.state.Connection(n.Connection)
		go processor.replaceStream(old, new)
	}
}

// onSetConnectionStatus is called when the status of a connection is changed.
func (processor *Processor) onSetConnectionStatus(n state.SetConnectionStatusNotification) {
	c, _ := processor.state.Connection(n.Connection)
	if processor.isSuitableStream(c) {
		if _, ok := processor.streams[c.ID]; !ok {
			go processor.replaceStream(nil, c)
		}
	} else {
		if s, ok := processor.streams[c.ID]; ok {
			go processor.replaceStream(s, nil)
		}
	}
}

// onSetWarehouseSettings is called when the settings of a workspace data
// warehouse are changed.
func (processor *Processor) onSetWarehouseSettings(n state.SetWarehouseSettingsNotification) {
	ws, _ := processor.state.Workspace(n.Workspace)
	if n.Settings == nil {
		// Close the streams of the workspace.
		for _, c := range ws.Connections() {
			if s, ok := processor.streams[c.ID]; ok {
				go processor.replaceStream(s, nil)
			}
		}
		return
	}
	// Open the streams of the workspace if they are not already open.
	for _, c := range ws.Connections() {
		if _, ok := processor.streams[c.ID]; !ok && processor.isSuitableStream(c) {
			go processor.replaceStream(nil, c)
		}
	}
}

// replaceStream replaces the stream old with new, opening the new stream and
// closing the old one. If old is nil, it only opens and adds the new stream.
// If new is nil, it only closes and removes the old one.
func (processor *Processor) replaceStream(old *processorStream, new *state.Connection) {
	// Open to the new stream.
	if new != nil {
		var stream connector.StreamConnection
		for stream == nil {
			var err error
			stream, err = connector.RegisteredStream(new.Connector().Name).Open(
				context.Background(), &connector.StreamConfig{
					Role:     connector.DestinationRole,
					Settings: new.Settings,
				})
			if err != nil {
				// Wait and retry.
				log.Printf("[warning] cannot connect to stream %d", new.ID)
				time.Sleep(10 * time.Millisecond)
				processor.Lock()
				if processor.streams[new.ID] != old {
					processor.Unlock()
					return
				}
				processor.Unlock()
			}
		}
		ctx, cancel := context.WithCancel(processor.ctx)
		s := &processorStream{
			id:        new.ID,
			workspace: new.Workspace().ID,
			stream:    stream,
			ctx:       ctx,
			cancel:    cancel,
		}
		processor.Lock()
		if processor.streams[new.ID] != old {
			if err := stream.Close(); err != nil {
				log.Printf("[warning] an error occurred closing the stream %d: %s", new.ID, err)
			}
			processor.Unlock()
			cancel()
			return
		}
		processor.streams[new.ID] = s
		processor.Unlock()
		go processor.process(s)
	}
	// Close the old stream.
	if old != nil {
		old.cancel()
	}
}

// process processes the events received from the stream s.
// If the stream's context is cancelled, it closes the stream and returns.
func (p *Processor) process(s *processorStream) {

	streamName := "default stream"
	if s.id > 0 {
		streamName = "stream " + strconv.Itoa(s.id)
	}

	defer func() {
		if err := s.stream.Close(); err != nil {
			log.Printf("[error] cannot close stream %s: %s", streamName, err)
		}
		if p.geoLiteDB != nil {
			if err := p.geoLiteDB.Close(); err != nil {
				log.Printf("[error] cannot close GeoLite: %s", err)
			}
		}
	}()

	for {
		message, ack, err := s.stream.Receive()
		if err != nil {
			select {
			case <-s.ctx.Done():
				err := s.stream.Close()
				if err != nil {
					log.Printf("[warning] an error occurred closing the %s: %s", streamName, err)
				}
				return
			default:
				log.Printf("[error] cannot receive message from the %s: %s", streamName, err)
				continue
			}
		}
		err = p.processMessage(s.id, message)
		if err != nil {
			log.Printf("[error] cannot process message, received from the %s: %s", streamName, err)
			continue
		}
		ack()
	}

}

// processMessage processes a message received from the stream with identifier
// streamID. If an error occurs processing the message, it returns the error.
func (p *Processor) processMessage(streamID int, message []byte) error {

	r := bytes.NewReader(message)

	// Verify that message contains JSON objects and locate their offsets after the first object.
	// The last offset is equals to the message length.
	offsets := make([]int, 0, 2)
	dec := json.NewDecoder(r)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if tok != json.Delim('{') {
			p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("expecting JSON object"))
			return nil
		}
		depth := 1
		for depth > 0 {
			tok, err = dec.Token()
			if err != nil {
				p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("invalid JSON"))
				return nil
			}
			if d, ok := tok.(json.Delim); ok {
				switch d {
				case '}', ']':
					depth--
				case '{', '[':
					depth++
				}
			}
		}
		offsets = append(offsets, int(dec.InputOffset()))
	}

	// Read the message header.
	header := new(collector.MessageHeader)
	r.Reset(message)
	dec = json.NewDecoder(r)
	dec.DisallowUnknownFields()
	err := dec.Decode(header)
	if err != nil {
		p.observer.AddEvent(0, 0, streamID, nil, message, err)
		return nil
	}

	// Read the authorization header.
	auth, ok := header.Headers["Authorization"]
	if !ok {
		p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("missing 'Authorization' header"))
		return nil
	}
	if len(auth) > 1 {
		p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("too many 'Authorization' headers"))
		return nil
	}
	src, key, ok := parseBasicAuth(auth[0])
	if !ok {
		p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("invalid 'Authorization' header"))
		return nil
	}

	// Validate the server.
	var serverID int
	var server *state.Connection
	if key != "" {
		// message was sent by a server.
		if !isWellFormedConnectorKey(key) {
			p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("invalid authorization key"))
		}
		server, ok := p.server(key)
		if !ok {
			p.observer.AddEvent(0, 0, streamID, nil, message, errors.New("does not exist a server with the given key"))
			return nil
		}
		serverID = server.ID
	}

	// Validate the source.
	if src == "" {
		p.observer.AddEvent(0, serverID, streamID, nil, message, errors.New("missing source in 'Authorization' header"))
		return nil
	}
	sourceID, _ := strconv.Atoi(src)
	source, ok := p.source(sourceID, server)
	if !ok {
		p.observer.AddEvent(0, serverID, streamID, nil, message, errors.New("invalid source in 'Authorization' header"))
	}

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(header.Headers.Get("Content-Type"))
	if err != nil || mt != "application/json" || len(params) > 1 {
		p.observer.AddEvent(sourceID, serverID, streamID, nil, message, errors.New("Content-Type header must be 'application/json'"))
		return nil
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		p.observer.AddEvent(sourceID, serverID, streamID, nil, message, errors.New("Content-Type header charset must be 'utf-8'"))
		return nil
	}

	// Validate received at.
	if err != nil || header.ReceivedAt.IsZero() {
		p.observer.AddEvent(sourceID, serverID, streamID, nil, message, errors.New("invalid received at"))
		return nil
	}

	// Validate the remote host.
	ip, _, err := net.SplitHostPort(header.RemoteAddr)
	if err != nil {
		p.observer.AddEvent(sourceID, serverID, streamID, nil, message, errors.New("invalid remote address"))
		return nil
	}
	remoteIP := net.ParseIP(ip)
	if remoteIP == nil {
		p.observer.AddEvent(sourceID, serverID, streamID, nil, message, errors.New("invalid remote address"))
		return nil
	}

	now := time.Now().UTC()

	typ := source.Connector().Type
	typeString := strings.ToLower(typ.String())

	num := len(offsets) - 1
	events := make([]Event, num)

	for i := 0; i < num; i++ {

		event := &events[i]
		event.data = message[offsets[i]:offsets[i+1]]

		dec = json.NewDecoder(bytes.NewReader(event.data))
		err = dec.Decode(event)
		if err != nil {
			event.err = err
			continue
		}

		// Source.
		event.source = int32(source.ID)

		// AnonymousId.
		if _, err := uuid.Parse(event.AnonymousId); err != nil {
			event.err = errors.New("invalid anonymous id")
			continue
		}

		// Properties.
		if len(event.Properties) > 0 && !bytes.Equal(event.Properties, emptyProperties) {
			if event.Properties[0] != '{' {
				event.err = errors.New("properties is not a JSON object")
				continue
			}
			// Decode the properties.
			dec := json.NewDecoder(bytes.NewReader(event.Properties))
			dec.UseNumber()
			var properties map[string]any
			err = dec.Decode(&properties)
			if err != nil {
				event.err = fmt.Errorf("unexpected error decoding properties: %s", err)
				continue
			}
			// Encode the properties.
			var b strings.Builder
			enc := json.NewEncoder(&b)
			enc.SetIndent("", "")
			enc.SetEscapeHTML(false)
			err = enc.Encode(properties)
			if err != nil {
				event.err = fmt.Errorf("unexpected error encoding properties: %s", err)
				continue
			}
			s := b.String()
			event.properties = s[:len(s)-1] // remove the new line.
		} else {
			event.properties = "{}"
		}

		// Language.
		locale := culture.Locale(event.Language)
		if locale == nil {
			event.err = errors.New("unknown language code")
			continue
		}
		event.Language = locale.LanguageCode()

		// Target.
		if event.Target != "" {
			if typ == state.MobileType {
				event.err = errors.New("mobile cannot have target")
				continue
			} else if utf8.RuneCountInString(event.Target) > 2048 {
				event.err = errors.New("target is longer than 2048")
				continue
			} else if u, err := url.Parse(event.Target); err != nil {
				event.err = errors.New("target is not a valid URL")
				continue
			} else if u.Scheme != "https" && u.Scheme != "http" {
				event.err = errors.New("target must begin with 'http' or 'https'")
				continue
			}
		}

		// Text.
		if utf8.RuneCountInString(event.Text) > 120 {
			event.Text = abbreviate(event.Text, 120)
		}

		// IP.
		var requestIP net.IP
		if server != nil {
			if event.IP == "" {
				event.err = errors.New("IP address cannot be empty for server requests")
				continue
			}
			requestIP = net.ParseIP(event.IP)
			if requestIP == nil {
				event.err = errors.New("IP address is not valid")
			}
		} else {
			if event.IP != "" {
				event.err = fmt.Errorf("%s requests cannot have IP address", typeString)
			}
			requestIP = remoteIP
		}
		event.ip = requestIP.To16().String()

		// page.
		if typ == state.MobileType {
			if event.URL != "" {
				event.err = errors.New("mobile cannot have URL")
				continue
			}
		} else {
			if event.URL == "" {
				event.err = errors.New("IP address can be empty")
				continue
			}
			if utf8.RuneCountInString(event.URL) > 2048 {
				event.err = errors.New("URL is longer than 2048")
				continue
			}
			u, err := url.Parse(event.URL)
			if err != nil {
				event.err = errors.New("URL is not a valid URL")
				continue
			}
			if u.Scheme != "https" && u.Scheme != "http" {
				event.err = errors.New("URL must begin with 'http' or 'https'")
				continue
			}
			if u.Host != source.WebsiteHost {
				event.err = errors.New("URL cannot belong to the source website")
				continue
			}
			event.page.url = u.String()
			event.page.path = u.Path
			event.page.title = event.Title
			event.page.search = u.RawQuery
			if event.Referrer != "" {
				if typ == state.MobileType {
					event.err = errors.New("mobile cannot have referrer")
					continue
				}
				if utf8.RuneCountInString(event.Referrer) > 2048 {
					event.err = errors.New("referrer is longer than 2048")
					continue
				}
				u, err := url.Parse(event.Referrer)
				if err != nil {
					event.err = errors.New("referrer is not a valid URL")
					continue
				}
				if u.Scheme != "https" && u.Scheme != "http" {
					event.err = errors.New("referrer must begin with 'http' or 'https'")
					continue
				}
				event.page.referrer = u.String()
			}
		}

		// screen.
		if d := event.Screen.Density; 0 < d && d < 10 {
			event.screen.density = uint16(math.Round(d * 100))
		}
		if w, h := event.Screen.Width, event.Screen.Height; (0 < w && w <= math.MaxInt16) && (0 < h && h <= math.MaxInt16) {
			event.screen.width = uint16(w)
			event.screen.height = uint16(h)
		}

		// Timestamp and date.
		if event.Timestamp == "" {
			event.timestamp = header.ReceivedAt
		} else {
			event.timestamp, err = iso8601.ParseString(event.Timestamp)
			if err != nil {
				event.err = errors.New("timestamp is not in ISO8601 format")
				continue
			}
			event.timestamp = event.timestamp.UTC()
			if server != nil {
				if event.timestamp.After(now) {
					event.timestamp = header.ReceivedAt
				}
			} else {
				if t := event.timestamp; t.Add(-15*time.Minute).Before(now) || t.After(now) {
					event.timestamp = header.ReceivedAt
				}
			}
		}
		event.date = event.timestamp.Format(time.DateOnly)

		// SentAt.
		if event.SentAt == "" {
			event.sentAt = header.ReceivedAt
		} else {
			event.sentAt, err = iso8601.ParseString(event.Timestamp)
			if err != nil {
				event.err = errors.New("sentAt is not in ISO8601 format")
				continue
			}
			event.sentAt = event.sentAt.UTC()
		}

		// ReceivedAt.
		event.receivedAt = header.ReceivedAt

		// Location.
		if server == nil {
			if event.Country != "" {
				event.err = fmt.Errorf("country is required for %s", typeString)
				continue
			}
			if event.City != "" {
				event.err = fmt.Errorf("country is required for %s", typeString)
				continue
			}
		}
		if event.Country == "" || event.City == "" {
			if p.geoLiteDB != nil {
				city, err := p.geoLiteDB.City(requestIP)
				if err != nil {
					return err
				}
				event.location.city = city.City.Names["en"]
				c := culture.Country(event.Country)
				if c != nil {
					event.location.country.code = c.Code()
					event.location.country.name = c.Name()
				}
				event.location.latitude = city.Location.Latitude
				event.location.longitude = city.Location.Longitude
				event.location.timezone = city.Location.TimeZone
			}
		} else if event.Country != "" {
			c := culture.Country(event.Country)
			if c == nil {
				event.err = fmt.Errorf("unknown country code")
				continue
			}
			event.location.country.code = c.Code()
			event.location.country.name = c.Name()
		} else if utf8.RuneCountInString(event.City) > 50 {
			event.err = fmt.Errorf("city is longer than 50")
			continue
		}

		// UserAgent, DeviceType and Browser.
		if typ == state.MobileType {
			if event.userAgent != "" {
				event.err = fmt.Errorf("mobile cannot have user agent")
				continue
			}
			if dt := event.DeviceType; dt != "" && !isDeviceType(dt) {
				event.err = fmt.Errorf("device type must be 'Mobile', 'Tablet' or 'Desktop'")
				continue
			}
			event.os.name = event.OSName
			event.os.version = event.OSVersion
		} else {
			event.userAgent = header.Headers.Get("User-Agent")
			ua := useragent.New(event.userAgent)
			osInfo := ua.OSInfo()
			switch osInfo.Name {
			case "Mac OS X":
				event.os.name = "macOS"
			case "Android", "Windows", "iOS", "Linux", "ChromeOS":
				event.os.name = osInfo.Name
			default:
				event.os.name = "Other"
			}
			if utf8.RuneCountInString(osInfo.Version) <= 10 {
				event.os.version = osInfo.Version
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
			if ua.Mobile() {
				if h := strings.ToLower(event.userAgent); strings.Contains(h, "ipad") || strings.Contains(h, "tablet") {
					event.DeviceType = "tablet"
				} else {
					event.DeviceType = "mobile"
				}
			} else {
				event.DeviceType = "desktop"
			}
		}

	}

	// Add the events to the queue of the workspace.
	workspaceID := source.Workspace().ID
	p.Lock()
	queue, ok := p.queues[workspaceID]
	if !ok {
		queue = newQueue(p.state, workspaceID)
		p.queues[workspaceID] = queue
	}
	p.Unlock()
	queue.add(events)

	// Log the events for the observers.
	for i := 0; i < num; i++ {
		data := events[i].data
		events[i].data = nil
		err := events[i].err
		p.observer.AddEvent(sourceID, serverID, streamID, header, data, err)
	}

	return nil
}

// server returns the server with the given key.
func (p *Processor) server(key string) (*state.Connection, bool) {
	server, ok := p.state.ConnectionByKey(key)
	if !ok || !server.Enabled || server.Role != state.SourceRole {
		return nil, false
	}
	if typ := server.Connector().Type; typ != state.ServerType {
		return nil, false
	}
	return server, true
}

// source returns the source with identifier id. If server is not nil,
// checks if the source and the sever have the same workspace.
func (p *Processor) source(id int, server *state.Connection) (*state.Connection, bool) {
	if id < 1 || id > math.MaxInt32 {
		return nil, false
	}
	source, ok := p.state.Connection(id)
	if !ok || !source.Enabled || source.Role != state.SourceRole {
		return nil, false
	}
	if typ := source.Connector().Type; typ != state.MobileType && typ != state.WebsiteType {
		return nil, false
	}
	if server != nil && source.Workspace().ID != server.Workspace().ID {
		return nil, false
	}
	return source, true
}

// queue is the queue of the processed events.
type queue struct {
	sync.Mutex // for the events field.
	state      *state.State
	workspace  int
	events     []*Event
}

// newQueue returns a new queue that flushed events into the given data
// warehouse.
func newQueue(state *state.State, workspace int) *queue {
	q := &queue{state: state, workspace: workspace}
	// Start a goroutine that flushes the events every flushQueueTimeout seconds.
	go func() {
		ticker := time.NewTicker(flushQueueTimeout)
		for {
			select {
			case <-ticker.C:
				q.Lock()
				toFlush := q.events
				q.events = nil
				q.Unlock()
				go q.flush(toFlush)
			}
		}
	}()
	return q
}

// add adds events to the queue.
func (q *queue) add(events []Event) {
	q.Lock()
	var n int
	for i := 0; i < len(events); i++ {
		if events[i].err == nil {
			n++
		}
	}
	if n == 0 {
		q.Unlock()
		return
	}
	for i := 0; i < len(events); i++ {
		var event Event
		event = events[i]
		q.events = append(q.events, &event)
	}
	var toFlush []*Event
	if len(q.events) == maxEventsQueueLen {
		toFlush = q.events
		q.events = nil
	}
	q.Unlock()
	if toFlush != nil {
		go q.flush(toFlush)
	}
}

var batchEventsColumns = []string{
	"source",
	"anonymous_id",
	"user_id",
	"date",
	"timestamp",
	"sent_at",
	"received_at",
	"ip",
	"os_name",
	"os_version",
	"user_agent",
	"screen_density",
	"screen_width",
	"screen_height",
	"browser_name",
	"browser_other",
	"browser_version",
	"location_city",
	"location_country_code",
	"location_country_name",
	"location_latitude",
	"location_longitude",
	"device_type",
	"event",
	"language",
	"page_path",
	"page_referrer",
	"page_title",
	"page_url",
	"page_search",
	"utm_source",
	"utm_medium",
	"utm_campaign",
	"utm_term",
	"utm_content",
	"target",
	"text",
	"properties",
}

// flush flushes a batch of events to the data warehouse.
func (q *queue) flush(events []*Event) {
	if len(events) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(events))
RETRY:
	for {
		ws, ok := q.state.Workspace(q.workspace)
		if !ok || ws.Warehouse == nil {
			return
		}
		batch, err := ws.Warehouse.PrepareBatch(context.Background(), "events", batchEventsColumns)
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		for _, e := range events {
			err = batch.Append(
				e.source,
				e.AnonymousId,
				e.UserId,
				e.date,
				e.timestamp,
				e.sentAt,
				e.receivedAt,
				e.ip,
				e.os.name,
				e.os.version,
				e.userAgent,
				e.screen.density,
				e.screen.width,
				e.screen.height,
				e.browser.name,
				e.browser.other,
				e.browser.version,
				e.location.city,
				e.location.country.code,
				e.location.country.name,
				e.location.latitude,
				e.location.longitude,
				e.DeviceType,
				e.Event,
				e.Language,
				e.page.path,
				e.page.referrer,
				e.page.title,
				e.page.url,
				e.page.search,
				e.UTM.Source,
				e.UTM.Medium,
				e.UTM.Campaign,
				e.UTM.Term,
				e.UTM.Content,
				e.Target,
				e.Text,
				e.properties,
			)
			if err != nil {
				log.Printf("[error] cannot log events: %s", err)
				time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
				continue RETRY
			}
		}
		err = batch.Send()
		if err != nil {
			log.Printf("[error] cannot log events: %s", err)
			time.Sleep(time.Duration(rand.Intn(2000)) * time.Millisecond)
			continue
		}
		break
	}
}

// isWellFormedConnectorKey reports whether key is a well-formed key.
func isWellFormedConnectorKey(key string) bool {
	if len(key) != 32 {
		return false
	}
	for i := 0; i < 32; i++ {
		if k := key[i]; k < '0' || k > '9' && k < 'A' || k > 'Z' && k < 'a' || k > 'z' {
			return false
		}
	}
	return true
}

// makeUserId returns a new random user identifier.
func makeUserId() (string, error) {
	b := make([]byte, 4)
	_, err := crand.Read(b)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(binary.LittleEndian.Uint32(b)), 10), nil
}

// isDeviceType reports whether t is a device type.
func isDeviceType(t string) bool {
	switch t {
	case "Mobile", "Tablet", "Desktop":
		return true
	}
	return false
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
//
// This function is the http.parseBasicAuth of the Go standard library,
// copyright Go Authors.
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !asciiEqualFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", "", false
	}
	cs := string(c)
	username, password, ok = strings.Cut(cs, ":")
	if !ok {
		return "", "", false
	}
	return username, password, true
}

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}

// asciiEqualFold is strings.EqualFold, ASCII only. It reports whether s and t
// are equal, ASCII-case-insensitively.
//
// This function is the ascii.EqualFold internal function of the http package
// of the Go standard library, copyright Go Authors.
func asciiEqualFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if lower(s[i]) != lower(t[i]) {
			return false
		}
	}
	return true
}

// lower returns the ASCII lowercase version of b.
//
// This function is the ascii.lower internal function of the http package of
// the Go standard library, copyright Go Authors.
func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

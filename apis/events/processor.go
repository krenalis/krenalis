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
	"database/sql"
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

	"chichi/apis/postgres"
	"chichi/apis/state"
	"chichi/connector"

	"github.com/mssola/user_agent"
	"github.com/open2b/nuts/culture"
	"github.com/oschwald/geoip2-golang"
	"golang.org/x/text/unicode/norm"
)

const (
	dateTimeLayout    = "2006-01-02 15:04:05"
	flushQueueTimeout = 1 * time.Second // interval to flushEvents the queue
	geoLite2Path      = "GeoLite2-City.mmdb"
	maxEventSize      = 32 * 1024
	maxEventsQueueLen = 10000
	testGEOIP         = "79.9.108.176"
)

type Event struct {
	City           string
	Country        string
	Device         string
	DeviceType     string
	Event          string // "pageview", "click", ...
	IP             string
	Language       string // "it-IT"
	OSName         string
	OSVersion      string
	Referrer       string // "https://example.com"
	Target         string // "https://example.com"
	Text           string // "Add to cart"
	Timestamp      string
	Title          string // "Product X"
	URL            string // "https://example.com/product/x/y?x=10"
	UserAgent      string // "https://example.com/product/x/y?x=10"
	browser        string
	browserOther   string
	browserVersion string
	date           string
	domain         string // "example.com"
	path           string // "product/x/y"
	queryString    string // "x=10"
	source         int32
	user           uint32

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
}

// processorStream represents a stream used by the processor.
type processorStream struct {
	id     int
	stream connector.StreamConnection
	ctx    context.Context
	cancel context.CancelFunc
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

	// Process the events from the default stream.
	go processor.process(&processorStream{
		stream: defaultStream,
		ctx:    ctx,
	})

	st.AddListener(processor.onAddConnection)
	st.AddListener(processor.onDeleteConnection)
	st.AddListener(processor.onSetConnectionSettings)
	st.AddListener(processor.onSetConnectionStatus)
	st.AddListener(processor.onSetWarehouseSettings)

	return &processor, nil
}

// isSuitableStream reports whether c is a stream from which the processor must
// read events.
func (processor *Processor) isSuitableStream(c *state.Connection) bool {
	if c == nil || !c.Enabled || c.Role != state.SourceRole || len(c.Settings) == 0 {
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
			id:     new.ID,
			stream: stream,
			ctx:    ctx,
			cancel: cancel,
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
			log.Printf("cannot close stream %s: %s", streamName, err)
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
			log.Printf("cannot process message, received from the %s: %s", streamName, err)
			continue
		}
		ack()
	}

}

// processMessage processes a message received from the stream with identifier
// streamID. If an error occurs processing the message, it returns the error.
func (p *Processor) processMessage(streamID int, message []byte) error {

	r := bytes.NewReader(message)

	// Check that message contains JSON objects and determines their offsets after the first object.
	offsets := make([]int, 0, 1)
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
	header := new(MessageHeader)
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
	receivedAt, err := time.Parse(eventDateLayout, header.ReceivedAt)
	if err != nil || receivedAt.IsZero() {
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

	num := len(offsets)
	offsets = append(offsets, len(message))
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

		// Device.
		if _, err := base64.StdEncoding.DecodeString(event.Device); err != nil || len(event.Device) != 28 {
			event.err = errors.New("invalid device")
			continue
		}

		// Event.
		if typ == state.MobileType {
			// TODO(marco)
		} else {
			if e := event.Event; e != "pageview" && e != "click" {
				event.err = errors.New("unknown event name")
				continue
			}
		}

		// Language.
		locale := culture.Locale(event.Language)
		if locale == nil {
			event.err = errors.New("unknown language code")
			continue
		}
		event.Language = locale.LanguageCode()

		// Referrer.
		if event.Referrer != "" {
			if typ == state.MobileType {
				event.err = errors.New("mobile cannot have referrer")
				continue
			} else if utf8.RuneCountInString(event.Referrer) > 2048 {
				event.err = errors.New("referrer is longer than 2048")
				continue
			} else if u, err := url.Parse(event.Referrer); err != nil {
				event.err = errors.New("referrer is not a valid URL")
				continue
			} else if u.Scheme != "https" && u.Scheme != "http" {
				event.err = errors.New("referrer must begin with 'http' or 'https'")
				continue
			}
		}

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

		ctx := context.Background()

		// Get the user or create it if it does not exist.
		err = p.db.QueryRow(ctx, "SELECT id FROM users WHERE source = $1 AND device = $2", source.ID, event.Device).Scan(&event.user)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			err = p.db.QueryRow(ctx, "SELECT user FROM devices WHERE source = $1 AND id = $2", source.ID, event.Device).Scan(&event.user)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == sql.ErrNoRows {
				event.user, err = makeUserID()
				if err != nil {
					return err
				}
				_, err = p.db.Exec(ctx, "INSERT INTO users (source, id, device) VALUES($1, $2, $3)", source.ID, event.user, event.Device)
				if err != nil {
					return err
				}
			}
		}

		// Text.
		if utf8.RuneCountInString(event.Text) > 120 {
			event.Text = abbreviate(event.Text, 120)
		}

		// Title.
		if utf8.RuneCountInString(event.Title) > 120 {
			event.Title = abbreviate(event.Title, 120)
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
		if ip := requestIP.String(); ip == "127.0.0.1" || ip == "::1" {
			requestIP = net.ParseIP(testGEOIP)
		}

		// URL.
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
			event.domain, _, _ = strings.Cut(u.Host, ":")
			event.path = strings.TrimLeft(strings.TrimRight(u.Path, "/"), "/")
			event.queryString = u.RawQuery
		}

		// Timestamp.
		if event.Timestamp == "" {
			event.Timestamp = now.Format(dateTimeLayout)
		} else {
			t, err := time.Parse(dateTimeLayout, event.Timestamp)
			if err != nil {
				event.err = errors.New("URL cannot belong to the source website")
				continue
			}
			if server != nil {
				if t.After(now) {
					event.Timestamp = now.Format(dateTimeLayout)
				}
			} else {
				if t.Add(-15*time.Minute).Before(now) || t.After(now) {
					event.Timestamp = now.Format(dateTimeLayout)
				}
			}
		}
		event.date = event.Timestamp[0:10]

		// Country and city.
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
			geoDB, err := geoip2.Open(geoLite2Path)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if !errors.Is(err, fs.ErrNotExist) {
				defer geoDB.Close()
				city, err := geoDB.City(requestIP)
				if err != nil {
					return err
				}
				event.Country = city.Country.IsoCode
				event.City = city.City.Names["en"]
				geoDB.Close()
			}
		} else if event.Country != "" && culture.Country(event.Country) == nil {
			event.err = fmt.Errorf("unknown country code")
			continue
		} else if utf8.RuneCountInString(event.City) > 50 {
			event.err = fmt.Errorf("city is longer than 50")
			continue
		}

		// UserAgent, DeviceType.
		if typ == state.MobileType {
			if event.UserAgent != "" {
				event.err = fmt.Errorf("mobile cannot have user agent")
				continue
			}
			if dt := event.DeviceType; dt != "" && !isDeviceType(dt) {
				event.err = fmt.Errorf("device type must be 'Mobile', 'Tablet' or 'Desktop'")
				continue
			}
		} else {
			userAgent := header.Headers.Get("User-Agent")
			ua := user_agent.New(userAgent)
			osInfo := ua.OSInfo()
			switch osInfo.Name {
			case "Android", "Windows", "iOS", "MacOS", "Linux", "ChromeOS":
				event.OSName = osInfo.Name
			default:
				event.OSName = "Other"
			}
			if utf8.RuneCountInString(event.OSVersion) <= 10 {
				event.OSVersion = osInfo.Version
			}
			browserName, browserVersion := ua.Browser()
			switch browserName {
			default:
				event.browser = "Other"
				if len(browserName) <= 25 {
					event.browserOther = browserName
				}
			case "Chrome":
				event.browser = "Chrome"
			case "Safari":
				event.browser = "Safari"
			case "Edge":
				event.browser = "Edge"
			case "Firefox":
				event.browser = "Firefox"
			case "Samsung Internet":
				event.browser = "Samsung Internet"
			case "Opera":
				event.browser = "Opera"
			}
			if event.browser != "Other" || event.browserOther != "" {
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
					event.browserVersion = browserVersion
				}
			}
			if ua.Mobile() {
				if h := strings.ToLower(userAgent); strings.Contains(h, "ipad") || strings.Contains(h, "tablet") {
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

var batchEventsColumns = []string{"source", "date", "timestamp", "language", "os_name", "os_version", "browser",
	"browser_other", "browser_version", "device_type", "referrer2", "target", "event", "text", "domain", "path",
	"query_string", "title", "user", "country", "city"}

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
		for _, event := range events {
			err := batch.Append(event.source, event.date, event.Timestamp, event.Language, event.OSName,
				event.OSVersion, event.browser, event.browserOther, event.browserVersion, event.DeviceType,
				event.Referrer, event.Target, event.Event, event.Text, event.domain, event.path, event.queryString,
				event.Title, event.user, event.Country, event.City)
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
	}
}

// decodeEvent decodes an event read from body into a value pointed by event.
func decodeEvent(body io.Reader, event any) error {
	body = norm.NFC.Reader(io.LimitReader(body, maxEventSize))
	err := json.NewDecoder(body).Decode(event)
	if err != nil {
		return errBadRequest
	}
	n, err := body.Read([]byte(" "))
	if n > 0 {
		return errBadRequest
	}
	if err != nil && err != io.EOF {
		return err
	}
	return nil
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

// makeUserID returns a new random user identifier.
func makeUserID() (uint32, error) {
	b := make([]byte, 4)
	_, err := crand.Read(b)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
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

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

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

	_connector "chichi/connector"
	"chichi/pkg/open2b/sql"

	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
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

	// data and err are used during event processing.
	data []byte
	err  error
}

// eventProcessor processes events received from source streams and sent them
// to ClickHouse.
type eventProcessor struct {
	myDB     *sql.DB
	chDB     chDriver.Conn
	streams  []*eventProcessorStream
	queue    *queue
	observer *eventObserver
}

type eventProcessorStream struct {
	ID        int
	Connector string
	Settings  []byte
}

// newEventProcessor returns a new event eventProcessor.
func newEventProcessor(myDB *sql.DB, chDB chDriver.Conn, streams []*eventProcessorStream) *eventProcessor {
	processor := eventProcessor{
		myDB:     myDB,
		streams:  streams,
		queue:    newQueue(chDB),
		observer: newEventObserver(),
	}
	return &processor
}

// Run executes the event eventProcessor p.
// It should be called in its own goroutine.
func (p *eventProcessor) Run(ctx context.Context) {
	for _, s := range p.streams {
		stream, err := _connector.RegisteredEventStream(s.Connector).Connect(ctx, &_connector.EventStreamConfig{
			Role:     _connector.SourceRole,
			Settings: s.Settings,
		})
		if err != nil {
			log.Printf("cannot connector to event stream connection %d: %s", s.ID, err)
		}
		go p.processStream(ctx, s.ID, stream)
	}
}

// processStream processes the events received from the stream with the given
// identifier and connection. When the context is canceled it closes the
// connection and returns.
func (p *eventProcessor) processStream(ctx context.Context, id int, connection _connector.EventStreamConnection) {

	defer func() {
		if err := connection.Close(); err != nil {
			log.Printf("cannot close stream %d: %s", id, err)
		}
	}()

	// Process the message received from the stream.
	for {
		message, ack, err := connection.Receive()
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				select {
				case <-ctx.Done():
					continue
				default:
				}
			}
			log.Printf("cannot receive message from stream %d: %s", id, err)
			continue
		}
		err = p.processMessage(id, message)
		if err != nil {
			log.Printf("cannot process message, received from stream %d: %s", id, err)
			continue
		}
		ack()
	}

}

// processMessage processes a message from the given stream. If an error occurs
// processing the message, it returns the error.
func (p *eventProcessor) processMessage(stream int, message []byte) error {

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
			p.observer.AddEvent(0, 0, stream, nil, message, errors.New("expecting JSON object"))
			return nil
		}
		depth := 1
		for depth > 0 {
			tok, err = dec.Token()
			if err != nil {
				p.observer.AddEvent(0, 0, stream, nil, message, errors.New("invalid JSON"))
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
		p.observer.AddEvent(0, 0, stream, nil, message, err)
		return nil
	}

	// Read the authorization header.
	auth, ok := header.Headers["Authorization"]
	if !ok {
		p.observer.AddEvent(0, 0, stream, nil, message, errors.New("missing 'Authorization' header"))
		return nil
	}
	if len(auth) > 1 {
		p.observer.AddEvent(0, 0, stream, nil, message, errors.New("too many 'Authorization' headers"))
		return nil
	}
	src, key, ok := parseBasicAuth(auth[0])
	if !ok {
		p.observer.AddEvent(0, 0, stream, nil, message, errors.New("invalid 'Authorization' header"))
		return nil
	}

	// Validate the server.
	var server int
	if key != "" {
		// message was sent by a server.
		if !isWellFormedConnectorKey(key) {
			p.observer.AddEvent(0, 0, stream, nil, message, errors.New("invalid authorization key"))
		}
		err = p.myDB.QueryRow("SELECT `c`.`id`\n"+
			"FROM `connections_keys` AS `k`\n"+
			"INNER JOIN `connections` AS `c` ON `k`.`connection` = `c`.`id`\n"+
			"WHERE `c`.`type` = 'Server' AND `c`.`role` = 'Source' AND `k`.`key` = ?", key).Scan(&server)
		if err != nil {
			if err == sql.ErrNoRows {
				p.observer.AddEvent(0, 0, stream, nil, message, errors.New("does not exist a server with the given key"))
				return nil
			}
			return err
		}
	}

	// Validate the source.
	if src == "" {
		p.observer.AddEvent(0, server, stream, nil, message, errors.New("missing source in 'Authorization' header"))
		return nil
	}
	source, _ := strconv.Atoi(src)
	if source <= 0 || source > math.MaxInt32 {
		p.observer.AddEvent(0, server, stream, nil, message, errors.New("invalid source in 'Authorization' header"))
		return nil
	}
	var typ ConnectorType
	var websiteHost string
	err = p.myDB.QueryRow("SELECT CAST(`type` AS UNSIGNED), `websiteHost`\n"+
		"FROM `connections`\n"+
		"WHERE `id` = ? AND `type` IN ('Mobile', 'Website') AND `role` = 'Source'", source).
		Scan(&typ, &websiteHost)
	if err != nil {
		if err == sql.ErrNoRows {
			p.observer.AddEvent(0, server, stream, nil, message, errors.New("source does not exist"))
			return nil
		}
		return err
	}

	serverRequest := server != 0

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(header.Headers.Get("Content-Type"))
	if err != nil || mt != "application/json" || len(params) > 1 {
		p.observer.AddEvent(source, server, stream, nil, message, errors.New("Content-Type header must be 'application/json'"))
		return nil
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		p.observer.AddEvent(source, server, stream, nil, message, errors.New("Content-Type header charset must be 'utf-8'"))
		return nil
	}

	// Validate received at.
	receivedAt, err := time.Parse(header.ReceivedAt, eventDateLayout)
	if err != nil || receivedAt.IsZero() {
		p.observer.AddEvent(source, server, stream, nil, message, errors.New("invalid received at"))
		return nil
	}

	// Validate the remote host.
	ip, _, err := net.SplitHostPort(header.RemoteAddr)
	if err != nil {
		p.observer.AddEvent(source, server, stream, nil, message, errors.New("invalid remote address"))
		return nil
	}
	remoteIP := net.ParseIP(ip)
	if remoteIP == nil {
		p.observer.AddEvent(source, server, stream, nil, message, errors.New("invalid remote address"))
		return nil
	}

	now := time.Now().UTC()

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
		event.source = int32(source)

		// Device.
		if _, err := base64.StdEncoding.DecodeString(event.Device); err != nil || len(event.Device) != 28 {
			event.err = errors.New("invalid device")
			continue
		}

		// Event.
		if typ == MobileType {
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
			if typ == MobileType {
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
			if typ == MobileType {
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

		// Get the user or create it if it does not exist.
		err = p.myDB.QueryRow("SELECT `id` FROM `users` WHERE `source` = ? AND `device` = ?", source, event.Device).Scan(&event.user)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			err = p.myDB.QueryRow("SELECT `user` FROM `devices` WHERE `source` = ? AND `id` = ?", source, event.Device).Scan(&event.user)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == sql.ErrNoRows {
				event.user, err = makeUserID()
				if err != nil {
					return err
				}
				_, err = p.myDB.Exec("INSERT INTO `users` SET `source` = ?, `id` = ?, `device` = ?", source, event.user, event.Device)
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
		if serverRequest {
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
		if typ == MobileType {
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
			if u.Host != websiteHost {
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
			if serverRequest {
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
		if !serverRequest {
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
		if typ == MobileType {
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

	// Add the events to the queue.
	p.queue.add(events)

	// Log the events for the observers.
	for i := 0; i < num; i++ {
		data := events[i].data
		events[i].data = nil
		err := events[i].err
		p.observer.AddEvent(source, server, stream, header, data, err)
	}

	return nil
}

// queue is the queue of the processed events.
type queue struct {
	eventsMu sync.Mutex
	events   []*Event
	chDB     chDriver.Conn
}

// newQueue returns a new queue that flushed events into the database chDB.
func newQueue(chDB chDriver.Conn) *queue {
	q := &queue{chDB: chDB}
	// Start a goroutine that flushes the events every flushQueueTimeout seconds.
	go func() {
		ticker := time.NewTicker(flushQueueTimeout)
		for {
			select {
			case <-ticker.C:
				q.eventsMu.Lock()
				toFlush := q.events
				q.events = nil
				q.eventsMu.Unlock()
				go q.flush(toFlush)
			}
		}
	}()
	return q
}

// add adds events to the queue.
func (q *queue) add(events []Event) {
	q.eventsMu.Lock()
	var n int
	for i := 0; i < len(events); i++ {
		if events[i].err == nil {
			n++
		}
	}
	if n == 0 {
		q.eventsMu.Unlock()
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
	q.eventsMu.Unlock()
	if toFlush != nil {
		go q.flush(toFlush)
	}
}

// flush flushes a batch of events to ClickHouse.
func (q *queue) flush(events []*Event) {
	if len(events) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(events))
RETRY:
	for {
		batch, err := q.chDB.PrepareBatch(context.Background(), "INSERT INTO `events`\n"+
			"(`source`, `date`, `timestamp`, `language`, `osName`, `osVersion`, `browser`, `browserOther`,"+
			" `browserVersion`, `deviceType`, `referrer`, `target`, `event`, `text`, `domain`, `path`,"+
			" `queryString`, `title`, `user`, `country`, `city`)")
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

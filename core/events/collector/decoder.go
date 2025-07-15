//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package collector

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"iter"
	"maps"
	"math"
	"mime"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/avct/uasurfer"
	"github.com/google/uuid"
	"github.com/oschwald/maxminddb-golang"
	"github.com/relvacode/iso8601"
	"golang.org/x/text/unicode/norm"
)

var errPayloadTooLarge = errors.BadRequest("body too large")
var errReadBody = errors.BadRequest("failed to read body")

type decoder struct {
	payload *bytes.Buffer
	dec     json.Decoder
	batch   json.Value
	maxmind *maxminddb.Reader

	receivedAt time.Time
	remoteAddr struct {
		ip  net.IP
		str string
	}
	userAgent  string
	sentAt     time.Time
	writeKey   string
	connection int
	context    map[string]any
	typ        string
}

// newDecoder returns a new decoder.
//
// In case of an error, it returns the following errors:
//   - errMethodNotAllowed: if the HTTP method is not allowed,
//   - errNotFound: if the requested URL does not exist,
//   - errBadRequest: if the request is not valid,
//   - errPayloadTooLarge: if the body exceeds maxRequestSize,
//   - a badRequestError: if the request's body is not valid.
func newDecoder(r *http.Request) (*decoder, error) {
	d := &decoder{}
	err := d.Reset(r)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Connection returns the connection property and a boolean indicating whether
// the property is present.
func (d *decoder) Connection() (int, bool) {
	if d.connection == 0 {
		return 0, false
	}
	return d.connection, true
}

// Events returns an iterator to iterate over events. connectionID and
// connectionType represent the identifier and type, respectively, of the source
// connection from which the events are received.
//
// For malformed errors, it returns nil and the corresponding error.
func (d *decoder) Events(connectionID int, connectionType state.ConnectorType) iter.Seq2[events.Event, error] {
	return func(yield func(events.Event, error) bool) {
		if d.typ != "batch" {
			// Decode a single event.
			var event events.Event
			var err error
			if k := d.dec.PeekKind(); k == json.Object {
				event, err = d.decodeEvent(connectionID, connectionType)
			} else {
				err = errors.BadRequest("expected an object for the event, but found %s instead", k)
			}
			yield(event, err)
			return
		}
		// Decode a batch of events.
		_ = d.dec.SkipToken() // skip '['.
		for {
			k := d.dec.PeekKind()
			switch k {
			case ']':
				return
			case '{':
			default:
				_ = d.dec.SkipValue()
				if !yield(nil, errors.BadRequest("expected an object for the event, but found %s instead", k)) {
					return
				}
				continue
			}
			event, err := d.decodeEvent(connectionID, connectionType)
			if !yield(event, err) {
				return
			}
		}
	}
}

// Reset resets the decoder, preparing it to parse a new request, and parses the
// given request.
//
// In case of an error, it returns the following errors:
//   - errMethodNotAllowed: if the HTTP method is not allowed,
//   - errNotFound: if the requested URL does not exist,
//   - errBadRequest: if the request is not valid,
//   - errPayloadTooLarge: if the body exceeds maxRequestSize,
//   - a badRequestError: if the request's body is not valid.
func (d *decoder) Reset(r *http.Request) error {

	if r.Method != "POST" {
		return errMethodNotAllowed
	}

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || (mt != "application/json" && mt != "text/plain") || len(params) > 1 {
		return errors.BadRequest("request's content type must be 'application/json' or 'text/plain'")
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		return errors.BadRequest("request's content type charset must be 'utf-8'")
	}

	// Validate the content length.
	if cl := r.Header.Get("Content-Length"); cl != "" {
		length, _ := strconv.Atoi(cl)
		if length < 1 || length > maxRequestSize {
			return errors.BadRequest("request's content length must be in the range [1,%d]", maxRequestSize)
		}
	}

	d.batch = nil
	d.receivedAt = time.Now().UTC()

	// If an 'X-Forwarded-For' header was provided, get the request address from
	// there.
	if ff := r.Header.Get("X-Forwarded-For"); ff != "" {
		clientIP, _, _ := strings.Cut(ff, ",")
		clientIP = strings.TrimSpace(clientIP)
		ip, str, err := parseIP(clientIP)
		if err != nil {
			return errors.BadRequest("the address specified in the 'X-Forwarded-For' header is not a valid IPv4 address")
		}
		d.remoteAddr.ip = ip
		d.remoteAddr.str = str
	}

	// If the address wasn't provided through the 'X-Forwarded-For' header, get
	// it from the request's RemoteAddr field.
	if d.remoteAddr.ip == nil {
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		ip, str, err := parseIP(host)
		if err != nil {
			return errors.BadRequest("only IP version 4 is supported")
		}
		d.remoteAddr.ip = ip
		d.remoteAddr.str = str
	}

	d.userAgent = r.Header.Get("User-Agent")
	d.sentAt = time.Time{}
	d.writeKey = ""
	d.connection = 0
	d.context = nil

	path, _ := strings.CutPrefix(r.URL.Path, "/events")
	switch path {
	case "":
	case "/alias", "/group", "/identify", "/page", "/screen", "/track":
		d.typ = path[1:]
	default:
		return errors.NotFound("")
	}

	// Read the body and check that is not be longer than maxRequestSize bytes and,
	// it is a streaming of JSON objects, otherwise return the errBadRequest error.
	lr := &io.LimitedReader{R: r.Body, N: maxRequestSize + 1}
	body, err := io.ReadAll(norm.NFC.Reader(lr))
	if err != nil {
		return errReadBody
	}
	if lr.N == 0 {
		return errPayloadTooLarge
	}
	if len(body) == 0 {
		return errors.BadRequest("request's body is empty")
	}
	d.payload = bytes.NewBuffer(body)
	d.dec.Reset(d.payload)

	if d.typ != "" {
		// It is a single-event request.
		d.sentAt = d.receivedAt
		return nil
	}
	kind := d.dec.PeekKind()
	if kind == '[' {
		// It is a batch-event request with a JSON array as body.
		d.typ = "batch"
		d.sentAt = d.receivedAt
		return nil
	}
	if kind != '{' {
		return errors.BadRequest("request's content is not a valid JSON object or array")
	}

	// It is either a single-event or a batch-event request, depending on the "batch" property.
	err = d.dec.SkipToken() // Skip the '{' token.
	if err != nil {
		return errRead(err)
	}
	var tok json.Token
	for {
		tok, err = d.dec.ReadToken()
		if err != nil {
			return errRead(err)
		}
		if tok.Kind() == '}' {
			break
		}
		key := tok.String()
		switch key {
		case "batch":
			batch, err := d.dec.ReadValue()
			if err != nil {
				return errRead(err)
			}
			if !batch.IsArray() {
				return errors.BadRequest("property 'batch' is not a valid array")
			}
			d.batch = batch
		case "context":
			kind := d.dec.PeekKind()
			if kind != '{' {
				return errors.BadRequest("property 'context' is not a valid object")
			}
			d.context, err = d.decodeContext(true)
			if err != nil {
				return err
			}
		case "sentAt":
			if !d.sentAt.IsZero() {
				return errors.BadRequest("property 'sentAt' is specified multiple times")
			}
			if tok, _ = d.dec.ReadToken(); tok.Kind() != '"' {
				return errors.BadRequest("property 'sentAt' is not a valid string")
			}
			d.sentAt, err = iso8601.ParseString(tok.String())
			if err != nil {
				return errors.BadRequest("property 'sentAt' is not a valid ISO 8601 timestamp")
			}
			d.sentAt = d.sentAt.UTC()
			if y := d.sentAt.Year(); y < 1 || y > 9999 {
				return errors.BadRequest("property 'sentAt' has an invalid year value")
			}
		case "writeKey":
			if d.writeKey != "" {
				return errors.BadRequest("property 'writeKey' is specified multiple times")
			}
			if tok, _ = d.dec.ReadToken(); tok.Kind() != '"' {
				return errors.BadRequest("property 'writeKey' is not a valid string")
			}
			d.writeKey = tok.String()
			if d.writeKey == "" {
				return errors.BadRequest("property 'writeKey' cannot be empty")
			}
		case "connection":
			if tok, _ = d.dec.ReadToken(); tok.Kind() != '0' {
				return errors.BadRequest("property 'connection' is not a number")
			}
			connection, _ := tok.Int()
			if connection < 1 || connection > math.MaxInt32 {
				return errors.BadRequest("property 'connection' is not a valid connection identifier")
			}
			d.connection = connection
		}
	}
	if d.batch == nil {
		// It is a single-event request. Reparse the entire request body
		d.payload = bytes.NewBuffer(body)
	} else {
		// It is a batch-event request. Parse only the slice of events.
		d.typ = "batch"
		d.payload = bytes.NewBuffer(d.batch)
	}
	d.dec.Reset(d.payload)

	if d.sentAt.IsZero() {
		d.sentAt = d.receivedAt
	}

	return nil
}

func (d *decoder) SetMaxMindDB(db *maxminddb.Reader) {
	d.maxmind = db
}

func (d *decoder) WriteKey() string {
	return d.writeKey
}

// decodeEvent decodes and returns an event.
func (d *decoder) decodeEvent(connection int, connectionType state.ConnectorType) (events.Event, error) {

	_ = d.dec.SkipToken() // Skip '{'.

	skipOut := true
	defer func() {
		if skipOut {
			_ = d.dec.SkipOut()
		}
	}()

	var name string
	var event = map[string]any{
		"connection": connection,
	}
	var context map[string]any

	var err error
	var tok json.Token
	var kind json.Kind

	for {
		tok, _ = d.dec.ReadToken()
		kind = tok.Kind()
		if kind == '}' {
			skipOut = false
			break
		}
		if kind == json.Invalid {
			return nil, errors.BadRequest("unexpected invalid token while decoding an event")
		}
		name = tok.String()
		kind = d.dec.PeekKind()
		switch name {
		case "anonymousId", "category", "groupId", "messageId", "originalTimestamp", "timestamp", "userId":
			if kind == 'n' {
				if _, ok := event[name]; ok {
					return nil, errors.BadRequest("property '%s' is specified multiple times", name)
				}
				_ = d.dec.SkipValue()
				continue
			}
			fallthrough
		case "channel", "event", "name", "sentAt", "type", "previousId":
			if _, ok := event[name]; ok {
				return nil, errors.BadRequest("property '%s' is specified multiple times", name)
			}
			if kind != '"' {
				return nil, errors.BadRequest("property '%s' is not a valid string", name)
			}
			tok, _ = d.dec.ReadToken()
			s := tok.String()
			if s == "" {
				continue
			}
			switch name {
			case "messageId":
				id := makeEventID(connection, s)
				event["id"] = id.String()
				event["messageId"] = s
			case "sentAt":
				sentAt, err := iso8601.ParseString(s)
				if err != nil {
					return nil, errors.BadRequest("property 'sentAt' is not a valid ISO 8601 timestamp")
				}
				sentAt = sentAt.UTC()
				if y := sentAt.Year(); y < 1 || y > 9999 {
					return nil, errors.BadRequest("property 'sentAt' has an invalid year value")
				}
				event["sentAt"] = sentAt
			case "originalTimestamp":
				timestamp, err := iso8601.ParseString(s)
				if err != nil {
					return nil, errors.BadRequest("property 'originalTimestamp' is not a valid ISO 8601 timestamp")
				}
				timestamp = timestamp.UTC()
				if y := timestamp.Year(); y < 1 || y > 9999 {
					return nil, errors.BadRequest("property 'originalTimestamp' has an invalid year value")
				}
				event["originalTimestamp"] = timestamp
			case "timestamp":
				timestamp, err := iso8601.ParseString(s)
				if err != nil {
					return nil, errors.BadRequest("property 'timestamp' is not a valid ISO 8601 timestamp")
				}
				timestamp = timestamp.UTC()
				if y := timestamp.Year(); y < 1 || y > 9999 {
					return nil, errors.BadRequest("property 'timestamp' has an invalid year value")
				}
				event["timestamp"] = timestamp
			case "type":
				event["type"] = s
				switch s {
				case "track", "page", "screen", "identify", "group", "alias":
				default:
					return nil, errors.BadRequest("property 'type' is not a valid event type")
				}
			default:
				event[name] = s
			}
		case "traits", "properties":
			if _, ok := event[name]; ok {
				return nil, errors.BadRequest("property '%s' is specified multiple times", name)
			}
			if kind == 'n' {
				continue
			}
			if kind != '{' {
				return nil, errors.BadRequest("property '%s' is not a valid object", name)
			}
			event[name], _ = d.dec.ReadValue()
		case "context":
			if _, ok := event["context"]; ok {
				return nil, errors.BadRequest("property 'context' is specified multiple times")
			}
			if kind != '{' {
				return nil, errors.BadRequest("property 'context' is not an valid object")
			}
			context, err = d.decodeContext(false)
			if err != nil {
				return nil, err
			}
			event["context"] = context
		default:
			_ = d.dec.SkipValue()
		}
	}

	if context == nil {
		context = map[string]any{}
		event["context"] = context
	}

	// Type.
	typ, ok := event["type"].(string)
	if !ok {
		if d.typ == "" {
			return nil, errors.BadRequest("property 'type' is required for a single-event request")
		}
		if d.typ == "batch" {
			return nil, errors.BadRequest("property 'type' is required for a batch request")
		}
		event["type"] = d.typ
		typ = d.typ
	}

	var (
		isAlias    = typ == "alias"
		isIdentify = typ == "identify"
		isGroup    = typ == "group"
		isPage     = typ == "page"
		isScreen   = typ == "screen"
		isTrack    = typ == "track"
	)

	// AnonymousId and UserId.
	if _, ok := event["anonymousId"]; !ok {
		if _, ok := event["userId"]; !ok {
			if isIdentify || isAlias {
				return nil, errors.BadRequest("property 'userId' is required for an %s event", typ)
			}
			return nil, errors.BadRequest("either 'anonymousId' or 'userId' properties are required for a %s event", typ)
		}
		event["anonymousId"] = uuid.NewString()
	}

	// Category.
	if !isPage {
		if _, ok := event["category"]; ok {
			return nil, errors.BadRequest("property 'category' is not permitted for a %s event", typ)
		}
	}

	// UserAgent.
	if ua, ok := context["userAgent"].(string); ok {
		if ua == "N/A" {
			// If the user agent is set to "N/A", this means that the user agent is
			// not applicable to this event, so it should not be taken from the HTTP
			// request either and the event should remain without a user agent.
			delete(context, "userAgent")
		} else {
			// Simply keep the user agent passed in the context.
		}
	} else {
		// User agent not provided in context, so it must be read from the
		// request.
		context["userAgent"] = d.userAgent
	}

	// Browser and OS.
	var hasBrowser, hasOS bool
	if browser, ok := context["browser"].(map[string]any); ok {
		normalizeContextBrowser(browser)
		hasBrowser = true
	}
	if os, ok := context["os"].(map[string]any); ok {
		normalizeContextOS(os)
		hasOS = true
	}
	if ua, ok := context["userAgent"].(string); ok && (!hasBrowser || !hasOS) {
		contextBrowser, contextOS := parseUserAgent(ua)
		if _, ok := context["browser"]; !ok {
			context["browser"] = contextBrowser
		}
		if _, ok := context["os"]; !ok {
			context["os"] = contextOS
		}
	}

	// IP.
	var requestIP net.IP
	if ip, ok := context["ip"].(string); ok {
		netIP, strIP, err := parseIP(ip)
		if err != nil {
			return nil, errors.BadRequest("property 'ip' is not a valid IP address")
		}
		if netIP.Equal(net.IPv4zero) {
			// If the IP is set to "0.0.0.0" (the zero IP), this means that the
			// IP is not applicable to this event (for example, in the case of
			// an event sent from a server generated by the server itself, for
			// which there is no client IP and the server IP has no meaning), so
			// it should not be taken from the HTTP request either and the event
			// should remain without a IP.
			delete(context, "ip")
		} else {
			requestIP, context["ip"] = netIP, strIP
		}
	} else {
		requestIP, context["ip"] = d.remoteAddr.ip, d.remoteAddr.str
	}

	// Location.
	if _, ok := context["location"]; !ok && requestIP != nil && d.maxmind != nil {
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
		if err := d.maxmind.Lookup(requestIP, &record); err == nil {
			loc := map[string]any{}
			if city := record.City.Names.EN; city != "" {
				loc["city"] = city
			}
			if lat := record.Location.Latitude; lat != 0 {
				loc["latitude"] = lat
			}
			if long := record.Location.Longitude; long != 0 {
				loc["longitude"] = long
			}
			if code, ok := countryCode(record.Country.IsoCode); ok {
				loc["country"] = code
			}
			if len(loc) > 0 {
				context["location"] = loc
			}
		}
	}

	// Screen.
	if screen, ok := context["screen"].(map[string]any); ok {
		if w, ok := screen["width"].(int); ok && (w <= 0 || w > math.MaxInt16) {
			return nil, errors.BadRequest("screen width value is outside the allowed range [1, 32767]")
		}
		if h, ok := screen["height"].(int); ok && (h <= 0 || h > math.MaxInt16) {
			return nil, errors.BadRequest("screen height value is outside the allowed range [1, 32767]")
		}
		if d, ok := screen["density"].(decimal.Decimal); ok && d.Sign() != 1 {
			return nil, errors.BadRequest("screen density value must be a positive number")
		}
	}

	// Event.
	if _, ok := event["event"]; ok {
		if !isTrack {
			return nil, errors.BadRequest("property 'event' is not permitted for a %s event", typ)
		}
	} else {
		if isTrack {
			return nil, errors.BadRequest("property 'event' is required for a track event")
		}
	}

	// GroupId.
	if _, ok := event["groupId"]; ok {
		if !isGroup {
			return nil, errors.BadRequest("property 'groupId' is not permitted for a %s event", typ)
		}
	} else {
		if isGroup {
			return nil, errors.BadRequest("property 'groupId' is required for a group event")
		}
	}

	// Id and MessageId.
	if _, ok := event["messageId"]; !ok {
		messageId := uuid.NewString()
		id := makeEventID(connection, messageId)
		event["id"] = id.String()
		event["messageId"] = messageId
	}

	// Name.
	if !isScreen && !isPage {
		if _, ok := event["name"]; ok {
			return nil, errors.BadRequest("property 'name' is not permitted for a %s event", typ)
		}
	}

	// PreviousId.
	if isAlias {
		if _, ok := event["previousId"]; !ok {
			return nil, errors.BadRequest("property 'previousId' is required for an alias event")
		}
	} else {
		if _, ok := event["previousId"]; ok {
			return nil, errors.BadRequest("property 'previousId' is not permitted for a %s event", typ)
		}
	}

	// Properties.
	if isPage || isScreen || isTrack {
		if _, ok := event["properties"]; !ok {
			event["properties"] = json.Value("{}")
		}
	} else {
		if _, ok := event["properties"]; ok {
			return nil, errors.BadRequest("property 'properties' is not permitted for a %s event", typ)
		}
	}

	// ReceivedAt.
	event["receivedAt"] = d.receivedAt.Truncate(time.Millisecond)

	// SentAt.
	sentAt, ok := event["sentAt"].(time.Time)
	if !ok {
		event["sentAt"] = d.sentAt
		sentAt = d.sentAt
	}

	// Timestamp and OriginalTimestamp.
	if _, ok := event["originalTimestamp"].(time.Time); ok {
		if _, ok := event["timestamp"].(time.Time); !ok {
			return nil, errors.BadRequest("property 'timestamp' is required if the property 'originalTimestamp' is present")
		}
	} else if timestamp, ok := event["timestamp"].(time.Time); ok {
		event["originalTimestamp"] = timestamp
		skew := d.receivedAt.Sub(sentAt)
		timestamp = timestamp.Add(skew)
		if y := timestamp.Year(); 1 <= y && y <= 9999 {
			event["timestamp"] = timestamp.Truncate(time.Millisecond)
		} else {
			event["timestamp"] = d.receivedAt.Truncate(time.Millisecond)
		}
	} else {
		event["timestamp"] = d.receivedAt.Truncate(time.Millisecond)
		event["originalTimestamp"] = event["timestamp"]
	}

	// Traits.
	if isIdentify || isGroup {
		if _, ok := event["traits"]; !ok {
			event["traits"] = json.Value("{}")
		}
	} else {
		if _, ok := event["traits"]; ok {
			return nil, errors.BadRequest("property 'traits' must be specified as 'context.traits' for a %s event", typ)
		}
		if traits, ok := context["traits"]; ok {
			event["traits"] = traits
			delete(context, "traits")
		} else {
			event["traits"] = json.Value("{}")
		}
	}

	// The context must be treated like the other fields: if it contains no
	// properties, the entire field must be removed.
	if context, ok := event["context"].(map[string]any); ok && len(context) == 0 {
		delete(event, "context")
	}

	return event, nil
}

// decodeContext decodes and returns a context. isDefault indicates if it is the
// default context. The first token must be '{'.
func (d *decoder) decodeContext(isDefault bool) (map[string]any, error) {

	_ = d.dec.SkipToken() // skip the first token.

	var kind json.Kind
	var tok json.Token
	var name string

	var context = map[string]any{}
	var err error

	for {
		tok, _ = d.dec.ReadToken()
		if tok.Kind() == '}' {
			break
		}
		name = tok.String()
		if _, ok := context[name]; ok {
			return nil, errors.BadRequest("property 'context.%s' is specified multiple times", name)
		}
		kind = d.dec.PeekKind()
		switch name {
		case "direct":
			if kind != 't' && kind != 'f' {
				return nil, errors.BadRequest("property 'context.direct' is not a valid boolean")
			}
			context[name] = kind == 't'
			_ = d.dec.SkipValue()
		case "sessionStart":
			if kind != 't' && kind != 'f' {
				return nil, errors.BadRequest("property 'context.sessionStart' is not a valid boolean")
			}
			if start := kind == 't'; start {
				if session, ok := context["session"].(map[string]any); ok {
					session["start"] = start
				} else {
					context["session"] = map[string]any{"start": start}
				}
			}
			_ = d.dec.SkipValue()
		case "ip", "locale", "groupId", "timezone", "userAgent":
			if kind != '"' {
				return nil, errors.BadRequest("property 'context.%s' is not a valid string", name)
			}
			tok, _ = d.dec.ReadToken()
			s := tok.String()
			if name == "locale" {
				s, _ = localeCode(s)
			}
			if s != "" {
				context[name] = s
			}
		case "sessionId":
			if kind != '0' {
				return nil, errors.BadRequest("property 'context.sessionId' is not a valid number")
			}
			tok, _ = d.dec.ReadToken()
			sessionId, err := tok.Int()
			if err != nil {
				return nil, errors.BadRequest("property 'context.sessionId' is not a 64-bit integer")
			}
			if session, ok := context["session"].(map[string]any); ok {
				session["id"] = sessionId
			} else {
				context["session"] = map[string]any{"id": sessionId}
			}
		case "traits":
			if kind == 'n' {
				continue
			}
			if kind != '{' {
				return nil, errors.BadRequest("property 'context.traits' is not a valid object")
			}
			context["traits"], _ = d.dec.ReadValue()
		default:
			section, ok := contextSections[name]
			if !ok {
				_ = d.dec.SkipValue()
				continue
			}
			if d.dec.PeekKind() != '{' {
				return nil, errors.BadRequest("property 'context.%s' is not a valid object", section.name)
			}
			v, err := d.decodeContextSection(section, isDefault)
			if err != nil {
				return nil, err
			}
			if v != nil {
				context[name] = v
			}
		}
	}

	return context, err
}

type contextProperty struct {
	name         string
	typ          types.Type
	readOptional bool
}

type contextSection struct {
	name       string
	properties []contextProperty
}

var contextSections = map[string]*contextSection{
	"app": {
		name: "app",
		properties: []contextProperty{
			{name: "name", typ: types.Text(), readOptional: true},
			{name: "version", typ: types.Text(), readOptional: true},
			{name: "build", typ: types.Text(), readOptional: true},
			{name: "namespace", typ: types.Text(), readOptional: true},
		},
	},
	"browser": {
		name: "browser",
		properties: []contextProperty{
			{name: "name", typ: types.Text(), readOptional: true},
			{name: "version", typ: types.Text(), readOptional: true},
		},
	},
	"campaign": {
		name: "campaign",
		properties: []contextProperty{
			{name: "name", typ: types.Text(), readOptional: true},
			{name: "source", typ: types.Text(), readOptional: true},
			{name: "medium", typ: types.Text(), readOptional: true},
			{name: "term", typ: types.Text(), readOptional: true},
			{name: "content", typ: types.Text(), readOptional: true},
		},
	},
	"device": {
		name: "device",
		properties: []contextProperty{
			{name: "id", typ: types.Text(), readOptional: true},
			{name: "advertisingId", typ: types.Text(), readOptional: true},
			{name: "adTrackingEnabled", typ: types.Boolean(), readOptional: true},
			{name: "manufacturer", typ: types.Text(), readOptional: true},
			{name: "model", typ: types.Text(), readOptional: true},
			{name: "name", typ: types.Text(), readOptional: true},
			{name: "type", typ: types.Text(), readOptional: true},
			{name: "token", typ: types.Text(), readOptional: true},
		},
	},
	"library": {
		name: "library",
		properties: []contextProperty{
			{name: "name", typ: types.Text(), readOptional: true},
			{name: "version", typ: types.Text(), readOptional: true},
		},
	},
	"location": {
		name: "location",
		properties: []contextProperty{
			{name: "city", typ: types.Text(), readOptional: true},
			{name: "country", typ: types.Text(), readOptional: true},
			{name: "latitude", typ: types.Float(64), readOptional: true},
			{name: "longitude", typ: types.Float(64), readOptional: true},
			{name: "speed", typ: types.Float(64), readOptional: true},
		},
	},
	"network": {
		name: "network",
		properties: []contextProperty{
			{name: "bluetooth", typ: types.Boolean(), readOptional: true},
			{name: "carrier", typ: types.Text(), readOptional: true},
			{name: "cellular", typ: types.Boolean(), readOptional: true},
			{name: "wifi", typ: types.Boolean(), readOptional: true},
		},
	},
	"os": {
		name: "os",
		properties: []contextProperty{
			{name: "name", typ: types.Text(), readOptional: true},
			{name: "version", typ: types.Text(), readOptional: true},
		},
	},
	"page": {
		name: "page",
		properties: []contextProperty{
			{name: "path", typ: types.Text(), readOptional: true},
			{name: "referrer", typ: types.Text(), readOptional: true},
			{name: "search", typ: types.Text(), readOptional: true},
			{name: "title", typ: types.Text(), readOptional: true},
			{name: "url", typ: types.Text(), readOptional: true},
		},
	},
	"referrer": {
		name: "referrer",
		properties: []contextProperty{
			{name: "id", typ: types.Text(), readOptional: true},
			{name: "type", typ: types.Text(), readOptional: true},
		},
	},
	"screen": {
		name: "screen",
		properties: []contextProperty{
			{name: "width", typ: types.Int(16), readOptional: true},
			{name: "height", typ: types.Int(16), readOptional: true},
			{name: "density", typ: types.Decimal(3, 2), readOptional: true},
		},
	},
	"session": {
		name: "session",
		properties: []contextProperty{
			{name: "id", typ: types.Int(64), readOptional: true},
			{name: "start", typ: types.Boolean(), readOptional: true},
		},
	},
}

// decodeContextSection decodes and returns a context section. The next token
// must be '{'.
func (d *decoder) decodeContextSection(section *contextSection, isDefault bool) (map[string]any, error) {

	_ = d.dec.SkipToken() // skip the first token.

	var err error
	var tok json.Token
	var sec map[string]any
	if !isDefault && d.context != nil {
		if v, ok := d.context[section.name].(map[string]any); ok {
			sec = maps.Clone(v)
		}
	}

	for {
		tok, _ = d.dec.ReadToken()
		if tok.Kind() == '}' {
			break
		}
		name := tok.String()
		var typ types.Type
		for _, property := range section.properties {
			if property.name == name {
				typ = property.typ
				break
			}
		}
		if !typ.Valid() {
			_ = d.dec.SkipValue()
			continue
		}
		tok, _ = d.dec.ReadToken()
		var v any
		switch typ.Kind() {
		case types.TextKind:
			if tok.Kind() != '"' {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a string", section.name, name)
			}
			s := tok.String()
			if s == "" {
				continue
			}
			if values := typ.Values(); values != nil && !slices.Contains(values, s) {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a valid value among the allowed options", section.name, name)
			}
			v = s
		case types.BooleanKind:
			switch tok.Kind() {
			case 'f':
			case 't':
				v = true
			default:
				return nil, errors.BadRequest("property 'context.%s.%s' is not a boolean", section.name, name)
			}
		case types.IntKind:
			if tok.Kind() != '0' {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a number", section.name, name)
			}
			v, err = tok.Int()
			if err != nil {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a valid %d-bit integer", section.name, name, typ.BitSize())
			}
			if v == 0 {
				continue
			}
		case types.FloatKind:
			if tok.Kind() != '0' {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a number", section.name, name)
			}
			v, err = tok.Float(typ.BitSize())
			if err != nil {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a valid %d-bit floating-point number",
					section.name, name, typ.BitSize())
			}
			if v == 0.0 {
				continue
			}
		case types.DecimalKind:
			if tok.Kind() != '0' {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a number", section.name, name)
			}
			f, err := tok.Float(64)
			if err != nil {
				return nil, errors.BadRequest("property 'context.%s.%s' is not a valid 64-bit floating-point number", section.name, name)
			}
			d, err := decimal.Float64(f, typ.Precision(), typ.Scale())
			if err != nil {
				return nil, errors.BadRequest("property 'context.%s.%s' exceeds the allowed precision of %d",
					section.name, name, typ.Precision())
			}
			if d.Sign() == 0 {
				continue
			}
			v = d
		default:
			panic("unexpected kind")
		}
		if sec == nil {
			sec = map[string]any{name: v}
		} else {
			sec[name] = v
		}
	}

	if isDefault || sec == nil {
		return sec, nil
	}
	if len(sec) == len(section.properties) {
		return sec, nil
	}
	for _, p := range section.properties {
		if p.readOptional {
			continue
		}
		if _, ok := sec[p.name]; ok {
			continue
		}
		var v any
		switch p.typ.Kind() {
		case types.TextKind:
			v = ""
		case types.BooleanKind:
			v = false
		case types.IntKind:
			v = 0
		case types.FloatKind:
			v = 0.0
		case types.DecimalKind:
			v = decimal.Decimal{}
		default:
			panic("unexpected kind")
		}
		sec[p.name] = v
	}

	return sec, nil
}

// errRead checks if the provided error is a *jsontext.SyntacticError. If it is,
// returns *errors.BadRequestError; otherwise, it returns errReadBody.
func errRead(err error) error {
	if _, ok := err.(*json.SyntaxError); ok {
		return errors.BadRequest("error parsing the request body as JSON: %s", err)
	}
	if err == io.EOF {
		return errors.BadRequest("request's body is empty")
	}
	if err == io.ErrUnexpectedEOF {
		return errors.BadRequest("error parsing the request body as JSON: it is not terminated")
	}
	return errReadBody
}

// makeEventID returns an event ID from its source and message ID.
func makeEventID(source int, messageId string) uuid.UUID {
	buf := [4]byte{}
	binary.BigEndian.PutUint32(buf[:], uint32(source))
	// The following code has been adapted from the uuid.NewHash function.
	h := sha1.New()
	h.Write(uuid.NameSpaceOID[:]) //nolint:errcheck
	h.Write([]byte(messageId))    //nolint:errcheck
	s := h.Sum(nil)
	var id uuid.UUID
	copy(id[:], s)
	id[6] = (id[6] & 0x0f) | uint8((5&0xf)<<4)
	id[8] = (id[8] & 0x3f) | 0x80 // RFC 4122 variant
	return id
}

// parseIP parses an IP address.
func parseIP(ip string) (net.IP, string, error) {
	addr := net.ParseIP(ip).To16()
	if addr == nil {
		return nil, "", errors.BadRequest("invalid IP")
	}
	return addr, addr.String(), nil
}

// parseUserAgent parses a user agent and returns context's browser and os.
//
// Note that in returned maps, if a value is empty or irrelevant, the
// corresponding key is omitted.
func parseUserAgent(userAgent string) (map[string]any, map[string]any) {

	// Parse the user agent.
	ua := uasurfer.Parse(userAgent)

	// Determine the browser.
	var name, other, version string
	switch ua.Browser.Name {
	default:
		name = "Other"
		ot := ua.Browser.Name.StringTrimPrefix()
		if n := utf8.RuneCountInString(ot); n <= 25 {
			other = ot
		}
	case uasurfer.BrowserChrome:
		name = "Chrome"
	case uasurfer.BrowserSafari:
		name = "Safari"
	case uasurfer.BrowserIE:
		name = "Edge"
	case uasurfer.BrowserFirefox:
		name = "Firefox"
	case uasurfer.BrowserSamsung:
		name = "Samsung Internet"
	case uasurfer.BrowserOpera:
		name = "Opera"
	}
	// If the browser has a name and its version has been parsed from the user
	// agent, then set the browser version as a tuple "major.minor.patch".
	if ver := ua.Browser.Version; (name != "Other" || other != "") && ver != (uasurfer.Version{}) {
		version = strconv.Itoa(ver.Major) +
			"." + strconv.Itoa(ver.Minor) +
			"." + strconv.Itoa(ver.Patch)
		// Discard versions longer than 25 characters.
		if len(version) > 25 {
			version = ""
		}
	}
	browser := map[string]any{
		"name": name,
	}
	if other != "" {
		browser["other"] = other
	}
	if version != "" {
		browser["version"] = version
	}

	// Determine the OS.
	switch ua.OS.Name {
	default:
		name = "Other"
		ot := ua.OS.Name.StringTrimPrefix()
		if n := utf8.RuneCountInString(ot); n <= 25 {
			other = ot
		}
	case uasurfer.OSMacOSX:
		name = "macOS"
	case uasurfer.OSAndroid:
		name = "Android"
	case uasurfer.OSWindows:
		name = "Windows"
	case uasurfer.OSiOS:
		name = "iOS"
	case uasurfer.OSLinux:
		name = "Linux"
	case uasurfer.OSChromeOS:
		name = "ChromeOS"
	}
	// If the OS version has been parsed from the user agent, then set the OS
	// version as a tuple "major.minor.patch".
	if ver := ua.OS.Version; ver != (uasurfer.Version{}) {
		version = strconv.Itoa(ver.Major) +
			"." + strconv.Itoa(ver.Minor) +
			"." + strconv.Itoa(ver.Patch)
		// Discard versions longer than 25 characters.
		if len(version) > 25 {
			version = ""
		}
	}
	os := map[string]any{
		"name": name,
	}
	if other != "" {
		os["other"] = other
	}
	if version != "" {
		os["version"] = version
	}
	return browser, os
}

// normalizeContextBrowser normalizes the content of 'context.browser' in an
// event, modifying the provided map.
//
// Specifically, it attempts to recognize the browser name and convert it into a
// normalized form. If the browser name is not among the recognized ones, the
// value for the 'name' key becomes 'Other' and an 'other' key is added with the
// given name; in all other cases, the 'other' key is removed.
//
// The 'version' field, if present, is preserved as is.
func normalizeContextBrowser(browser map[string]any) {
	name, ok := browser["name"].(string)
	if !ok {
		return // no browser name, so there's nothing to normalize.
	}
	var other string
	switch strings.ToLower(name) {
	case "chrome":
		name = "Chrome"
	case "safari":
		name = "Safari"
	case "edge":
		name = "Edge"
	case "firefox":
		name = "Firefox"
	case "samsung internet":
		name = "Samsung Internet"
	case "opera":
		name = "Opera"
	default:
		other = name
		name = "Other"
	}
	browser["name"] = name
	if other != "" {
		browser["other"] = other
	} else {
		delete(browser, "other")
	}
}

// normalizeContextOS normalizes the content of 'context.os' in an event,
// modifying the provided map.
//
// Specifically, it attempts to recognize the OS name and convert it into a
// normalized form. If the OS name is not among the recognized ones, the value
// for the 'name' key becomes 'Other' and an 'other' key is added with the given
// name; in all other cases, the 'other' key is removed.
//
// The 'version' field, if present, is preserved as is.
func normalizeContextOS(os map[string]any) {
	name, ok := os["name"].(string)
	if !ok {
		return // no os name, so there's nothing to normalize.
	}
	var other string
	switch strings.ToLower(name) {
	case "android":
		name = "Android"
	case "windows":
		name = "Windows"
	case "ios":
		name = "iOS"
	case "macos", "darwin":
		name = "macOS"
	case "linux":
		name = "Linux"
	case "chrome os":
		name = "Chrome OS"
	default:
		other = name
		name = "Other"
	}
	os["name"] = name
	if other != "" {
		os["other"] = other
	} else {
		delete(os, "other")
	}
}

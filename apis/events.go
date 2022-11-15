//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/pkg/open2b/sql"

	"github.com/mssola/user_agent"
	"github.com/open2b/nuts/culture"
	"github.com/oschwald/geoip2-golang"
	"golang.org/x/text/unicode/norm"
)

var errUnauthorized = errors.New("unauthorized")

const (
	dateTimeLayout    = "2006-01-02 15:04:05"
	flushQueueTimeout = 1 * time.Second // interval to flush the queue
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
}

// AddEvent adds an event.
func (apis *APIs) AddEvent(connection int, event *Event) {
	apis.eventsQueueMutex.Lock()
	apis.eventsQueue = append(apis.eventsQueue, event)
	var toFlush []*Event
	if len(apis.eventsQueue) == maxEventsQueueLen {
		toFlush = apis.eventsQueue
		apis.eventsQueue = nil
	}
	apis.eventsQueueMutex.Unlock()
	if toFlush != nil {
		go apis.flushEvents(toFlush)
	}
}

// startEventFlusher starts a goroutine that flushes the events queue every
// flushQueueTimeout seconds.
func (apis *APIs) startEventFlusher() {
	ticker := time.NewTicker(flushQueueTimeout)
	go func() {
		for {
			select {
			case <-ticker.C:
				apis.eventsQueueMutex.Lock()
				toFlush := apis.eventsQueue
				apis.eventsQueue = nil
				apis.eventsQueueMutex.Unlock()
				go apis.flushEvents(toFlush)
			}
		}
	}()
}

// flushEvents writes a batch of events to ClickHouse.
func (apis *APIs) flushEvents(events []*Event) {
	if len(events) == 0 {
		return
	}
	log.Printf("[info] flushing %d events", len(events))
RETRY:
	for {
		batch, err := apis.chDB.PrepareBatch(context.Background(), "INSERT INTO `events`\n"+
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

// serveEvents servers the "/apis/v1/events/" endpoint.
func (apis *APIs) serveEvents(w http.ResponseWriter, r *http.Request) error {

	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	now := time.Now().UTC()

	// Validate the content type.
	mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" || len(params) > 1 {
		return errBadRequest
	}
	if charset, ok := params["charset"]; ok && strings.ToLower(charset) != "utf-8" {
		return errBadRequest
	}

	// Authenticate the request.
	conn, key, ok := r.BasicAuth()
	if !ok || conn != "" && key != "" {
		return errUnauthorized
	}

	var connection int
	var typ ConnectorType
	var websiteHost string

	if key == "" {
		// Website request.
		connection, _ = strconv.Atoi(conn)
		if connection <= 0 {
			return errUnauthorized
		}
		err = apis.myDB.QueryRow("SELECT `websiteHost`\n"+
			"FROM `connections`\n"+
			"WHERE `id` = ? AND `type` = 'Website' AND `role` = 'Source'", connection).
			Scan(&websiteHost)
		typ = WebsiteType
	} else {
		// Mobile or server request.
		if !isWellFormedConnectorKey(key) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return nil
		}
		err = apis.myDB.QueryRow("SELECT `c`.`id`, CAST(`c`.`type` AS UNSIGNED)\n"+
			"FROM `connections_keys` AS `k`\n"+
			"INNER JOIN `connections` AS `c` ON `k`.`connection` = `c`.`id`\n"+
			"WHERE `c`.`type` IN ('Mobile', 'Server') AND `c`.`role` = 'Source' AND `k`.`key` = ?", key).
			Scan(&connection, &typ)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return nil
		}
		return err
	}

	// Decode the event.
	var event Event
	err = decodeEvent(r.Body, &event)
	if err != nil {
		return err
	}

	// Source.
	event.source = int32(connection)

	// Device.
	if _, err := base64.StdEncoding.DecodeString(event.Device); err != nil || len(event.Device) != 28 {
		return errBadRequest
	}

	// Event.
	switch typ {
	case MobileType:
		// TODO(marco)
	case ServerType, WebsiteType:
		if e := event.Event; e != "pageview" && e != "click" {
			return errBadRequest
		}
	}

	// Language.
	locale := culture.Locale(event.Language)
	if locale == nil {
		return errBadRequest
	}
	event.Language = locale.LanguageCode()

	// Referrer.
	if event.Referrer != "" {
		if typ == MobileType || utf8.RuneCountInString(event.Referrer) > 2048 {
			return errBadRequest
		} else if u, err := url.Parse(event.Referrer); err != nil || u.Scheme != "https" && u.Scheme != "http" {
			return errBadRequest
		}
	}

	// Target.
	if event.Target != "" {
		if typ == MobileType || utf8.RuneCountInString(event.Target) > 2048 {
			return errBadRequest
		} else if u, err := url.Parse(event.Target); err != nil || u.Scheme != "https" && u.Scheme != "http" {
			return errBadRequest
		}
	}

	// Get the user or create it if it does not exist.
	err = apis.myDB.QueryRow("SELECT `id` FROM `users` WHERE `source` = ? AND `device` = ?", connection, event.Device).Scan(&event.user)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == sql.ErrNoRows {
		err = apis.myDB.QueryRow("SELECT `user` FROM `devices` WHERE `source` = ? AND `id` = ?", connection, event.Device).Scan(&event.user)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			event.user, err = makeUserID()
			if err != nil {
				return err
			}
			_, err = apis.myDB.Exec("INSERT INTO `users` SET `source` = ?, `id` = ?, `device` = ?", connection, event.user, event.Device)
			if err != nil {
				return err
			}
		}
	}

	// Text.
	if utf8.RuneCountInString(event.Text) > 120 {
		event.Text = abbreviate(event.Text, 120)
	}

	// Title
	if utf8.RuneCountInString(event.Title) > 120 {
		event.Title = abbreviate(event.Title, 120)
	}

	// IP.
	var ip string
	switch typ {
	case MobileType, WebsiteType:
		if event.IP != "" {
			return errBadRequest
		}
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return errBadRequest
		}
	case ServerType:
		if event.IP == "" {
			return errBadRequest
		}
		ip = event.IP
	}
	requestIP := net.ParseIP(ip)
	if requestIP == nil {
		return errBadRequest
	}
	if ip := requestIP.String(); ip == "127.0.0.1" || ip == "::1" {
		requestIP = net.ParseIP(testGEOIP)
	}

	// URL.
	switch typ {
	case MobileType:
		if event.URL != "" {
			return errBadRequest
		}
	case WebsiteType:
		if event.URL == "" {
			return errBadRequest
		}
	}
	if event.URL != "" {
		if utf8.RuneCountInString(event.URL) > 2048 {
			return errBadRequest
		}
		u, err := url.Parse(event.URL)
		if err != nil {
			return errBadRequest
		}
		if u.Scheme != "https" && u.Scheme != "http" {
			return errBadRequest
		}
		if u.Host != websiteHost {
			return errBadRequest
		}
		if typ == WebsiteType {
			event.domain, _, _ = strings.Cut(websiteHost, ":")
		} else {
			event.domain, _, _ = strings.Cut(u.Host, ":")
		}
		event.path = strings.TrimLeft(strings.TrimRight(u.Path, "/"), "/")
		event.queryString = u.RawQuery
	}

	// Timestamp.
	if event.Timestamp == "" {
		event.Timestamp = now.Format(dateTimeLayout)
	} else {
		t, err := time.Parse(dateTimeLayout, event.Timestamp)
		if err != nil {
			return errBadRequest
		}
		switch typ {
		case MobileType, WebsiteType:
			if t.Add(-15*time.Minute).Before(now) || t.After(now) {
				event.Timestamp = now.Format(dateTimeLayout)
			}
		case ServerType:
			if t.After(now) {
				event.Timestamp = now.Format(dateTimeLayout)
			}
		}
	}
	event.date = event.Timestamp[0:10]

	// Country and city.
	if typ == MobileType || typ == WebsiteType {
		if event.Country != "" {
			return errBadRequest
		}
		if event.City != "" {
			return errBadRequest
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
		return errBadRequest
	} else if utf8.RuneCountInString(event.City) > 50 {
		return errBadRequest
	}

	// UserAgent, DeviceType.
	var userAgent string
	switch typ {
	case MobileType:
		if event.UserAgent != "" {
			return errBadRequest
		}
		if dt := event.DeviceType; dt != "" && !isDeviceType(dt) {
			return errBadRequest
		}
	case ServerType:
		userAgent = event.UserAgent
	case WebsiteType:
		userAgent = r.Header.Get("User-Agent")
	}
	if userAgent != "" {
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

	// Add the event.
	apis.AddEvent(connection, &event)

	return nil
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

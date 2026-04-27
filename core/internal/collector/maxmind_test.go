// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

// These tests verify Krenalis' MaxMind integration after updating
// github.com/oschwald/maxminddb-golang/v2. They are skipped unless
// KRENALIS_TEST_MAXMIND_DB_PATH points to a local City database with at least
// one US IPv4 record containing non-zero latitude and longitude, for example:
//
//	KRENALIS_TEST_MAXMIND_DB_PATH=GeoLite2-City.mmdb go test ./core/internal/collector -run TestDecoderMaxMindLocation -v
//
// Relative paths are resolved from the package test directory. The database
// file is not committed because of licensing restrictions.

import (
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/events"

	"github.com/oschwald/maxminddb-golang/v2"
)

type maxMindTestRecord struct {
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

// TestDecoderMaxMindLocation verifies that the decoder uses maxminddb-golang
// and a real MaxMind database to enrich events with location data.
func TestDecoderMaxMindLocation(t *testing.T) {
	dbPath, ok := os.LookupEnv("KRENALIS_TEST_MAXMIND_DB_PATH")
	if !ok || dbPath == "" {
		t.Log("skipping MaxMind integration test: KRENALIS_TEST_MAXMIND_DB_PATH is not set")
		t.SkipNow()
	}
	absDBPath, err := filepath.Abs(dbPath)
	if err != nil {
		t.Fatalf("cannot resolve absolute MaxMind database path for %q: %v", dbPath, err)
	}
	_, err = os.Stat(dbPath)
	if err != nil {
		t.Fatalf("MaxMind database not found at %q (absolute path %q): %v", dbPath, absDBPath, err)
	}

	db, err := maxminddb.Open(dbPath)
	if err != nil {
		t.Fatalf("maxminddb.Open(%q) failed (absolute path %q): %v", dbPath, absDBPath, err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("closing MaxMind DB: %v", err)
		}
	})

	ip, record := findMaxMindUSIPv4Record(t, db)
	wantLocation := maxMindLocation(record)
	if len(wantLocation) == 0 {
		t.Fatal("test record does not contain any location fields used by Krenalis")
	}
	if gotCountry := wantLocation["country"]; gotCountry != "US" {
		t.Fatalf("test record should resolve to country US, got %v", gotCountry)
	}
	t.Logf("using MaxMind test IP %s with location %#v", ip, wantLocation)

	t.Run("uses-context-ip", func(t *testing.T) {
		body := `{"type":"track","event":"click","anonymousId":"anon-1","context":{"ip":"` + ip.String() + `"}}`
		event := decodeMaxMindTestEvent(t, db, body, "198.51.100.23:9000", false)

		context := eventContext(t, event)
		gotLocation, ok := context["location"].(map[string]any)
		if !ok {
			t.Fatalf("expected context.location, got %T", context["location"])
		}
		if !reflect.DeepEqual(gotLocation, wantLocation) {
			t.Fatalf("context.location mismatch\nwant: %#v\n got: %#v", wantLocation, gotLocation)
		}
	})

	t.Run("uses-request-ip-when-fallback-is-enabled", func(t *testing.T) {
		body := `{"type":"track","event":"click","anonymousId":"anon-1"}`
		event := decodeMaxMindTestEvent(t, db, body, ip.String()+":9000", true)

		context := eventContext(t, event)
		gotLocation, ok := context["location"].(map[string]any)
		if !ok {
			t.Fatalf("expected context.location, got %T", context["location"])
		}
		if !reflect.DeepEqual(gotLocation, wantLocation) {
			t.Fatalf("context.location mismatch\nwant: %#v\n got: %#v", wantLocation, gotLocation)
		}
	})

	t.Run("does-not-overwrite-existing-location", func(t *testing.T) {
		body := `{"type":"track","event":"click","anonymousId":"anon-1","context":{"ip":"` + ip.String() + `","location":{"city":"Existing"}}}`
		event := decodeMaxMindTestEvent(t, db, body, "198.51.100.23:9000", false)

		context := eventContext(t, event)
		want := map[string]any{"city": "Existing"}
		gotLocation, ok := context["location"].(map[string]any)
		if !ok {
			t.Fatalf("expected context.location, got %T", context["location"])
		}
		if !reflect.DeepEqual(gotLocation, want) {
			t.Fatalf("context.location mismatch\nwant: %#v\n got: %#v", want, gotLocation)
		}
	})
}

// findMaxMindUSIPv4Record returns a US IPv4 address with coordinates.
func findMaxMindUSIPv4Record(t *testing.T, db *maxminddb.Reader) (netip.Addr, maxMindTestRecord) {
	t.Helper()

	for result := range db.Networks(maxminddb.SkipEmptyValues()) {
		var record maxMindTestRecord
		if err := result.Decode(&record); err != nil {
			t.Fatalf("decoding MaxMind network %s: %v", result.Prefix(), err)
		}
		if record.Country.IsoCode != "US" || record.Location.Latitude == 0 || record.Location.Longitude == 0 {
			continue
		}

		ip, ok := usableIPv4InPrefix(result.Prefix())
		if ok {
			return ip, record
		}
	}

	t.Fatal("MaxMind DB does not contain a US IPv4 record with non-zero latitude and longitude")
	return netip.Addr{}, maxMindTestRecord{}
}

// usableIPv4InPrefix returns a non-special IPv4 address contained in prefix.
func usableIPv4InPrefix(prefix netip.Prefix) (netip.Addr, bool) {
	if !prefix.Addr().Is4() {
		return netip.Addr{}, false
	}

	ip := prefix.Addr()
	for range 256 {
		if ip.Is4() && prefix.Contains(ip) && ip != ip0 && ip != ip16 && ip != ip24 && ip != ip32 && !ip.IsMulticast() {
			return ip, true
		}
		next := ip.Next()
		if !next.IsValid() || !prefix.Contains(next) {
			return netip.Addr{}, false
		}
		ip = next
	}
	return netip.Addr{}, false
}

// maxMindLocation applies Krenalis' MaxMind record-to-location rules.
func maxMindLocation(record maxMindTestRecord) map[string]any {
	location := map[string]any{}
	if city := record.City.Names.EN; city != "" {
		location["city"] = city
	}
	if lat := record.Location.Latitude; lat != 0 {
		location["latitude"] = lat
	}
	if long := record.Location.Longitude; long != 0 {
		location["longitude"] = long
	}
	if code, ok := countryCode(record.Country.IsoCode); ok {
		location["country"] = code
	}
	return location
}

// decodeMaxMindTestEvent decodes one event with a MaxMind reader attached.
func decodeMaxMindTestEvent(t *testing.T, db *maxminddb.Reader, body string, remoteAddr string, fallbackToRequestIP bool) events.Event {
	t.Helper()

	requestURL, err := url.Parse("/events/track")
	if err != nil {
		t.Fatalf("parsing request URL: %v", err)
	}
	request := &http.Request{
		Method: http.MethodPost,
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
			"User-Agent":   []string{"DecoderMaxMindLocationTest/1.0"},
		},
		RemoteAddr: remoteAddr,
		URL:        requestURL,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	dec, err := newDecoder(request)
	if err != nil {
		t.Fatalf("newDecoder: %v", err)
	}
	dec.SetMaxMindDB(db)

	var (
		event events.Event
		count int
	)
	for gotEvent, err := range dec.Events(42, fallbackToRequestIP) {
		if err != nil {
			t.Fatalf("decoding event: %v", err)
		}
		event = gotEvent
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
	if event == nil {
		t.Fatal("expected event, got nil")
	}
	return event
}

// eventContext returns the decoded event context.
func eventContext(t *testing.T, event events.Event) map[string]any {
	t.Helper()

	context, ok := event["context"].(map[string]any)
	if !ok {
		t.Fatalf("expected context object, got %T", event["context"])
	}
	return context
}

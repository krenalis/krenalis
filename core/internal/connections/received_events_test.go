// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/events"
	"github.com/krenalis/krenalis/tools/decimal"
)

// Test_ReceivedEvent checks that ReceivedEvent exposes all event fields
// correctly.
func Test_ReceivedEvent(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	event := events.Event{
		"anonymousId": "anon1",
		"channel":     "web",
		"category":    "marketing",
		"context": map[string]any{
			"app": map[string]any{
				"name":      "app",
				"version":   "1.0",
				"build":     "100",
				"namespace": "ns",
			},
			"browser": map[string]any{
				"name":    "Chrome",
				"other":   "",
				"version": "123",
			},
			"campaign": map[string]any{
				"name":    "cmp",
				"source":  "src",
				"medium":  "med",
				"term":    "term",
				"content": "cont",
			},
			"device": map[string]any{
				"id":                "dev1",
				"advertisingId":     "ad1",
				"adTrackingEnabled": true,
				"manufacturer":      "manu",
				"model":             "model",
				"name":              "dname",
				"type":              "mobile",
				"token":             "tok",
			},
			"ip": "192.0.2.1",
			"library": map[string]any{
				"name":    "lib",
				"version": "v1",
			},
			"locale": "en-US",
			"location": map[string]any{
				"city":      "Rome",
				"country":   "IT",
				"latitude":  12.3,
				"longitude": 45.6,
				"speed":     80.5,
			},
			"network": map[string]any{
				"bluetooth": true,
				"carrier":   "Vodafone",
				"cellular":  false,
				"wifi":      true,
			},
			"os": map[string]any{
				"name":    "Linux",
				"version": "5.0",
			},
			"page": map[string]any{
				"path":     "/home",
				"referrer": "https://example.com",
				"search":   "?a=1",
				"title":    "Home",
				"url":      "https://example.com/home",
			},
			"referrer": map[string]any{
				"id":   "ref1",
				"type": "link",
			},
			"screen": map[string]any{
				"width":   1080,
				"height":  1920,
				"density": decimal.MustInt(2),
			},
			"session": map[string]any{
				"id":    1751031467043,
				"start": true,
			},
			"timezone":  "Europe/Rome",
			"userAgent": "UA",
		},
		"event":             "Login",
		"groupId":           "group1",
		"messageId":         "ca333bd6-38d8-4de5-8d7f-f49f14752e8f",
		"name":              "LoginName",
		"receivedAt":        now.Add(time.Second),
		"sentAt":            now,
		"originalTimestamp": now,
		"timestamp":         now,
		"type":              "track",
		"previousId":        "user0",
		"userId":            "user1",
	}

	r := ReceivedEvent(event)

	if r.AnonymousID() != "anon1" {
		t.Fatalf("unexpected anonymousId %q", r.AnonymousID())
	}
	if channel, _ := r.Channel(); channel != "web" {
		t.Fatalf("unexpected channel %q", channel)
	}
	if category, _ := r.Category(); category != "marketing" {
		t.Fatalf("unexpected category %q", category)
	}
	if event, _ := r.Event(); event != "Login" {
		t.Fatalf("unexpected event %q", event)
	}
	if groupId, _ := r.GroupID(); groupId != "group1" {
		t.Fatalf("unexpected groupId %q", groupId)
	}
	if r.MessageID() != "ca333bd6-38d8-4de5-8d7f-f49f14752e8f" {
		t.Fatalf("unexpected messageId %q", r.MessageID())
	}
	if name, _ := r.Name(); name != "LoginName" {
		t.Fatalf("unexpected name %q", name)
	}
	if r.ReceivedAt() != now.Add(time.Second) {
		t.Fatalf("unexpected receivedAt %v", r.ReceivedAt())
	}
	if r.SentAt() != now {
		t.Fatalf("unexpected sentAt %v", r.SentAt())
	}
	if r.Timestamp() != now {
		t.Fatalf("unexpected timestamp %v", r.Timestamp())
	}
	if r.Type() != "track" {
		t.Fatalf("unexpected type %q", r.Type())
	}
	if previousId, _ := r.PreviousID(); previousId != "user0" {
		t.Fatalf("unexpected previousId %q", previousId)
	}
	if userId, _ := r.UserID(); userId != "user1" {
		t.Fatalf("unexpected userId %q", userId)
	}

	ctx, ok := r.Context()
	if !ok {
		t.Fatal("expected context")
	}
	appContext, ok := ctx.App()
	if !ok {
		t.Fatalf("unexpected app context")
	}
	if name, _ := appContext.Name(); name != "app" {
		t.Fatalf("unexpected app context")
	}
	if version, _ := appContext.Version(); version != "1.0" {
		t.Fatalf("unexpected app context")
	}
	if build, _ := appContext.Build(); build != "100" {
		t.Fatalf("unexpected app context")
	}
	if ns, _ := appContext.Namespace(); ns != "ns" {
		t.Fatalf("unexpected app context")
	}

	browser, ok := ctx.Browser()
	if !ok {
		t.Fatalf("unexpected browser context")
	}
	if name, _ := browser.Name(); name != "Chrome" {
		t.Fatalf("unexpected browser context")
	}
	if other, _ := browser.Other(); other != "" {
		t.Fatalf("unexpected browser context")
	}
	if version, _ := browser.Version(); version != "123" {
		t.Fatalf("unexpected browser context")
	}

	campaign, ok := ctx.Campaign()
	if !ok {
		t.Fatalf("unexpected campaign context")
	}
	if name, _ := campaign.Name(); name != "cmp" {
		t.Fatalf("unexpected campaign context")
	}
	if source, _ := campaign.Source(); source != "src" {
		t.Fatalf("unexpected campaign context")
	}
	if medium, _ := campaign.Medium(); medium != "med" {
		t.Fatalf("unexpected campaign context")
	}
	if term, _ := campaign.Term(); term != "term" {
		t.Fatalf("unexpected campaign context")
	}
	if content, _ := campaign.Content(); content != "cont" {
		t.Fatalf("unexpected campaign context")
	}

	device, ok := ctx.Device()
	if !ok {
		t.Fatalf("unexpected device context")
	}
	if id, _ := device.ID(); id != "dev1" {
		t.Fatalf("unexpected device context")
	}
	if advId, _ := device.AdvertisingID(); advId != "ad1" {
		t.Fatalf("unexpected device context")
	}
	if enabled, _ := device.AdTrackingEnabled(); !enabled {
		t.Fatalf("unexpected device context")
	}
	if manu, _ := device.Manufacturer(); manu != "manu" {
		t.Fatalf("unexpected device context")
	}
	if model, _ := device.Model(); model != "model" {
		t.Fatalf("unexpected device context")
	}
	if name, _ := device.Name(); name != "dname" {
		t.Fatalf("unexpected device context")
	}
	if typ, _ := device.Type(); typ != "mobile" {
		t.Fatalf("unexpected device context")
	}
	if tok, _ := device.Token(); tok != "tok" {
		t.Fatalf("unexpected device context")
	}

	if ip, _ := ctx.IP(); ip != "192.0.2.1" {
		t.Fatalf("unexpected IP %q", ip)
	}

	library, ok := ctx.Library()
	if !ok {
		t.Fatalf("unexpected library context")
	}
	if name, _ := library.Name(); name != "lib" {
		t.Fatalf("unexpected library context")
	}
	if version, _ := library.Version(); version != "v1" {
		t.Fatalf("unexpected library context")
	}

	if locale, _ := ctx.Locale(); locale != "en-US" {
		t.Fatalf("unexpected locale %q", locale)
	}

	location, ok := ctx.Location()
	if !ok {
		t.Fatalf("unexpected location context")
	}
	if city, _ := location.City(); city != "Rome" {
		t.Fatalf("unexpected location context")
	}
	if country, _ := location.Country(); country != "IT" {
		t.Fatalf("unexpected location context")
	}
	if lat, _ := location.Latitude(); lat != 12.3 {
		t.Fatalf("unexpected location context")
	}
	if lon, _ := location.Longitude(); lon != 45.6 {
		t.Fatalf("unexpected location context")
	}
	if speed, _ := location.Speed(); speed != 80.5 {
		t.Fatalf("unexpected location context")
	}

	network, ok := ctx.Network()
	if !ok {
		t.Fatalf("unexpected network context")
	}
	if bt, _ := network.Bluetooth(); !bt {
		t.Fatalf("unexpected network context")
	}
	if carrier, _ := network.Carrier(); carrier != "Vodafone" {
		t.Fatalf("unexpected network context")
	}
	if cellular, _ := network.Cellular(); cellular {
		t.Fatalf("unexpected network context")
	}
	if wifi, _ := network.WiFi(); !wifi {
		t.Fatalf("unexpected network context")
	}

	os, ok := ctx.OS()
	if !ok {
		t.Fatalf("unexpected OS context")
	}
	if name, _ := os.Name(); name != "Linux" {
		t.Fatalf("unexpected os context")
	}
	if version, _ := os.Version(); version != "5.0" {
		t.Fatalf("unexpected os context")
	}

	page, ok := ctx.Page()
	if !ok {
		t.Fatalf("unexpected page context")
	}
	if path, _ := page.Path(); path != "/home" {
		t.Fatalf("unexpected page context")
	}
	if referrer, _ := page.Referrer(); referrer != "https://example.com" {
		t.Fatalf("unexpected page context")
	}
	if search, _ := page.Search(); search != "?a=1" {
		t.Fatalf("unexpected page context")
	}
	if title, _ := page.Title(); title != "Home" {
		t.Fatalf("unexpected page context")
	}
	if url, _ := page.URL(); url != "https://example.com/home" {
		t.Fatalf("unexpected page context")
	}

	referrer, ok := ctx.Referrer()
	if !ok {
		t.Fatalf("unexpected referrer context")
	}
	if id, _ := referrer.ID(); id != "ref1" {
		t.Fatalf("unexpected referrer context")
	}
	if typ, _ := referrer.Type(); typ != "link" {
		t.Fatalf("unexpected referrer context")
	}

	if v, ok := ctx.Screen(); !ok {
		t.Fatalf("unexpected screen context")
	} else {
		w, _ := v.Width()
		h, _ := v.Height()
		d, _ := v.Density()
		if w != 1080 || h != 1920 || d.Cmp(decimal.MustUint(2)) != 0 {
			t.Fatalf("unexpected screen context")
		}
	}
	session, ok := ctx.Session()
	if !ok {
		t.Fatalf("unexpected session context")
	}
	if id, _ := session.ID(); id != 1751031467043 {
		t.Fatalf("unexpected session context")
	}
	if start, _ := session.Start(); !start {
		t.Fatalf("unexpected session context")
	}
	if tz, _ := ctx.Timezone(); tz != "Europe/Rome" {
		t.Fatalf("unexpected timezone %q", tz)
	}
	if ua, _ := ctx.UserAgent(); ua != "UA" {
		t.Fatalf("unexpected user agent %q", ua)
	}
}

// Test_ReceivedEventMissingFields verifies default return values when fields
// are absent.
func Test_ReceivedEventMissingFields(t *testing.T) {
	now := time.Now()
	event := events.Event{
		"anonymousId":       "a",
		"context":           map[string]any{"ip": "1.2.3.4"},
		"messageId":         "m",
		"receivedAt":        now,
		"sentAt":            now,
		"originalTimestamp": now,
		"timestamp":         now,
		"type":              "page",
		"userId":            nil,
	}

	r := ReceivedEvent(event)

	if _, ok := r.Channel(); ok {
		t.Fatal("expected no channel")
	}

	if _, ok := r.Category(); ok {
		t.Fatal("expected no category")
	}

	if _, ok := r.Event(); ok {
		t.Fatal("expected no 'event'")
	}

	if _, ok := r.GroupID(); ok {
		t.Fatal("expected no groupId")
	}

	if _, ok := r.Name(); ok {
		t.Fatalf("expected no name")
	}

	if _, ok := r.UserID(); ok {
		t.Fatal("expected no userId")
	}

	ctx, ok := r.Context()
	if !ok {
		t.Fatal("expected content")
	}

	if _, ok := ctx.App(); ok {
		t.Fatalf("expected no app context")
	}
	if ip, _ := ctx.IP(); ip != "1.2.3.4" {
		t.Fatalf("unexpected IP %q", ip)
	}
	if _, ok := ctx.Session(); ok {
		t.Fatalf("expected no session context")
	}
	if _, ok := ctx.Locale(); ok {
		t.Fatal("expected no locale")
	}
	if _, ok := ctx.Timezone(); ok {
		t.Fatal("expected no timezone")
	}
	if _, ok := ctx.UserAgent(); ok {
		t.Fatal("expected no user agent")
	}
}

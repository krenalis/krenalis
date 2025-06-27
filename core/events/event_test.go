//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package events

import (
	"testing"
	"time"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// Test_RawEvent checks that RawEvent exposes all event fields correctly.
func Test_RawEvent(t *testing.T) {
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	event := Event{
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
		"userId":            "user1",
	}

	r := RawEvent(event)

	if r.AnonymousId() != "anon1" {
		t.Fatalf("unexpected AnonymousId %q", r.AnonymousId())
	}
	if r.Channel() != "web" {
		t.Fatalf("unexpected Channel %q", r.Channel())
	}
	if r.Category() != "marketing" {
		t.Fatalf("unexpected Category %q", r.Category())
	}
	if r.Event() != "Login" {
		t.Fatalf("unexpected Event %q", r.Event())
	}
	if r.GroupId() != "group1" {
		t.Fatalf("unexpected GroupId %q", r.GroupId())
	}
	if r.MessageId() != "ca333bd6-38d8-4de5-8d7f-f49f14752e8f" {
		t.Fatalf("unexpected MessageId %q", r.MessageId())
	}
	if r.Name() != "LoginName" {
		t.Fatalf("unexpected Name %q", r.Name())
	}
	if r.ReceivedAt() != now.Add(time.Second) {
		t.Fatalf("unexpected ReceivedAt %v", r.ReceivedAt())
	}
	if r.SentAt() != now {
		t.Fatalf("unexpected SentAt %v", r.SentAt())
	}
	if r.Timestamp() != now {
		t.Fatalf("unexpected Timestamp %v", r.Timestamp())
	}
	if r.Type() != "track" {
		t.Fatalf("unexpected Type %q", r.Type())
	}
	if r.UserId() != "user1" {
		t.Fatalf("unexpected UserId %q", r.UserId())
	}

	ctx := r.Context()
	if v, ok := ctx.App(); !ok || v.Name() != "app" || v.Version() != "1.0" || v.Build() != "100" || v.Namespace() != "ns" {
		t.Fatalf("unexpected app context")
	}
	if v, ok := ctx.Browser(); !ok || v.Name() != "Chrome" || v.Other() != "" || v.Version() != "123" {
		t.Fatalf("unexpected browser context")
	}
	if v, ok := ctx.Campaign(); !ok || v.Name() != "cmp" || v.Source() != "src" || v.Medium() != "med" || v.Term() != "term" || v.Content() != "cont" {
		t.Fatalf("unexpected campaign context")
	}
	if v, ok := ctx.Device(); !ok || v.Id() != "dev1" || v.AdvertisingId() != "ad1" || !v.AdTrackingEnabled() || v.Manufacturer() != "manu" || v.Model() != "model" || v.Name() != "dname" || v.Type() != "mobile" || v.Token() != "tok" {
		t.Fatalf("unexpected device context")
	}
	if ctx.IP() != "192.0.2.1" {
		t.Fatalf("unexpected IP %q", ctx.IP())
	}
	if v, ok := ctx.Library(); !ok || v.Name() != "lib" || v.Version() != "v1" {
		t.Fatalf("unexpected library context")
	}
	if ctx.Locale() != "en-US" {
		t.Fatalf("unexpected locale %q", ctx.Locale())
	}
	if v, ok := ctx.Location(); !ok || v.City() != "Rome" || v.Country() != "IT" || v.Latitude() != 12.3 || v.Longitude() != 45.6 || v.Speed() != 80.5 {
		t.Fatalf("unexpected location context")
	}
	if v, ok := ctx.Network(); !ok || !v.Bluetooth() || v.Carrier() != "Vodafone" || v.Cellular() || !v.WiFi() {
		t.Fatalf("unexpected network context")
	}
	if v, ok := ctx.OS(); !ok || v.Name() != "Linux" || v.Version() != "5.0" {
		t.Fatalf("unexpected OS context")
	}
	if v, ok := ctx.Page(); !ok || v.Path() != "/home" || v.Referrer() != "https://example.com" || v.Search() != "?a=1" || v.Title() != "Home" || v.URL() != "https://example.com/home" {
		t.Fatalf("unexpected page context")
	}
	if v, ok := ctx.Referrer(); !ok || v.Id() != "ref1" || v.Type() != "link" {
		t.Fatalf("unexpected referrer context")
	}
	if v, ok := ctx.Screen(); !ok || v.Width() != 1080 || v.Height() != 1920 || v.Density().Cmp(decimal.MustUint(2)) != 0 {
		t.Fatalf("unexpected screen context")
	}
	if v, ok := ctx.Session(); !ok || v.Id() != 1751031467043 || !v.Start() {
		t.Fatalf("unexpected session context")
	}
	if ctx.Timezone() != "Europe/Rome" {
		t.Fatalf("unexpected timezone %q", ctx.Timezone())
	}
	if ctx.UserAgent() != "UA" {
		t.Fatalf("unexpected user agent %q", ctx.UserAgent())
	}
}

// Test_RawEventMissingFields verifies default return values when fields are
// absent.
func Test_RawEventMissingFields(t *testing.T) {
	now := time.Now()
	event := Event{
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

	r := RawEvent(event)

	if r.Channel() != "" || r.Category() != "" || r.Event() != "" || r.GroupId() != "" || r.Name() != "" || r.UserId() != "" {
		t.Fatalf("unexpected values for optional fields")
	}

	ctx := r.Context()
	if _, ok := ctx.App(); ok {
		t.Fatalf("expected no app context")
	}
	if ctx.IP() != "1.2.3.4" {
		t.Fatalf("unexpected IP %q", ctx.IP())
	}
	if _, ok := ctx.Session(); ok {
		t.Fatalf("expected no session context")
	}
	if ctx.Locale() != "" || ctx.Timezone() != "" || ctx.UserAgent() != "" {
		t.Fatalf("unexpected non-empty context values")
	}
}

func Test_Schema(t *testing.T) {
	if n := types.NumProperties(Schema); n != 19 {
		t.Fatalf("expected 18 properties, got %d", n)
	}
}

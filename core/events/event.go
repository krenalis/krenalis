//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// Event represents an event.
type Event map[string]any

// Schema is the event schema.
var Schema = types.Object([]types.Property{
	{Name: "id", Type: types.UUID()},
	{Name: "user", Type: types.UUID(), ReadOptional: true},
	{Name: "connection", Type: types.Int(32)},
	{Name: "anonymousId", Type: types.Text()},
	{Name: "channel", Type: types.Text(), ReadOptional: true},
	{Name: "category", Type: types.Text(), ReadOptional: true},
	{
		Name: "context",
		Type: types.Object([]types.Property{
			{
				Name: "app",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text()},
					{Name: "version", Type: types.Text()},
					{Name: "build", Type: types.Text()},
					{Name: "namespace", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{
				Name: "browser",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("None", "Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other")},
					{Name: "other", Type: types.Text()},
					{Name: "version", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{
				Name: "campaign",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text()},
					{Name: "source", Type: types.Text()},
					{Name: "medium", Type: types.Text()},
					{Name: "term", Type: types.Text()},
					{Name: "content", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{
				Name: "device",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Text()},
					{Name: "advertisingId", Type: types.Text()},
					{Name: "adTrackingEnabled", Type: types.Boolean()},
					{Name: "manufacturer", Type: types.Text()},
					{Name: "model", Type: types.Text()},
					{Name: "name", Type: types.Text()},
					{Name: "type", Type: types.Text()},
					{Name: "token", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{Name: "ip", Type: types.Inet(), ReadOptional: true},
			{
				Name: "library",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text()},
					{Name: "version", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{Name: "locale", Type: types.Text(), ReadOptional: true},
			{
				Name: "location",
				Type: types.Object([]types.Property{
					{Name: "city", Type: types.Text()},
					{Name: "country", Type: types.Text()},
					{Name: "latitude", Type: types.Float(64)},
					{Name: "longitude", Type: types.Float(64)},
					{Name: "speed", Type: types.Float(64)},
				}),
				ReadOptional: true,
			},
			{
				Name: "network",
				Type: types.Object([]types.Property{
					{Name: "bluetooth", Type: types.Boolean()},
					{Name: "carrier", Type: types.Text()},
					{Name: "cellular", Type: types.Boolean()},
					{Name: "wifi", Type: types.Boolean()},
				}),
				ReadOptional: true,
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("None", "Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other")},
					{Name: "version", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{
				Name: "page",
				Type: types.Object([]types.Property{
					{Name: "path", Type: types.Text()},
					{Name: "referrer", Type: types.Text()},
					{Name: "search", Type: types.Text()},
					{Name: "title", Type: types.Text()},
					{Name: "url", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{
				Name: "referrer",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Text()},
					{Name: "type", Type: types.Text()},
				}),
				ReadOptional: true,
			},
			{
				Name: "screen",
				Type: types.Object([]types.Property{
					{Name: "width", Type: types.Int(16)},
					{Name: "height", Type: types.Int(16)},
					{Name: "density", Type: types.Decimal(3, 2)},
				}),
				ReadOptional: true,
			},
			{
				Name: "session",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Int(64)},
					{Name: "start", Type: types.Boolean(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{Name: "timezone", Type: types.Text(), ReadOptional: true},
			{Name: "userAgent", Type: types.Text(), ReadOptional: true},
		}),
	},
	{Name: "event", Type: types.Text(), ReadOptional: true},
	{Name: "groupId", Type: types.Text(), ReadOptional: true},
	{Name: "messageId", Type: types.Text()},
	{Name: "name", Type: types.Text(), ReadOptional: true},
	{Name: "properties", Type: types.JSON(), ReadOptional: true},
	{Name: "receivedAt", Type: types.DateTime()},
	{Name: "sentAt", Type: types.DateTime()},
	{Name: "originalTimestamp", Type: types.DateTime()},
	{Name: "timestamp", Type: types.DateTime()},
	{Name: "traits", Type: types.JSON(), ReadOptional: true},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{Name: "userId", Type: types.Text(), Nullable: true},
})

// rawEvent implements the meergo.RawEvent interface. A RawEvent is passed to
// the SendEvents method of an app connector.
type rawEvent struct {
	event Event
}

// RawEvent wraps an Event and returns a value that implements the
// meergo.RawEvent interface.
//
// The provided event must conform to the event schema (Schema), otherwise
// calling methods on the returned value may cause a panic.
func RawEvent(event Event) meergo.RawEvent {
	return rawEvent{event}
}

func (e rawEvent) AnonymousId() string {
	return e.event["anonymousId"].(string)
}

func (e rawEvent) Channel() string {
	if channel, ok := e.event["channel"]; ok {
		return channel.(string)
	}
	return ""
}

func (e rawEvent) Category() string {
	if category, ok := e.event["category"]; ok {
		return category.(string)
	}
	return ""
}

func (e rawEvent) Context() meergo.RawEventContext {
	return rawEventContext{e.event["context"].(map[string]any)}
}

func (e rawEvent) Event() string {
	if event, ok := e.event["event"]; ok {
		return event.(string)
	}
	return ""
}

func (e rawEvent) GroupId() string {
	if groupId, ok := e.event["groupId"]; ok {
		return groupId.(string)
	}
	return ""
}

func (e rawEvent) MessageId() string {
	return e.event["messageId"].(string)
}

func (e rawEvent) Name() string {
	if name, ok := e.event["name"]; ok {
		return name.(string)
	}
	return ""
}

func (e rawEvent) ReceivedAt() time.Time {
	return e.event["receivedAt"].(time.Time)
}

func (e rawEvent) SentAt() time.Time {
	return e.event["sentAt"].(time.Time)
}

func (e rawEvent) Timestamp() time.Time {
	return e.event["timestamp"].(time.Time)
}

func (e rawEvent) Type() string {
	return e.event["type"].(string)
}

func (e rawEvent) UserId() string {
	if userId := e.event["userId"]; userId != nil {
		return userId.(string)
	}
	return ""
}

type rawEventContext struct {
	context map[string]any
}

func (c rawEventContext) App() (meergo.RawEventContextApp, bool) {
	if app, ok := c.context["app"].(map[string]any); ok {
		return rawEventContextApp{app}, true
	}
	return nil, false
}

func (c rawEventContext) Browser() (meergo.RawEventContextBrowser, bool) {
	if browser, ok := c.context["browser"].(map[string]any); ok {
		return rawEventContextBrowser{browser}, true
	}
	return nil, false
}

func (c rawEventContext) Campaign() (meergo.RawEventContextCampaign, bool) {
	if campaign, ok := c.context["campaign"].(map[string]any); ok {
		return rawEventContextCampaign{campaign}, true
	}
	return nil, false
}

func (c rawEventContext) Device() (meergo.RawEventContextDevice, bool) {
	if campaign, ok := c.context["device"].(map[string]any); ok {
		return rawEventContextDevice{campaign}, true
	}
	return nil, false
}

func (c rawEventContext) IP() string {
	if ip, ok := c.context["ip"]; ok {
		return ip.(string)
	}
	return ""
}

func (c rawEventContext) Library() (meergo.RawEventContextLibrary, bool) {
	if library, ok := c.context["library"].(map[string]any); ok {
		return rawEventContextLibrary{library}, true
	}
	return nil, false
}

func (c rawEventContext) Locale() string {
	if locale, ok := c.context["locale"]; ok {
		return locale.(string)
	}
	return ""
}

func (c rawEventContext) Location() (meergo.RawEventContextLocation, bool) {
	if location, ok := c.context["location"].(map[string]any); ok {
		return rawEventContextLocation{location}, true
	}
	return nil, false
}

func (c rawEventContext) Network() (meergo.RawEventContextNetwork, bool) {
	if network, ok := c.context["network"].(map[string]any); ok {
		return rawEventContextNetwork{network}, true
	}
	return nil, false
}

func (c rawEventContext) OS() (meergo.RawEventContextOS, bool) {
	if os, ok := c.context["os"].(map[string]any); ok {
		return rawEventContextOS{os}, true
	}
	return nil, false
}

func (c rawEventContext) Page() (meergo.RawEventContextPage, bool) {
	if page, ok := c.context["page"].(map[string]any); ok {
		return rawEventContextPage{page}, true
	}
	return nil, false
}

func (c rawEventContext) Referrer() (meergo.RawEventContextReferrer, bool) {
	if referrer, ok := c.context["referrer"].(map[string]any); ok {
		return rawEventContextReferrer{referrer}, true
	}
	return nil, false
}

func (c rawEventContext) Screen() (meergo.RawEventContextScreen, bool) {
	if screen, ok := c.context["screen"].(map[string]any); ok {
		return rawEventContextScreen{screen}, true
	}
	return nil, false
}

func (c rawEventContext) Session() (meergo.RawEventContextSession, bool) {
	if session, ok := c.context["session"].(map[string]any); ok {
		return rawEventContextSession{session}, true
	}
	return nil, false
}

func (c rawEventContext) Timezone() string {
	if timezone, ok := c.context["timezone"]; ok {
		return timezone.(string)
	}
	return ""
}

func (c rawEventContext) UserAgent() string {
	if userAgent, ok := c.context["userAgent"]; ok {
		return userAgent.(string)
	}
	return ""
}

type rawEventContextApp struct {
	app map[string]any
}

func (c rawEventContextApp) Name() string {
	return c.app["name"].(string)
}

func (c rawEventContextApp) Version() string {
	return c.app["version"].(string)
}

func (c rawEventContextApp) Build() string {
	return c.app["build"].(string)
}

func (c rawEventContextApp) Namespace() string {
	return c.app["namespace"].(string)
}

type rawEventContextBrowser struct {
	browser map[string]any
}

func (c rawEventContextBrowser) Name() string {
	return c.browser["name"].(string)
}

func (c rawEventContextBrowser) Other() string {
	return c.browser["other"].(string)
}

func (c rawEventContextBrowser) Version() string {
	return c.browser["version"].(string)
}

type rawEventContextCampaign struct {
	campaign map[string]any
}

func (c rawEventContextCampaign) Name() string {
	return c.campaign["name"].(string)
}

func (c rawEventContextCampaign) Source() string {
	return c.campaign["source"].(string)
}

func (c rawEventContextCampaign) Medium() string {
	return c.campaign["medium"].(string)
}

func (c rawEventContextCampaign) Term() string {
	return c.campaign["term"].(string)
}

func (c rawEventContextCampaign) Content() string {
	return c.campaign["content"].(string)
}

type rawEventContextDevice struct {
	device map[string]any
}

func (c rawEventContextDevice) Id() string {
	return c.device["id"].(string)
}

func (c rawEventContextDevice) AdvertisingId() string {
	return c.device["advertisingId"].(string)
}

func (c rawEventContextDevice) AdTrackingEnabled() bool {
	return c.device["adTrackingEnabled"].(bool)
}

func (c rawEventContextDevice) Manufacturer() string {
	return c.device["manufacturer"].(string)
}

func (c rawEventContextDevice) Model() string {
	return c.device["model"].(string)
}

func (c rawEventContextDevice) Name() string {
	return c.device["name"].(string)
}

func (c rawEventContextDevice) Type() string {
	return c.device["type"].(string)
}

func (c rawEventContextDevice) Token() string {
	return c.device["token"].(string)
}

type rawEventContextLibrary struct {
	library map[string]any
}

func (c rawEventContextLibrary) Name() string {
	return c.library["id"].(string)
}

func (c rawEventContextLibrary) Version() string {
	return c.library["version"].(string)
}

type rawEventContextLocation struct {
	location map[string]any
}

func (c rawEventContextLocation) City() string {
	return c.location["city"].(string)
}

func (c rawEventContextLocation) Country() string {
	return c.location["country"].(string)
}

func (c rawEventContextLocation) Latitude() float64 {
	return c.location["latitude"].(float64)
}

func (c rawEventContextLocation) Longitude() float64 {
	return c.location["longitude"].(float64)
}

func (c rawEventContextLocation) Speed() float64 {
	return c.location["speed"].(float64)
}

type rawEventContextNetwork struct {
	network map[string]any
}

func (c rawEventContextNetwork) Bluetooth() bool {
	return c.network["bluetooth"].(bool)
}

func (c rawEventContextNetwork) Carrier() string {
	return c.network["carrier"].(string)
}

func (c rawEventContextNetwork) Cellular() bool {
	return c.network["cellular"].(bool)
}

func (c rawEventContextNetwork) WiFi() bool {
	return c.network["wifi"].(bool)
}

type rawEventContextOS struct {
	os map[string]any
}

func (c rawEventContextOS) Name() string {
	return c.os["name"].(string)
}

func (c rawEventContextOS) Version() string {
	return c.os["version"].(string)
}

type rawEventContextPage struct {
	page map[string]any
}

func (c rawEventContextPage) Path() string {
	return c.page["path"].(string)
}

func (c rawEventContextPage) Referrer() string {
	return c.page["referrer"].(string)
}

func (c rawEventContextPage) Search() string {
	return c.page["search"].(string)
}

func (c rawEventContextPage) Title() string {
	return c.page["title"].(string)
}

func (c rawEventContextPage) URL() string {
	return c.page["url"].(string)
}

type rawEventContextReferrer struct {
	referrer map[string]any
}

func (c rawEventContextReferrer) Id() string {
	return c.referrer["id"].(string)
}

func (c rawEventContextReferrer) Type() string {
	return c.referrer["type"].(string)
}

type rawEventContextScreen struct {
	screen map[string]any
}

func (c rawEventContextScreen) Width() int {
	return c.screen["width"].(int)
}

func (c rawEventContextScreen) Height() int {
	return c.screen["height"].(int)
}

func (c rawEventContextScreen) Density() decimal.Decimal {
	return c.screen["density"].(decimal.Decimal)
}

type rawEventContextSession struct {
	session map[string]any
}

func (c rawEventContextSession) Id() int {
	return c.session["id"].(int)
}

func (c rawEventContextSession) Start() bool {
	start, _ := c.session["start"].(bool)
	return start
}

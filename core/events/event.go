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
	// The "user" field may be set only by the Identity Resolution for events
	// stored in the data warehouse.
	// For all other events, it never has a value.
	// For consistency, it is included in all event schemas to avoid having to
	// differentiate between schemas.
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
					{Name: "name", Type: types.Text(), ReadOptional: true},
					{Name: "version", Type: types.Text(), ReadOptional: true},
					{Name: "build", Type: types.Text(), ReadOptional: true},
					{Name: "namespace", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "browser",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("Chrome", "Safari", "Edge", "Firefox", "Samsung Internet", "Opera", "Other"), ReadOptional: true},
					{Name: "other", Type: types.Text(), ReadOptional: true},
					{Name: "version", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "campaign",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text(), ReadOptional: true},
					{Name: "source", Type: types.Text(), ReadOptional: true},
					{Name: "medium", Type: types.Text(), ReadOptional: true},
					{Name: "term", Type: types.Text(), ReadOptional: true},
					{Name: "content", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "device",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Text(), ReadOptional: true},
					{Name: "advertisingId", Type: types.Text(), ReadOptional: true},
					{Name: "adTrackingEnabled", Type: types.Boolean(), ReadOptional: true},
					{Name: "manufacturer", Type: types.Text(), ReadOptional: true},
					{Name: "model", Type: types.Text(), ReadOptional: true},
					{Name: "name", Type: types.Text(), ReadOptional: true},
					{Name: "type", Type: types.Text(), ReadOptional: true},
					{Name: "token", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{Name: "ip", Type: types.Inet(), ReadOptional: true},
			{
				Name: "library",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text(), ReadOptional: true},
					{Name: "version", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{Name: "locale", Type: types.Text(), ReadOptional: true},
			{
				Name: "location",
				Type: types.Object([]types.Property{
					{Name: "city", Type: types.Text(), ReadOptional: true},
					{Name: "country", Type: types.Text(), ReadOptional: true},
					{Name: "latitude", Type: types.Float(64), ReadOptional: true},
					{Name: "longitude", Type: types.Float(64), ReadOptional: true},
					{Name: "speed", Type: types.Float(64), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "network",
				Type: types.Object([]types.Property{
					{Name: "bluetooth", Type: types.Boolean(), ReadOptional: true},
					{Name: "carrier", Type: types.Text(), ReadOptional: true},
					{Name: "cellular", Type: types.Boolean(), ReadOptional: true},
					{Name: "wifi", Type: types.Boolean(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "os",
				Type: types.Object([]types.Property{
					{Name: "name", Type: types.Text().WithValues("Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"), ReadOptional: true},
					{Name: "other", Type: types.Text(), ReadOptional: true},
					{Name: "version", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "page",
				Type: types.Object([]types.Property{
					{Name: "path", Type: types.Text(), ReadOptional: true},
					{Name: "referrer", Type: types.Text(), ReadOptional: true},
					{Name: "search", Type: types.Text(), ReadOptional: true},
					{Name: "title", Type: types.Text(), ReadOptional: true},
					{Name: "url", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "referrer",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Text(), ReadOptional: true},
					{Name: "type", Type: types.Text(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "screen",
				Type: types.Object([]types.Property{
					{Name: "width", Type: types.Int(16), ReadOptional: true},
					{Name: "height", Type: types.Int(16), ReadOptional: true},
					{Name: "density", Type: types.Decimal(3, 2), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{
				Name: "session",
				Type: types.Object([]types.Property{
					{Name: "id", Type: types.Int(64), ReadOptional: true},
					{Name: "start", Type: types.Boolean(), ReadOptional: true},
				}),
				ReadOptional: true,
			},
			{Name: "timezone", Type: types.Text(), ReadOptional: true},
			{Name: "userAgent", Type: types.Text(), ReadOptional: true},
		}),
		ReadOptional: true,
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
	{Name: "traits", Type: types.JSON()},
	{Name: "type", Type: types.Text().WithValues("alias", "identify", "group", "page", "screen", "track")},
	{Name: "userId", Type: types.Text()},
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

func (e rawEvent) User() (string, bool) {
	// The 'user' field of an event is populated by Identity Resolution only for
	// events stored in the data warehouse.
	// For events received from a source and then forwarded to a connector for
	// sending, the 'user' is therefore never set.
	// This method always returns an empty string and false, and exists solely
	// to maintain consistency with the event schema, which always includes the
	// 'user' field by definition.
	return "", false
}

func (e rawEvent) AnonymousId() string {
	return e.event["anonymousId"].(string)
}

func (e rawEvent) Channel() (string, bool) {
	channel, ok := e.event["channel"].(string)
	return channel, ok
}

func (e rawEvent) Category() (string, bool) {
	category, ok := e.event["category"].(string)
	return category, ok
}

func (e rawEvent) Context() (meergo.RawEventContext, bool) {
	if context, ok := e.event["context"].(map[string]any); ok {
		return rawEventContext{context}, true
	}
	return nil, false
}

func (e rawEvent) Event() (string, bool) {
	event, ok := e.event["event"].(string)
	return event, ok
}

func (e rawEvent) GroupId() (string, bool) {
	groupId, ok := e.event["groupId"].(string)
	return groupId, ok
}

func (e rawEvent) MessageId() string {
	return e.event["messageId"].(string)
}

func (e rawEvent) Name() (string, bool) {
	name, ok := e.event["name"].(string)
	return name, ok
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

func (e rawEvent) UserId() (string, bool) {
	userId, ok := e.event["userId"].(string)
	return userId, ok
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

func (c rawEventContext) IP() (string, bool) {
	ip, ok := c.context["ip"].(string)
	return ip, ok
}

func (c rawEventContext) Library() (meergo.RawEventContextLibrary, bool) {
	if library, ok := c.context["library"].(map[string]any); ok {
		return rawEventContextLibrary{library}, true
	}
	return nil, false
}

func (c rawEventContext) Locale() (string, bool) {
	locale, ok := c.context["locale"].(string)
	return locale, ok
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

func (c rawEventContext) Timezone() (string, bool) {
	timezone, ok := c.context["timezone"].(string)
	return timezone, ok
}

func (c rawEventContext) UserAgent() (string, bool) {
	userAgent, ok := c.context["userAgent"].(string)
	return userAgent, ok
}

type rawEventContextApp struct {
	app map[string]any
}

func (c rawEventContextApp) Name() (string, bool) {
	name, ok := c.app["name"].(string)
	return name, ok
}

func (c rawEventContextApp) Version() (string, bool) {
	version, ok := c.app["version"].(string)
	return version, ok
}

func (c rawEventContextApp) Build() (string, bool) {
	build, ok := c.app["build"].(string)
	return build, ok
}

func (c rawEventContextApp) Namespace() (string, bool) {
	namespace, ok := c.app["namespace"].(string)
	return namespace, ok
}

type rawEventContextBrowser struct {
	browser map[string]any
}

func (c rawEventContextBrowser) Name() (string, bool) {
	name, ok := c.browser["name"].(string)
	return name, ok
}

func (c rawEventContextBrowser) Other() (string, bool) {
	other, ok := c.browser["other"].(string)
	return other, ok
}

func (c rawEventContextBrowser) Version() (string, bool) {
	version, ok := c.browser["version"].(string)
	return version, ok
}

type rawEventContextCampaign struct {
	campaign map[string]any
}

func (c rawEventContextCampaign) Name() (string, bool) {
	name, ok := c.campaign["name"].(string)
	return name, ok
}

func (c rawEventContextCampaign) Source() (string, bool) {
	source, ok := c.campaign["source"].(string)
	return source, ok
}

func (c rawEventContextCampaign) Medium() (string, bool) {
	medium, ok := c.campaign["medium"].(string)
	return medium, ok
}

func (c rawEventContextCampaign) Term() (string, bool) {
	term, ok := c.campaign["term"].(string)
	return term, ok
}

func (c rawEventContextCampaign) Content() (string, bool) {
	content, ok := c.campaign["content"].(string)
	return content, ok
}

type rawEventContextDevice struct {
	device map[string]any
}

func (c rawEventContextDevice) Id() (string, bool) {
	id, ok := c.device["id"].(string)
	return id, ok
}

func (c rawEventContextDevice) AdvertisingId() (string, bool) {
	advertisingId, ok := c.device["advertisingId"].(string)
	return advertisingId, ok
}

func (c rawEventContextDevice) AdTrackingEnabled() (bool, bool) {
	adTrackingEnabled, ok := c.device["adTrackingEnabled"].(bool)
	return adTrackingEnabled, ok
}

func (c rawEventContextDevice) Manufacturer() (string, bool) {
	manufacturer, ok := c.device["manufacturer"].(string)
	return manufacturer, ok
}

func (c rawEventContextDevice) Model() (string, bool) {
	model, ok := c.device["model"].(string)
	return model, ok
}

func (c rawEventContextDevice) Name() (string, bool) {
	name, ok := c.device["name"].(string)
	return name, ok
}

func (c rawEventContextDevice) Type() (string, bool) {
	typ, ok := c.device["type"].(string)
	return typ, ok
}

func (c rawEventContextDevice) Token() (string, bool) {
	token, ok := c.device["token"].(string)
	return token, ok
}

type rawEventContextLibrary struct {
	library map[string]any
}

func (c rawEventContextLibrary) Name() (string, bool) {
	name, ok := c.library["name"].(string)
	return name, ok
}

func (c rawEventContextLibrary) Version() (string, bool) {
	version, ok := c.library["version"].(string)
	return version, ok
}

type rawEventContextLocation struct {
	location map[string]any
}

func (c rawEventContextLocation) City() (string, bool) {
	city, ok := c.location["city"].(string)
	return city, ok
}

func (c rawEventContextLocation) Country() (string, bool) {
	country, ok := c.location["country"].(string)
	return country, ok
}

func (c rawEventContextLocation) Latitude() (float64, bool) {
	latitude, ok := c.location["latitude"].(float64)
	return latitude, ok
}

func (c rawEventContextLocation) Longitude() (float64, bool) {
	longitude, ok := c.location["longitude"].(float64)
	return longitude, ok
}

func (c rawEventContextLocation) Speed() (float64, bool) {
	speed, ok := c.location["speed"].(float64)
	return speed, ok
}

type rawEventContextNetwork struct {
	network map[string]any
}

func (c rawEventContextNetwork) Bluetooth() (bool, bool) {
	bluetooth, ok := c.network["bluetooth"].(bool)
	return bluetooth, ok
}

func (c rawEventContextNetwork) Carrier() (string, bool) {
	carrier, ok := c.network["carrier"].(string)
	return carrier, ok
}

func (c rawEventContextNetwork) Cellular() (bool, bool) {
	cellular, ok := c.network["cellular"].(bool)
	return cellular, ok
}

func (c rawEventContextNetwork) WiFi() (bool, bool) {
	wifi, ok := c.network["wifi"].(bool)
	return wifi, ok
}

type rawEventContextOS struct {
	os map[string]any
}

func (c rawEventContextOS) Name() (string, bool) {
	name, ok := c.os["name"].(string)
	return name, ok
}

func (c rawEventContextOS) Version() (string, bool) {
	version, ok := c.os["version"].(string)
	return version, ok
}

type rawEventContextPage struct {
	page map[string]any
}

func (c rawEventContextPage) Path() (string, bool) {
	path, ok := c.page["path"].(string)
	return path, ok
}

func (c rawEventContextPage) Referrer() (string, bool) {
	referrer, ok := c.page["referrer"].(string)
	return referrer, ok
}

func (c rawEventContextPage) Search() (string, bool) {
	search, ok := c.page["search"].(string)
	return search, ok
}

func (c rawEventContextPage) Title() (string, bool) {
	title, ok := c.page["title"].(string)
	return title, ok
}

func (c rawEventContextPage) URL() (string, bool) {
	url, ok := c.page["url"].(string)
	return url, ok
}

type rawEventContextReferrer struct {
	referrer map[string]any
}

func (c rawEventContextReferrer) Id() (string, bool) {
	id, ok := c.referrer["id"].(string)
	return id, ok
}

func (c rawEventContextReferrer) Type() (string, bool) {
	typ, ok := c.referrer["type"].(string)
	return typ, ok
}

type rawEventContextScreen struct {
	screen map[string]any
}

func (c rawEventContextScreen) Width() (int, bool) {
	width, ok := c.screen["width"].(int)
	return width, ok
}

func (c rawEventContextScreen) Height() (int, bool) {
	height, ok := c.screen["height"].(int)
	return height, ok
}

func (c rawEventContextScreen) Density() (decimal.Decimal, bool) {
	density, ok := c.screen["density"].(decimal.Decimal)
	return density, ok
}

type rawEventContextSession struct {
	session map[string]any
}

func (c rawEventContextSession) Id() (int, bool) {
	id, ok := c.session["id"].(int)
	return id, ok
}

func (c rawEventContextSession) Start() (bool, bool) {
	start, ok := c.session["start"].(bool)
	return start, ok
}

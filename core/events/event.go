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
	{Name: "previousId", Type: types.Text(), ReadOptional: true},
	{Name: "userId", Type: types.Text()},
})

// receivedEvent implements the meergo.ReceivedEvent interface. A ReceivedEvent
// is passed to the SendEvents method of an app connector.
type receivedEvent struct {
	event Event
}

// ReceivedEvent wraps a meergo.Event and returns a value that implements the
// meergo.ReceivedEvent interface.
//
// The provided event must conform to the event schema (Schema), otherwise
// calling methods on the returned value may cause a panic.
func ReceivedEvent(event Event) meergo.ReceivedEvent {
	return receivedEvent{event}
}

func (e receivedEvent) AnonymousId() string {
	return e.event["anonymousId"].(string)
}

func (e receivedEvent) Channel() (string, bool) {
	channel, ok := e.event["channel"].(string)
	return channel, ok
}

func (e receivedEvent) Category() (string, bool) {
	category, ok := e.event["category"].(string)
	return category, ok
}

func (e receivedEvent) Context() (meergo.ReceivedEventContext, bool) {
	if context, ok := e.event["context"].(map[string]any); ok {
		return receivedEventContext{context}, true
	}
	return nil, false
}

func (e receivedEvent) Event() (string, bool) {
	event, ok := e.event["event"].(string)
	return event, ok
}

func (e receivedEvent) GroupId() (string, bool) {
	groupId, ok := e.event["groupId"].(string)
	return groupId, ok
}

func (e receivedEvent) PreviousId() (string, bool) {
	previousId, ok := e.event["messageId"].(string)
	return previousId, ok
}

func (e receivedEvent) MessageId() string {
	return e.event["messageId"].(string)
}

func (e receivedEvent) Name() (string, bool) {
	name, ok := e.event["name"].(string)
	return name, ok
}

func (e receivedEvent) ReceivedAt() time.Time {
	return e.event["receivedAt"].(time.Time)
}

func (e receivedEvent) SentAt() time.Time {
	return e.event["sentAt"].(time.Time)
}

func (e receivedEvent) Timestamp() time.Time {
	return e.event["timestamp"].(time.Time)
}

func (e receivedEvent) Type() string {
	return e.event["type"].(string)
}

func (e receivedEvent) UserId() (string, bool) {
	userId, ok := e.event["userId"].(string)
	return userId, ok
}

type receivedEventContext struct {
	context map[string]any
}

func (c receivedEventContext) App() (meergo.ReceivedEventContextApp, bool) {
	if app, ok := c.context["app"].(map[string]any); ok {
		return receivedEventContextApp{app}, true
	}
	return nil, false
}

func (c receivedEventContext) Browser() (meergo.ReceivedEventContextBrowser, bool) {
	if browser, ok := c.context["browser"].(map[string]any); ok {
		return receivedEventContextBrowser{browser}, true
	}
	return nil, false
}

func (c receivedEventContext) Campaign() (meergo.ReceivedEventContextCampaign, bool) {
	if campaign, ok := c.context["campaign"].(map[string]any); ok {
		return receivedEventContextCampaign{campaign}, true
	}
	return nil, false
}

func (c receivedEventContext) Device() (meergo.ReceivedEventContextDevice, bool) {
	if campaign, ok := c.context["device"].(map[string]any); ok {
		return receivedEventContextDevice{campaign}, true
	}
	return nil, false
}

func (c receivedEventContext) IP() (string, bool) {
	ip, ok := c.context["ip"].(string)
	return ip, ok
}

func (c receivedEventContext) Library() (meergo.ReceivedEventContextLibrary, bool) {
	if library, ok := c.context["library"].(map[string]any); ok {
		return receivedEventContextLibrary{library}, true
	}
	return nil, false
}

func (c receivedEventContext) Locale() (string, bool) {
	locale, ok := c.context["locale"].(string)
	return locale, ok
}

func (c receivedEventContext) Location() (meergo.ReceivedEventContextLocation, bool) {
	if location, ok := c.context["location"].(map[string]any); ok {
		return receivedEventContextLocation{location}, true
	}
	return nil, false
}

func (c receivedEventContext) Network() (meergo.ReceivedEventContextNetwork, bool) {
	if network, ok := c.context["network"].(map[string]any); ok {
		return receivedEventContextNetwork{network}, true
	}
	return nil, false
}

func (c receivedEventContext) OS() (meergo.ReceivedEventContextOS, bool) {
	if os, ok := c.context["os"].(map[string]any); ok {
		return receivedEventContextOS{os}, true
	}
	return nil, false
}

func (c receivedEventContext) Page() (meergo.ReceivedEventContextPage, bool) {
	if page, ok := c.context["page"].(map[string]any); ok {
		return receivedEventContextPage{page}, true
	}
	return nil, false
}

func (c receivedEventContext) Referrer() (meergo.ReceivedEventContextReferrer, bool) {
	if referrer, ok := c.context["referrer"].(map[string]any); ok {
		return receivedEventContextReferrer{referrer}, true
	}
	return nil, false
}

func (c receivedEventContext) Screen() (meergo.ReceivedEventContextScreen, bool) {
	if screen, ok := c.context["screen"].(map[string]any); ok {
		return receivedEventContextScreen{screen}, true
	}
	return nil, false
}

func (c receivedEventContext) Session() (meergo.ReceivedEventContextSession, bool) {
	if session, ok := c.context["session"].(map[string]any); ok {
		return receivedEventContextSession{session}, true
	}
	return nil, false
}

func (c receivedEventContext) Timezone() (string, bool) {
	timezone, ok := c.context["timezone"].(string)
	return timezone, ok
}

func (c receivedEventContext) UserAgent() (string, bool) {
	userAgent, ok := c.context["userAgent"].(string)
	return userAgent, ok
}

type receivedEventContextApp struct {
	app map[string]any
}

func (c receivedEventContextApp) Name() (string, bool) {
	name, ok := c.app["name"].(string)
	return name, ok
}

func (c receivedEventContextApp) Version() (string, bool) {
	version, ok := c.app["version"].(string)
	return version, ok
}

func (c receivedEventContextApp) Build() (string, bool) {
	build, ok := c.app["build"].(string)
	return build, ok
}

func (c receivedEventContextApp) Namespace() (string, bool) {
	namespace, ok := c.app["namespace"].(string)
	return namespace, ok
}

type receivedEventContextBrowser struct {
	browser map[string]any
}

func (c receivedEventContextBrowser) Name() (string, bool) {
	name, ok := c.browser["name"].(string)
	return name, ok
}

func (c receivedEventContextBrowser) Other() (string, bool) {
	other, ok := c.browser["other"].(string)
	return other, ok
}

func (c receivedEventContextBrowser) Version() (string, bool) {
	version, ok := c.browser["version"].(string)
	return version, ok
}

type receivedEventContextCampaign struct {
	campaign map[string]any
}

func (c receivedEventContextCampaign) Name() (string, bool) {
	name, ok := c.campaign["name"].(string)
	return name, ok
}

func (c receivedEventContextCampaign) Source() (string, bool) {
	source, ok := c.campaign["source"].(string)
	return source, ok
}

func (c receivedEventContextCampaign) Medium() (string, bool) {
	medium, ok := c.campaign["medium"].(string)
	return medium, ok
}

func (c receivedEventContextCampaign) Term() (string, bool) {
	term, ok := c.campaign["term"].(string)
	return term, ok
}

func (c receivedEventContextCampaign) Content() (string, bool) {
	content, ok := c.campaign["content"].(string)
	return content, ok
}

type receivedEventContextDevice struct {
	device map[string]any
}

func (c receivedEventContextDevice) Id() (string, bool) {
	id, ok := c.device["id"].(string)
	return id, ok
}

func (c receivedEventContextDevice) AdvertisingId() (string, bool) {
	advertisingId, ok := c.device["advertisingId"].(string)
	return advertisingId, ok
}

func (c receivedEventContextDevice) AdTrackingEnabled() (bool, bool) {
	adTrackingEnabled, ok := c.device["adTrackingEnabled"].(bool)
	return adTrackingEnabled, ok
}

func (c receivedEventContextDevice) Manufacturer() (string, bool) {
	manufacturer, ok := c.device["manufacturer"].(string)
	return manufacturer, ok
}

func (c receivedEventContextDevice) Model() (string, bool) {
	model, ok := c.device["model"].(string)
	return model, ok
}

func (c receivedEventContextDevice) Name() (string, bool) {
	name, ok := c.device["name"].(string)
	return name, ok
}

func (c receivedEventContextDevice) Type() (string, bool) {
	typ, ok := c.device["type"].(string)
	return typ, ok
}

func (c receivedEventContextDevice) Token() (string, bool) {
	token, ok := c.device["token"].(string)
	return token, ok
}

type receivedEventContextLibrary struct {
	library map[string]any
}

func (c receivedEventContextLibrary) Name() (string, bool) {
	name, ok := c.library["name"].(string)
	return name, ok
}

func (c receivedEventContextLibrary) Version() (string, bool) {
	version, ok := c.library["version"].(string)
	return version, ok
}

type receivedEventContextLocation struct {
	location map[string]any
}

func (c receivedEventContextLocation) City() (string, bool) {
	city, ok := c.location["city"].(string)
	return city, ok
}

func (c receivedEventContextLocation) Country() (string, bool) {
	country, ok := c.location["country"].(string)
	return country, ok
}

func (c receivedEventContextLocation) Latitude() (float64, bool) {
	latitude, ok := c.location["latitude"].(float64)
	return latitude, ok
}

func (c receivedEventContextLocation) Longitude() (float64, bool) {
	longitude, ok := c.location["longitude"].(float64)
	return longitude, ok
}

func (c receivedEventContextLocation) Speed() (float64, bool) {
	speed, ok := c.location["speed"].(float64)
	return speed, ok
}

type receivedEventContextNetwork struct {
	network map[string]any
}

func (c receivedEventContextNetwork) Bluetooth() (bool, bool) {
	bluetooth, ok := c.network["bluetooth"].(bool)
	return bluetooth, ok
}

func (c receivedEventContextNetwork) Carrier() (string, bool) {
	carrier, ok := c.network["carrier"].(string)
	return carrier, ok
}

func (c receivedEventContextNetwork) Cellular() (bool, bool) {
	cellular, ok := c.network["cellular"].(bool)
	return cellular, ok
}

func (c receivedEventContextNetwork) WiFi() (bool, bool) {
	wifi, ok := c.network["wifi"].(bool)
	return wifi, ok
}

type receivedEventContextOS struct {
	os map[string]any
}

func (c receivedEventContextOS) Name() (string, bool) {
	name, ok := c.os["name"].(string)
	return name, ok
}

func (c receivedEventContextOS) Version() (string, bool) {
	version, ok := c.os["version"].(string)
	return version, ok
}

type receivedEventContextPage struct {
	page map[string]any
}

func (c receivedEventContextPage) Path() (string, bool) {
	path, ok := c.page["path"].(string)
	return path, ok
}

func (c receivedEventContextPage) Referrer() (string, bool) {
	referrer, ok := c.page["referrer"].(string)
	return referrer, ok
}

func (c receivedEventContextPage) Search() (string, bool) {
	search, ok := c.page["search"].(string)
	return search, ok
}

func (c receivedEventContextPage) Title() (string, bool) {
	title, ok := c.page["title"].(string)
	return title, ok
}

func (c receivedEventContextPage) URL() (string, bool) {
	url, ok := c.page["url"].(string)
	return url, ok
}

type receivedEventContextReferrer struct {
	referrer map[string]any
}

func (c receivedEventContextReferrer) Id() (string, bool) {
	id, ok := c.referrer["id"].(string)
	return id, ok
}

func (c receivedEventContextReferrer) Type() (string, bool) {
	typ, ok := c.referrer["type"].(string)
	return typ, ok
}

type receivedEventContextScreen struct {
	screen map[string]any
}

func (c receivedEventContextScreen) Width() (int, bool) {
	width, ok := c.screen["width"].(int)
	return width, ok
}

func (c receivedEventContextScreen) Height() (int, bool) {
	height, ok := c.screen["height"].(int)
	return height, ok
}

func (c receivedEventContextScreen) Density() (decimal.Decimal, bool) {
	density, ok := c.screen["density"].(decimal.Decimal)
	return density, ok
}

type receivedEventContextSession struct {
	session map[string]any
}

func (c receivedEventContextSession) Id() (int, bool) {
	id, ok := c.session["id"].(int)
	return id, ok
}

func (c receivedEventContextSession) Start() (bool, bool) {
	start, ok := c.session["start"].(bool)
	return start, ok
}

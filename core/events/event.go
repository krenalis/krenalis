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

// ConnectorEvent implements the meergo.Event interface. A ConnectorEvent is
// passed as an argument to the SendEvent and PreviewSendEvent methods of an app
// connector.
type ConnectorEvent struct {
	event Event
}

func NewConnectorEvent(event Event) ConnectorEvent {
	return ConnectorEvent{event}
}

func (c ConnectorEvent) AnonymousId() string {
	return c.event["anonymousId"].(string)
}

func (c ConnectorEvent) Channel() string {
	return c.event["channel"].(string)
}

func (c ConnectorEvent) Category() string {
	return c.event["category"].(string)
}

func (c ConnectorEvent) Context() meergo.EventContext {
	return connectorEventContext{c.event["context"].(map[string]any)}
}

func (c ConnectorEvent) Event() string {
	return c.event["event"].(string)
}

func (c ConnectorEvent) GroupId() string {
	groupId, _ := c.event["groupId"].(string)
	return groupId
}

func (c ConnectorEvent) MessageId() string {
	return c.event["messageId"].(string)
}

func (c ConnectorEvent) Name() string {
	return c.event["name"].(string)
}

func (c ConnectorEvent) ReceivedAt() time.Time {
	return c.event["receivedAt"].(time.Time)
}

func (c ConnectorEvent) SentAt() time.Time {
	return c.event["sentAt"].(time.Time)
}

func (c ConnectorEvent) Timestamp() time.Time {
	return c.event["timestamp"].(time.Time)
}

func (c ConnectorEvent) Type() string {
	return c.event["type"].(string)
}

func (c ConnectorEvent) UserId() string {
	userId, _ := c.event["userId"].(string)
	return userId
}

type connectorEventContext struct {
	context map[string]any
}

func (c connectorEventContext) App() (meergo.EventContextApp, bool) {
	if app, ok := c.context["app"].(map[string]any); ok {
		return connectorEventContextApp{app}, true
	}
	return nil, false
}

func (c connectorEventContext) Browser() (meergo.EventContextBrowser, bool) {
	if browser, ok := c.context["browser"].(map[string]any); ok {
		return connectorEventContextBrowser{browser}, true
	}
	return nil, false
}

func (c connectorEventContext) Campaign() (meergo.EventContextCampaign, bool) {
	if campaign, ok := c.context["campaign"].(map[string]any); ok {
		return connectorEventContextCampaign{campaign}, true
	}
	return nil, false
}

func (c connectorEventContext) Device() (meergo.EventContextDevice, bool) {
	if campaign, ok := c.context["device"].(map[string]any); ok {
		return connectorEventContextDevice{campaign}, true
	}
	return nil, false
}

func (c connectorEventContext) IP() string {
	return c.context["ip"].(string)
}

func (c connectorEventContext) Library() (meergo.EventContextLibrary, bool) {
	if library, ok := c.context["library"].(map[string]any); ok {
		return connectorEventContextLibrary{library}, true
	}
	return nil, false
}

func (c connectorEventContext) Locale() string {
	return c.context["locale"].(string)
}

func (c connectorEventContext) Location() (meergo.EventContextLocation, bool) {
	if location, ok := c.context["location"].(map[string]any); ok {
		return connectorEventContextLocation{location}, true
	}
	return nil, false
}

func (c connectorEventContext) Network() (meergo.EventContextNetwork, bool) {
	if network, ok := c.context["network"].(map[string]any); ok {
		return connectorEventContextNetwork{network}, true
	}
	return nil, false
}

func (c connectorEventContext) OS() (meergo.EventContextOS, bool) {
	if os, ok := c.context["os"].(map[string]any); ok {
		return connectorEventContextOS{os}, true
	}
	return nil, false
}

func (c connectorEventContext) Page() (meergo.EventContextPage, bool) {
	if page, ok := c.context["page"].(map[string]any); ok {
		return connectorEventContextPage{page}, true
	}
	return nil, false
}

func (c connectorEventContext) Referrer() (meergo.EventContextReferrer, bool) {
	if referrer, ok := c.context["referrer"].(map[string]any); ok {
		return connectorEventContextReferrer{referrer}, true
	}
	return nil, false
}

func (c connectorEventContext) Screen() (meergo.EventContextScreen, bool) {
	if screen, ok := c.context["screen"].(map[string]any); ok {
		return connectorEventContextScreen{screen}, true
	}
	return nil, false
}

func (c connectorEventContext) Session() (meergo.EventContextSession, bool) {
	if session, ok := c.context["Session"].(map[string]any); ok {
		return connectorEventContextSession{session}, true
	}
	return nil, false
}

func (c connectorEventContext) Timezone() string {
	return c.context["timezone"].(string)
}

func (c connectorEventContext) UserAgent() string {
	return c.context["userAgent"].(string)
}

type connectorEventContextApp struct {
	app map[string]any
}

func (c connectorEventContextApp) Name() string {
	return c.app["name"].(string)
}

func (c connectorEventContextApp) Version() string {
	return c.app["version"].(string)
}

func (c connectorEventContextApp) Build() string {
	return c.app["build"].(string)
}

func (c connectorEventContextApp) Namespace() string {
	return c.app["namespace"].(string)
}

type connectorEventContextBrowser struct {
	browser map[string]any
}

func (c connectorEventContextBrowser) Name() string {
	return c.browser["name"].(string)
}

func (c connectorEventContextBrowser) Other() string {
	return c.browser["other"].(string)
}

func (c connectorEventContextBrowser) Version() string {
	return c.browser["version"].(string)
}

type connectorEventContextCampaign struct {
	campaign map[string]any
}

func (c connectorEventContextCampaign) Name() string {
	return c.campaign["name"].(string)
}

func (c connectorEventContextCampaign) Source() string {
	return c.campaign["source"].(string)
}

func (c connectorEventContextCampaign) Medium() string {
	return c.campaign["medium"].(string)
}

func (c connectorEventContextCampaign) Term() string {
	return c.campaign["term"].(string)
}

func (c connectorEventContextCampaign) Content() string {
	return c.campaign["content"].(string)
}

type connectorEventContextDevice struct {
	device map[string]any
}

func (c connectorEventContextDevice) Id() string {
	return c.device["id"].(string)
}

func (c connectorEventContextDevice) AdvertisingId() string {
	return c.device["advertisingId"].(string)
}

func (c connectorEventContextDevice) AdTrackingEnabled() bool {
	return c.device["adTrackingEnabled"].(bool)
}

func (c connectorEventContextDevice) Manufacturer() string {
	return c.device["manufacturer"].(string)
}

func (c connectorEventContextDevice) Model() string {
	return c.device["model"].(string)
}

func (c connectorEventContextDevice) Name() string {
	return c.device["name"].(string)
}

func (c connectorEventContextDevice) Type() string {
	return c.device["type"].(string)
}

func (c connectorEventContextDevice) Token() string {
	return c.device["token"].(string)
}

type connectorEventContextLibrary struct {
	library map[string]any
}

func (c connectorEventContextLibrary) Name() string {
	return c.library["id"].(string)
}

func (c connectorEventContextLibrary) Version() string {
	return c.library["version"].(string)
}

type connectorEventContextLocation struct {
	location map[string]any
}

func (c connectorEventContextLocation) City() string {
	return c.location["city"].(string)
}

func (c connectorEventContextLocation) Country() string {
	return c.location["country"].(string)
}

func (c connectorEventContextLocation) Latitude() float64 {
	return c.location["latitude"].(float64)
}

func (c connectorEventContextLocation) Longitude() float64 {
	return c.location["longitude"].(float64)
}

func (c connectorEventContextLocation) Speed() float64 {
	return c.location["speed"].(float64)
}

type connectorEventContextNetwork struct {
	network map[string]any
}

func (c connectorEventContextNetwork) Bluetooth() bool {
	return c.network["bluetooth"].(bool)
}

func (c connectorEventContextNetwork) Carrier() string {
	return c.network["carrier"].(string)
}

func (c connectorEventContextNetwork) Cellular() bool {
	return c.network["cellular"].(bool)
}

func (c connectorEventContextNetwork) WiFi() bool {
	return c.network["wifi"].(bool)
}

type connectorEventContextOS struct {
	os map[string]any
}

func (c connectorEventContextOS) Name() string {
	return c.os["name"].(string)
}

func (c connectorEventContextOS) Version() string {
	return c.os["version"].(string)
}

type connectorEventContextPage struct {
	page map[string]any
}

func (c connectorEventContextPage) Path() string {
	return c.page["path"].(string)
}

func (c connectorEventContextPage) Referrer() string {
	return c.page["referrer"].(string)
}

func (c connectorEventContextPage) Search() string {
	return c.page["search"].(string)
}

func (c connectorEventContextPage) Title() string {
	return c.page["title"].(string)
}

func (c connectorEventContextPage) URL() string {
	return c.page["url"].(string)
}

type connectorEventContextReferrer struct {
	referrer map[string]any
}

func (c connectorEventContextReferrer) Id() string {
	return c.referrer["id"].(string)
}

func (c connectorEventContextReferrer) Type() string {
	return c.referrer["type"].(string)
}

type connectorEventContextScreen struct {
	screen map[string]any
}

func (c connectorEventContextScreen) Width() int {
	return c.screen["id"].(int)
}

func (c connectorEventContextScreen) Height() int {
	return c.screen["height"].(int)
}

func (c connectorEventContextScreen) Density() decimal.Decimal {
	return c.screen["density"].(decimal.Decimal)
}

type connectorEventContextSession struct {
	session map[string]any
}

func (c connectorEventContextSession) Id() int {
	return c.session["id"].(int)
}

func (c connectorEventContextSession) Start() bool {
	start, _ := c.session["start"].(bool)
	return start
}

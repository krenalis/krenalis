//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"net/http"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/json"

	"github.com/shopspring/decimal"
)

// Header represents the Header of an event.
type Header struct {
	ReceivedAt time.Time   `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
	Source     int
}

// Event represents an event.
type Event struct {
	Header       *Header
	Id           [20]byte
	AnonymousId  string
	Category     string
	Context      Context
	Event        string
	GroupId      string
	Integrations json.Value
	MessageId    string
	Name         string
	ReceivedAt   time.Time
	SentAt       time.Time
	Timestamp    time.Time
	Traits       json.Value
	Type         *string
	UserId       string
	PreviousId   string
	Properties   json.Value
}

type Context struct {
	App struct {
		Name      string `json:"name,omitempty"`
		Version   string `json:"version,omitempty"`
		Build     string `json:"build,omitempty"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"app,omitempty"`
	Browser struct { // TODO: this should be unexported
		Name    string `json:"name,omitempty"`
		Other   string `json:"other,omitempty"`
		Version string `json:"version,omitempty"`
	}
	Campaign struct {
		Name    string `json:"name,omitempty"`
		Source  string `json:"source,omitempty"`
		Medium  string `json:"medium,omitempty"`
		Term    string `json:"term,omitempty"`
		Content string `json:"content,omitempty"`
	} `json:"campaign,omitempty"`
	Device struct {
		Id                string `json:"id,omitempty"`
		AdvertisingId     string `json:"advertisingId,omitempty"`
		AdTrackingEnabled bool   `json:"adTrackingEnabled,omitempty"`
		Manufacturer      string `json:"manufacturer,omitempty"`
		Model             string `json:"model,omitempty"`
		Name              string `json:"name,omitempty"`
		Type              string `json:"type,omitempty"`
		Token             string `json:"token,omitempty"`
	} `json:"device,omitempty"`
	Direct  bool   `json:"direct,omitempty"`
	IP      string `json:"ip,omitempty"`
	Library struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"library,omitempty"`
	Locale   string `json:"locale,omitempty"`
	Location struct {
		City      string  `json:"city,omitempty"`
		Country   string  `json:"country,omitempty"`
		Latitude  float64 `json:"latitude,omitempty"`
		Longitude float64 `json:"longitude,omitempty"`
		Speed     float64 `json:"speed,omitempty"`
	} `json:"location,omitempty"`
	Network struct {
		Bluetooth bool   `json:"bluetooth,omitempty"`
		Carrier   string `json:"carrier,omitempty"`
		Cellular  bool   `json:"cellular,omitempty"`
		WiFi      bool   `json:"wifi,omitempty"`
	} `json:"network,omitempty"`
	OS struct {
		Name    string `json:"name,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"os,omitempty"`
	Page struct {
		Path     string `json:"path,omitempty"`
		Referrer string `json:"referrer,omitempty"`
		Search   string `json:"search,omitempty"`
		Title    string `json:"title,omitempty"`
		URL      string `json:"url,omitempty"`
	} `json:"page,omitempty"`
	Referrer struct {
		Id   string `json:"id,omitempty"`
		Type string `json:"type,omitempty"`
	} `json:"referrer,omitempty"`
	Screen struct {
		Width   int     `json:"width,omitempty"`
		Height  int     `json:"height,omitempty"`
		Density float32 `json:"density,omitempty"`
	} `json:"screen,omitempty"`
	SessionId    int            `json:"sessionId,omitempty"`
	SessionStart bool           `json:"sessionStart,omitempty"`
	GroupId      string         `json:"groupId,omitempty"`
	Timezone     string         `json:"timezone,omitempty"`
	Traits       map[string]any `json:"traits,omitempty"`
	UserAgent    string         `json:"userAgent,omitempty"`
}

// ToConnectorEvent returns event as a connector event to be passed as an
// argument to the SendEvent and PreviewSendEvent methods of an app connector.
func (event *Event) ToConnectorEvent() *meergo.Event {
	// Keep in sync with the connector.EventI type.
	groupId := event.GroupId
	if event.GroupId == "" {
		groupId = event.Context.GroupId
	}
	e := meergo.Event{}
	e.AnonymousId = event.AnonymousId
	e.Category = event.Category
	e.Context.App.Name = event.Context.App.Name
	e.Context.App.Version = event.Context.App.Version
	e.Context.App.Build = event.Context.App.Build
	e.Context.App.Namespace = event.Context.App.Namespace
	e.Context.Campaign.Name = event.Context.Campaign.Name
	e.Context.Campaign.Source = event.Context.Campaign.Source
	e.Context.Campaign.Medium = event.Context.Campaign.Medium
	e.Context.Campaign.Term = event.Context.Campaign.Term
	e.Context.Campaign.Content = event.Context.Campaign.Content
	e.Context.Device.Id = event.Context.Device.Id
	e.Context.Device.AdvertisingId = event.Context.Device.AdvertisingId
	e.Context.Device.AdTrackingEnabled = event.Context.Device.AdTrackingEnabled
	e.Context.Device.Manufacturer = event.Context.Device.Manufacturer
	e.Context.Device.Model = event.Context.Device.Model
	e.Context.Device.Name = event.Context.Device.Name
	e.Context.Device.Type = event.Context.Device.Type
	e.Context.Device.Token = event.Context.Device.Token
	e.Context.Device.Token = event.Context.Device.Token
	e.Context.IP = event.Context.IP
	e.Context.Library.Name = event.Context.Library.Name
	e.Context.Library.Version = event.Context.Library.Version
	e.Context.Locale = event.Context.Locale
	e.Context.Location.City = event.Context.Location.City
	e.Context.Location.Country = event.Context.Location.Country
	e.Context.Location.Latitude = event.Context.Location.Latitude
	e.Context.Location.Longitude = event.Context.Location.Longitude
	e.Context.Location.Speed = event.Context.Location.Speed
	e.Context.Network.Bluetooth = event.Context.Network.Bluetooth
	e.Context.Network.Carrier = event.Context.Network.Carrier
	e.Context.Network.Cellular = event.Context.Network.Cellular
	e.Context.Network.WiFi = event.Context.Network.WiFi
	e.Context.OS.Name = event.Context.OS.Name
	e.Context.OS.Version = event.Context.OS.Version
	e.Context.Page.Path = event.Context.Page.Path
	e.Context.Page.Referrer = event.Context.Page.Referrer
	e.Context.Page.Search = event.Context.Page.Search
	e.Context.Page.Title = event.Context.Page.Path
	e.Context.Page.URL = event.Context.Page.URL
	e.Context.Referrer.Id = event.Context.Referrer.Id
	e.Context.Referrer.Type = event.Context.Referrer.Type
	e.Context.Screen.Width = event.Context.Screen.Width
	e.Context.Screen.Height = event.Context.Screen.Height
	e.Context.Screen.Density = decimal.NewFromFloat(float64(event.Context.Screen.Density)).Round(2)
	e.Context.Session.Id = event.Context.SessionId
	e.Context.Session.Start = event.Context.SessionStart
	e.Context.Timezone = event.Context.Timezone
	e.Context.UserAgent = event.Context.UserAgent
	e.Event = event.Event
	e.GroupId = groupId
	e.MessageId = event.MessageId
	e.Name = event.Name
	e.ReceivedAt = event.ReceivedAt
	e.SentAt = event.SentAt
	e.Timestamp = event.Timestamp
	e.Type = *event.Type
	e.UserId = event.UserId
	return &e
}

// AsProperties converts the event into properties that conform to Schema.
func (event *Event) AsProperties() map[string]any {

	// Keep in sync with the schema in "schema.go".

	context := map[string]any{}

	// TODO(Gianluca): define datetime layout and parse/convert the values.
	mapEvent := map[string]any{
		"anonymousId": event.AnonymousId,
		"context":     context,
		"messageId":   event.MessageId,
		"receivedAt":  event.ReceivedAt,
		"sentAt":      event.SentAt,
		"source":      event.Header.Source,
		"timestamp":   event.Timestamp,
		"type":        *event.Type,
	}

	if event.UserId == "" {
		mapEvent["userId"] = nil
	} else {
		mapEvent["userId"] = event.UserId
	}

	if event.Category != "" {
		mapEvent["category"] = event.Category
	}
	if event.Context.App.Name != "" {
		context["app"] = map[string]any{
			"name":      event.Context.App.Name,
			"version":   event.Context.App.Version,
			"build":     event.Context.App.Build,
			"namespace": event.Context.App.Namespace,
		}
	}
	if event.Context.Browser.Name != "None" {
		context["browser"] = map[string]any{
			"name":    event.Context.Browser.Name,
			"other":   event.Context.Browser.Other,
			"version": event.Context.Browser.Version,
		}
	}
	if event.Context.Campaign.Name != "" {
		context["campaign"] = map[string]any{
			"name":    event.Context.Campaign.Name,
			"source":  event.Context.Campaign.Source,
			"medium":  event.Context.Campaign.Medium,
			"term":    event.Context.Campaign.Term,
			"content": event.Context.Campaign.Content,
		}
	}
	if event.Context.Device.Id != "" {
		context["device"] = map[string]any{
			"id":                event.Context.Device.Id,
			"advertisingId":     event.Context.Device.AdvertisingId,
			"adTrackingEnabled": event.Context.Device.AdTrackingEnabled,
			"manufacturer":      event.Context.Device.Manufacturer,
			"model":             event.Context.Device.Model,
			"name":              event.Context.Device.Name,
			"type":              event.Context.Device.Type,
			"token":             event.Context.Device.Token,
		}
	}
	if event.Context.IP != "" {
		context["ip"] = event.Context.IP
	}
	if event.Context.Library.Name != "" {
		context["library"] = map[string]any{
			"name":    event.Context.Library.Name,
			"version": event.Context.Library.Version,
		}
	}
	if event.Context.Locale != "" {
		context["locale"] = event.Context.Locale
	}
	if event.Context.Locale != "" {
		context["location"] = map[string]any{
			"city":      event.Context.Location.City,
			"country":   event.Context.Location.Country,
			"latitude":  event.Context.Location.Latitude,
			"longitude": event.Context.Location.Longitude,
			"speed":     event.Context.Location.Speed,
		}
	}
	if event.Context.Network.Carrier != "" {
		context["network"] = map[string]any{
			"bluetooth": event.Context.Network.Bluetooth,
			"carrier":   event.Context.Network.Carrier,
			"cellular":  event.Context.Network.Cellular,
			"wifi":      event.Context.Network.WiFi,
		}
	}
	if event.Context.OS.Name != "None" {
		context["os"] = map[string]any{
			"name":    event.Context.OS.Name,
			"version": event.Context.OS.Version,
		}
	}
	if event.Context.Page.Path != "" {
		context["page"] = map[string]any{
			"path":     event.Context.Page.Path,
			"referrer": event.Context.Page.Referrer,
			"search":   event.Context.Page.Search,
			"title":    event.Context.Page.Title,
			"url":      event.Context.Page.URL,
		}
	}
	if event.Context.Referrer.Id != "" {
		context["referrer"] = map[string]any{
			"id":   event.Context.Referrer.Id,
			"type": event.Context.Referrer.Type,
		}
	}
	if event.Context.Screen.Width != 0 {
		context["screen"] = map[string]any{
			"width":   event.Context.Screen.Width,
			"height":  event.Context.Screen.Height,
			"density": decimal.NewFromFloat(float64(event.Context.Screen.Density)).Round(2),
		}
	}
	if event.Context.SessionId != 0 {
		session := map[string]any{
			"id": event.Context.SessionId,
		}
		if event.Context.SessionStart {
			session["start"] = event.Context.SessionStart
		}
		context["session"] = session
	}
	if event.Context.Timezone != "" {
		context["timezone"] = event.Context.Timezone
	}
	if event.Context.UserAgent != "" {
		context["userAgent"] = event.Context.UserAgent
	}
	if event.Context.Traits != nil {
		context["traits"] = event.Context.Traits
	}
	if event.Event != "" {
		mapEvent["event"] = event.Event
	}
	if event.GroupId != "" {
		mapEvent["groupId"] = event.GroupId
	} else if event.Context.GroupId != "" {
		mapEvent["groupId"] = event.Context.GroupId
	}
	if event.Name != "" {
		mapEvent["name"] = event.Name
	}
	if event.Properties != nil {
		mapEvent["properties"] = event.Properties
	}
	if event.Traits != nil {
		mapEvent["traits"] = event.Traits
	}

	return mapEvent
}

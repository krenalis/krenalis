//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"chichi/apis/datastore"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/errors"
	"chichi/apis/state"
	"chichi/connector/types"
)

// User represents a user.
type User struct {
	workspace *state.Workspace
	store     *datastore.Store
	id        int
}

// Event represents a user event.
type Event struct {
	AnonymousId string `json:"anonymousId,omitempty"`
	Category    string `json:"category,omitempty"`
	Context     struct {
		Active       bool                  `json:"active,omitempty"`
		App          *EventContextApp      `json:"app,omitempty"`
		Browser      *EventContextBrowser  `json:"browser,omitempty"`
		Campaign     *EventContextCampaign `json:"campaign,omitempty"`
		Device       *EventContextDevice   `json:"device,omitempty"`
		IP           string                `json:"ip,omitempty"`
		Library      *EventContextLibrary  `json:"library,omitempty"`
		Locale       string                `json:"locale,omitempty"`
		Location     *EventContextLocation `json:"location,omitempty"`
		Network      *EventContextNetwork  `json:"network,omitempty"`
		OS           *EventContextOS       `json:"os,omitempty"`
		Page         *EventContextPage     `json:"page,omitempty"`
		Referrer     *EventContextReferrer `json:"referrer,omitempty"`
		Screen       *EventContextScreen   `json:"screen,omitempty"`
		SessionId    string                `json:"sessionId,omitempty"`
		SessionStart bool                  `json:"sessionStart,omitempty"`
		Timezone     string                `json:"timezone,omitempty"`
		UserAgent    string                `json:"userAgent,omitempty"`
	} `json:"context"`
	Event      string          `json:"event,omitempty"`
	GroupId    string          `json:"groupId,omitempty"`
	MessageId  string          `json:"messageId,omitempty"`
	Name       string          `json:"name,omitempty"`
	Properties json.RawMessage `json:"properties,omitempty"`
	ReceivedAt string          `json:"receivedAt,omitempty"`
	SentAt     string          `json:"sentAt,omitempty"`
	Source     int             `json:"source,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	Traits     json.RawMessage `json:"traits,omitempty"`
	Type       string          `json:"type,omitempty"`
	UserId     string          `json:"userId,omitempty"`
}

type EventContextApp struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Build     string `json:"build"`
	Namespace string `json:"namespace"`
}

type EventContextBrowser struct {
	Name    string `json:"name"`
	Other   string `json:"other"`
	Version string `json:"version"`
}

type EventContextCampaign struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Medium  string `json:"medium"`
	Term    string `json:"term"`
	Content string `json:"content"`
}

type EventContextDevice struct {
	Id                string `json:"id"`
	AdvertisingId     string `json:"advertisingId"`
	AdTrackingEnabled bool   `json:"AdTrackingEnabled"`
	Manufacturer      string `json:"manufacturer"`
	Model             string `json:"model"`
	Name              string `json:"name"`
	Type              string `json:"type"`
	Token             string `json:"token"`
}

type EventContextLibrary struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type EventContextLocation struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
}

type EventContextNetwork struct {
	Bluetooth bool   `json:"bluetooth"`
	Carrier   string `json:"carrier"`
	Cellular  bool   `json:"cellular"`
	WiFi      bool   `json:"wifi"`
}

type EventContextOS struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type EventContextPage struct {
	Path     string `json:"path"`
	Referrer string `json:"referrer"`
	Search   string `json:"search"`
	Title    string `json:"title"`
	URL      string `json:"url"`
}

type EventContextReferrer struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type EventContextScreen struct {
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Density float64 `json:"density"`
}

var eventColumns = []types.Property{
	{Name: "anonymous_id", Type: types.Text()},
	{Name: "category", Type: types.Text()},
	{Name: "app_name", Type: types.Text()},
	{Name: "app_version", Type: types.Text()},
	{Name: "app_build", Type: types.Text()},
	{Name: "app_namespace", Type: types.Text()},
	{Name: "browser_name", Type: types.Text()},
	{Name: "browser_other", Type: types.Text()},
	{Name: "browser_version", Type: types.Text()},
	{Name: "campaign_name", Type: types.Text()},
	{Name: "campaign_source", Type: types.Text()},
	{Name: "campaign_medium", Type: types.Text()},
	{Name: "campaign_term", Type: types.Text()},
	{Name: "campaign_content", Type: types.Text()},
	{Name: "device_id", Type: types.Text()},
	{Name: "device_advertising_id", Type: types.Text()},
	{Name: "device_ad_tracking_enabled", Type: types.Boolean()},
	{Name: "device_manufacturer", Type: types.Text()},
	{Name: "device_model", Type: types.Text()},
	{Name: "device_name", Type: types.Text()},
	{Name: "device_type", Type: types.Text()},
	{Name: "device_token", Type: types.Text()},
	{Name: "ip", Type: types.Inet()},
	{Name: "library_name", Type: types.Text()},
	{Name: "library_version", Type: types.Text()},
	{Name: "locale", Type: types.Text()},
	{Name: "location_city", Type: types.Text()},
	{Name: "location_country", Type: types.Text()},
	{Name: "location_latitude", Type: types.Float()},
	{Name: "location_longitude", Type: types.Float()},
	{Name: "location_speed", Type: types.Float()},
	{Name: "network_bluetooth", Type: types.Boolean()},
	{Name: "network_carrier", Type: types.Text()},
	{Name: "network_cellular", Type: types.Boolean()},
	{Name: "network_wifi", Type: types.Boolean()},
	{Name: "os_name", Type: types.Text().WithEnum([]string{"Android", "Windows", "iOS", "macOS", "Linux", "Chrome OS", "Other"})},
	{Name: "os_version", Type: types.Text()},
	{Name: "page_path", Type: types.Text()},
	{Name: "page_referrer", Type: types.Text()},
	{Name: "page_search", Type: types.Text()},
	{Name: "page_title", Type: types.Text()},
	{Name: "page_url", Type: types.Text()},
	{Name: "referrer_id", Type: types.Text()},
	{Name: "referrer_type", Type: types.Text()},
	{Name: "screen_width", Type: types.Int16()},
	{Name: "screen_height", Type: types.Int16()},
	{Name: "screen_density", Type: types.Int16()},
	{Name: "session_id", Type: types.Int64()},
	{Name: "session_start", Type: types.Boolean()},
	{Name: "timezone", Type: types.Text()},
	{Name: "user_agent", Type: types.Text()},
	{Name: "event", Type: types.Text()},
	{Name: "group_id", Type: types.Text()},
	{Name: "message_id", Type: types.Text()},
	{Name: "name", Type: types.Text()},
	{Name: "properties", Type: types.JSON()},
	{Name: "received_at", Type: types.DateTime()},
	{Name: "sent_at", Type: types.DateTime()},
	{Name: "source", Type: types.Int()},
	{Name: "timestamp", Type: types.DateTime()},
	{Name: "traits", Type: types.JSON()},
	{Name: "type", Type: types.Text().WithEnum([]string{"alias", "identify", "group", "page", "screen", "track"})},
	{Name: "user_id", Type: types.Text()},
}

// Events returns the events of the user. limit is the maximum number of events
// to return, it must be in range [1, 200].
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - WarehouseFailed, if the data warehouse failed.
func (this *User) Events(limit int) ([]Event, error) {

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", this.workspace.ID)
	}

	// Read the events.
	where := warehouses.NewBaseExpr(
		warehouses.ExprColumn{Name: "gid", Type: types.PtInt},
		warehouses.OperatorEqual,
		this.id,
	)
	rows, err := this.store.Select(context.Background(), "events", eventColumns, where, types.Property{}, 0, limit)
	if err != nil {
		if err2, ok := err.(*datastore.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get a user from the data warehouse of the workspace %d: %s", this.workspace.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return nil, err
	}

	events := make([]Event, len(rows))
	for i, row := range rows {

		var e Event

		e.AnonymousId = row[0].(string)
		e.Category = row[1].(string)

		// App.
		app := EventContextApp{
			Name:      row[2].(string),
			Version:   row[3].(string),
			Build:     row[4].(string),
			Namespace: row[5].(string),
		}
		if app != (EventContextApp{}) {
			e.Context.App = &app
		}

		// Browser.
		browser := EventContextBrowser{
			Name:    row[6].(string),
			Other:   row[7].(string),
			Version: row[8].(string),
		}
		if browser != (EventContextBrowser{}) {
			e.Context.Browser = &browser
		}

		// Campaign.
		campaign := EventContextCampaign{
			Name:    row[9].(string),
			Source:  row[10].(string),
			Medium:  row[11].(string),
			Term:    row[12].(string),
			Content: row[13].(string),
		}
		if campaign != (EventContextCampaign{}) {
			e.Context.Campaign = &campaign
		}

		// Device.
		device := EventContextDevice{
			Id:                row[14].(string),
			AdvertisingId:     row[15].(string),
			AdTrackingEnabled: row[16].(bool),
			Manufacturer:      row[17].(string),
			Model:             row[18].(string),
			Name:              row[19].(string),
			Type:              row[20].(string),
			Token:             row[21].(string),
		}
		if device != (EventContextDevice{}) {
			e.Context.Device = &device
		}

		// IP.
		e.Context.IP = row[22].(string)

		// Library.
		library := EventContextLibrary{
			Name:    row[23].(string),
			Version: row[24].(string),
		}
		if library != (EventContextLibrary{}) {
			e.Context.Library = &library
		}

		// Locale.
		e.Context.Locale = row[25].(string)

		// Location.
		location := EventContextLocation{
			City:      row[26].(string),
			Country:   row[27].(string),
			Latitude:  row[28].(float64),
			Longitude: row[29].(float64),
			Speed:     row[30].(float64),
		}
		if location != (EventContextLocation{}) {
			e.Context.Location = &location
		}

		// Network.
		network := EventContextNetwork{
			Bluetooth: row[31].(bool),
			Carrier:   row[32].(string),
			Cellular:  row[33].(bool),
			WiFi:      row[34].(bool),
		}
		if network != (EventContextNetwork{}) {
			e.Context.Network = &network
		}

		// OS.
		os := EventContextOS{
			Name:    row[35].(string),
			Version: row[36].(string),
		}
		if os != (EventContextOS{}) {
			e.Context.OS = &os
		}

		// Page.
		page := EventContextPage{
			Path:     row[37].(string),
			Referrer: row[38].(string),
			Search:   row[39].(string),
			Title:    row[40].(string),
			URL:      row[41].(string),
		}
		if page != (EventContextPage{}) {
			e.Context.Page = &page
		}

		// Referrer.
		referrer := EventContextReferrer{
			Id:   row[42].(string),
			Type: row[43].(string),
		}
		if referrer != (EventContextReferrer{}) {
			e.Context.Referrer = &referrer
		}

		// Screen.
		screen := EventContextScreen{
			Width:   row[44].(int),
			Height:  row[45].(int),
			Density: float64(row[46].(int) / 100),
		}
		if screen != (EventContextScreen{}) {
			e.Context.Screen = &screen
		}

		e.Context.SessionId = strconv.Itoa(row[47].(int))
		e.Context.SessionStart = row[48].(bool)
		e.Context.Timezone = row[49].(string)
		e.Context.UserAgent = row[50].(string)

		e.Event = row[51].(string)
		e.GroupId = row[52].(string)
		e.MessageId = row[53].(string)
		e.Name = row[54].(string)
		e.Properties = json.RawMessage(row[55].(string))
		e.ReceivedAt = row[56].(time.Time).Format(time.RFC3339)
		e.SentAt = row[57].(time.Time).Format(time.RFC3339)
		e.Source = row[58].(int)
		e.Timestamp = row[59].(time.Time).Format(time.RFC3339)
		e.Traits = json.RawMessage(row[60].(string))
		e.Type = row[61].(string)
		e.UserId = row[62].(string)

		events[i] = e

	}

	return events, nil
}

// Traits returns the traits of the user.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - NoUsersSchema, if the data warehouse does not have users schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - WarehouseFailed, if the data warehouse failed.
func (this *User) Traits() (map[string]any, error) {

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Read the schema.
	var properties []types.Property
	if typ, ok := ws.Schemas["users"]; ok {
		properties = typ.Properties()
	} else {
		return nil, errors.Unprocessable(NoUsersSchema, "workspace %d does not have users schema", this.workspace.ID)
	}

	columns := datastore.PropertiesToColumns(properties)
	where := warehouses.NewBaseExpr(
		warehouses.ExprColumn{Name: "id", Type: types.PtInt},
		warehouses.OperatorEqual,
		this.id,
	)
	rows, err := this.store.Select(context.Background(), "users", columns, where, types.Property{}, 0, 1)
	if err != nil {
		if err2, ok := err.(*datastore.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get a user from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.NotFound("user %d does not exist", this.id)
	}

	traits, _ := datastore.DeserializeRowAsMap(properties, rows[0])

	return traits, nil
}

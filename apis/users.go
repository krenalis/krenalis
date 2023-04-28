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
	"net/netip"
	"strconv"
	"time"

	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/state"
	"chichi/apis/types"
	"chichi/apis/warehouses"
)

// User represents a user.
type User struct {
	workspace *state.Workspace
	id        int
}

// Event represents a user event.
type Event struct {
	AnonymousID string `json:"anonymousId,omitempty"`
	Context     struct {
		App       *EventContextApp      `json:"app,omitempty"`
		Campaign  *EventContextCampaign `json:"campaign,omitempty"`
		Device    *EventContextDevice   `json:"device,omitempty"`
		IP        netip.Addr            `json:"ip,omitempty"`
		Library   *EventContextLibrary  `json:"library,omitempty"`
		Locale    string                `json:"locale,omitempty"`
		Location  *EventContextLocation `json:"location,omitempty"`
		Network   *EventContextNetwork  `json:"network,omitempty"`
		OS        *EventContextOS       `json:"os,omitempty"`
		Page      *EventContextPage     `json:"page,omitempty"`
		Referrer  *EventContextReferrer `json:"referrer,omitempty"`
		Screen    *EventContextScreen   `json:"screen,omitempty"`
		Timezone  string                `json:"timezone,omitempty"`
		Traits    json.RawMessage       `json:"traits,omitempty"`
		UserAgent string                `json:"userAgent,omitempty"`
	} `json:"context"`
	Event      string          `json:"event,omitempty"`
	GroupID    string          `json:"groupId,omitempty"`
	MessageID  string          `json:"messageId,omitempty"`
	Name       string          `json:"name,omitempty"`
	PreviousID string          `json:"previousId,omitempty"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
	Traits     json.RawMessage `json:"traits,omitempty"`
	Type       *string         `json:"type,omitempty"`
	UserID     string          `json:"userId,omitempty"`
}

type EventContextApp struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Build     string `json:"build"`
	Namespace string `json:"namespace"`
}

type EventContextCampaign struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Medium  string `json:"medium"`
	Term    string `json:"term"`
	Content string `json:"content"`
}

type EventContextDevice struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Manufacturer  string `json:"manufacturer"`
	Model         string `json:"model"`
	Type          string `json:"type"`
	Version       string `json:"version"`
	AdvertisingID string `json:"advertisingId"`
}

type EventContextLibrary struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type EventContextLocation struct {
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
}

type EventContextNetwork struct {
	Cellular  bool   `json:"cellular"`
	WiFi      bool   `json:"wifi"`
	Bluetooth bool   `json:"bluetooth"`
	Carrier   string `json:"carrier"`
}

type EventContextOS struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type EventContextPage struct {
	URL      string `json:"url"`
	Path     string `json:"path"`
	Search   string `json:"search"`
	Hash     string `json:"hash,omitempty"`
	Title    string `json:"title"`
	Referrer string `json:"referrer"`
}

type EventContextReferrer struct {
	Type string `json:"type"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Link string `json:"link"`
}

type EventContextScreen struct {
	Density int `json:"density"`
	Width   int `json:"width"`
	Height  int `json:"height"`
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

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if ws.Warehouse == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Read the events.
	columns := columnsOfProperties(events.Schema.Properties())
	where := warehouses.NewBaseExpr(
		warehouses.ExprColumn{Name: "user_id", Type: types.PtText},
		warehouses.OperatorEqual,
		strconv.Itoa(this.id),
	)
	rows, err := ws.Warehouse.Select(context.Background(), "events", columns, where, types.Property{}, 0, limit)
	if err != nil {
		if err2, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get a user from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return nil, err
	}

	idx := make(map[string]int, len(columns))
	for i, c := range columns {
		idx[c.Name] = i
	}

	events := make([]Event, len(rows))
	for i, row := range rows {

		var e Event

		e.AnonymousID = row[idx["anonymous_id"]].(string)

		// App.
		app := EventContextApp{
			Name:      row[idx["app_name"]].(string),
			Version:   row[idx["app_version"]].(string),
			Build:     row[idx["app_build"]].(string),
			Namespace: row[idx["app_namespace"]].(string),
		}
		if app != (EventContextApp{}) {
			e.Context.App = &app
		}

		// Campaign.
		campaign := EventContextCampaign{
			Name:    row[idx["campaign_name"]].(string),
			Source:  row[idx["campaign_source"]].(string),
			Medium:  row[idx["campaign_medium"]].(string),
			Term:    row[idx["campaign_term"]].(string),
			Content: row[idx["campaign_content"]].(string),
		}
		if campaign != (EventContextCampaign{}) {
			e.Context.Campaign = &campaign
		}

		// Device.
		device := EventContextDevice{
			ID:            row[idx["device_id"]].(string),
			Name:          row[idx["device_name"]].(string),
			Manufacturer:  row[idx["device_manufacturer"]].(string),
			Model:         row[idx["device_model"]].(string),
			Type:          row[idx["device_type"]].(string),
			Version:       row[idx["device_version"]].(string),
			AdvertisingID: row[idx["device_advertising_id"]].(string),
		}
		if device != (EventContextDevice{}) {
			e.Context.Device = &device
		}

		// IP.
		e.Context.IP = row[idx["ip"]].(netip.Addr)

		// Library.
		library := EventContextLibrary{
			Name:    row[idx["library_name"]].(string),
			Version: row[idx["library_version"]].(string),
		}
		if library != (EventContextLibrary{}) {
			e.Context.Library = &library
		}

		// Locale.
		e.Context.Locale = row[idx["locale"]].(string)

		// Location.
		location := EventContextLocation{
			City:      row[idx["location_city"]].(string),
			Country:   row[idx["location_country"]].(string),
			Region:    row[idx["location_region"]].(string),
			Latitude:  row[idx["location_latitude"]].(float64),
			Longitude: row[idx["location_longitude"]].(float64),
			Speed:     row[idx["location_speed"]].(float64),
		}
		if location != (EventContextLocation{}) {
			e.Context.Location = &location
		}

		// Network.
		network := EventContextNetwork{
			Cellular:  row[idx["network_cellular"]].(bool),
			WiFi:      row[idx["network_wifi"]].(bool),
			Bluetooth: row[idx["network_bluetooth"]].(bool),
			Carrier:   row[idx["network_carrier"]].(string),
		}
		if network != (EventContextNetwork{}) {
			e.Context.Network = &network
		}

		// OS.
		os := EventContextOS{
			Name:    row[idx["os_name"]].(string),
			Version: row[idx["os_version"]].(string),
		}
		if os != (EventContextOS{}) {
			e.Context.OS = &os
		}

		// Page.
		page := EventContextPage{
			URL:      row[idx["page_url"]].(string),
			Path:     row[idx["page_path"]].(string),
			Search:   row[idx["page_search"]].(string),
			Hash:     row[idx["page_hash"]].(string),
			Title:    row[idx["page_title"]].(string),
			Referrer: row[idx["page_referrer"]].(string),
		}
		if page != (EventContextPage{}) {
			e.Context.Page = &page
		}

		// Referrer.
		referrer := EventContextReferrer{
			Type: row[idx["referrer_type"]].(string),
			Name: row[idx["referrer_name"]].(string),
			URL:  row[idx["referrer_url"]].(string),
			Link: row[idx["referrer_link"]].(string),
		}
		if referrer != (EventContextReferrer{}) {
			e.Context.Referrer = &referrer
		}

		// Screen.
		screen := EventContextScreen{
			Density: row[idx["screen_density"]].(int),
			Width:   row[idx["screen_width"]].(int),
			Height:  row[idx["screen_height"]].(int),
		}
		if screen != (EventContextScreen{}) {
			e.Context.Screen = &screen
		}

		// Timezone.
		e.Context.Timezone = row[idx["timezone"]].(string)

		// UserAgent.
		e.Context.UserAgent = row[idx["user_agent"]].(string)

		e.Event = row[idx["event"]].(string)
		e.MessageID = row[idx["message_id"]].(string)
		e.Timestamp = row[idx["timestamp"]].(time.Time).Format(time.RFC3339)

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
	if ws.Warehouse == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Read the schema.
	var properties []types.Property
	if typ, ok := ws.Schemas["users"]; ok {
		properties = typ.Properties()
	} else {
		return nil, errors.Unprocessable(NoUsersSchema, "workspace %d does not have users schema", this.workspace.ID)
	}

	columns := columnsOfProperties(properties)
	where := warehouses.NewBaseExpr(
		warehouses.ExprColumn{Name: "id", Type: types.PtInt},
		warehouses.OperatorEqual,
		this.id,
	)
	rows, err := ws.Warehouse.Select(context.Background(), "users", columns, where, types.Property{}, 0, 1)
	if err != nil {
		if err2, ok := err.(*warehouses.Error); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			log.Printf("[error] cannot get a user from the data warehouse of the workspace %d: %s", ws.ID, err)
			err = errors.Unprocessable(WarehouseFailed, "warehouse connection is failed: %w", err2.Err)
		}
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.NotFound("user %d does not exist", this.id)
	}

	traits, _ := deserializeDataWarehouseRowAsMap(properties, rows[0])

	return traits, nil
}

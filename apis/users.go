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
	"fmt"
	"log/slog"
	"time"

	"chichi/apis/datastore"
	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/errors"
	"chichi/apis/events"
	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/shopspring/decimal"
)

// User represents a user.
type User struct {
	apis      *APIs
	workspace *state.Workspace
	store     *datastore.Store
	id        int
}

// Event represents a user event.
type Event struct {
	AnonymousId string `json:"anonymousId,omitempty"`
	Category    string `json:"category,omitempty"`
	Context     struct {
		App       *EventContextApp      `json:"app,omitempty"`
		Browser   *EventContextBrowser  `json:"browser,omitempty"`
		Campaign  *EventContextCampaign `json:"campaign,omitempty"`
		Device    *EventContextDevice   `json:"device,omitempty"`
		IP        string                `json:"ip,omitempty"`
		Library   *EventContextLibrary  `json:"library,omitempty"`
		Locale    string                `json:"locale,omitempty"`
		Location  *EventContextLocation `json:"location,omitempty"`
		Network   *EventContextNetwork  `json:"network,omitempty"`
		OS        *EventContextOS       `json:"os,omitempty"`
		Page      *EventContextPage     `json:"page,omitempty"`
		Referrer  *EventContextReferrer `json:"referrer,omitempty"`
		Screen    *EventContextScreen   `json:"screen,omitempty"`
		Session   *EventContextSession  `json:"session,omitempty"`
		Timezone  string                `json:"timezone,omitempty"`
		UserAgent string                `json:"userAgent,omitempty"`
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
	AdTrackingEnabled bool   `json:"adTrackingEnabled"`
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
	Width   int             `json:"width"`
	Height  int             `json:"height"`
	Density decimal.Decimal `json:"density"`
}

type EventContextSession struct {
	Id    int64 `json:"id"`
	Start bool  `json:"start"`
}

// Events returns the events of the user. limit is the maximum number of events
// to return, it must be in range [1, 200].
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - NoEventsSchema, if the data warehouse does not have events schema.
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *User) Events(ctx context.Context, limit int) ([]Event, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Build the "where" expression.
	gid := types.Property{Name: "gid", Type: types.Int(32)}
	where := whereExpr(gid, this.id)
	if where == nil {
		return nil, errors.New("unexpected nil where")
	}

	// Determine the property paths to select.
	var toSelect []types.Path
	for _, p := range events.Schema.PropertiesNames() {
		toSelect = append(toSelect, types.Path{p})
	}

	// Determine the schema of the events, which should include the ID, as such.
	// property is referenced in the "where".
	var schema types.Type
	{
		props := events.Schema.Properties()
		props = append([]types.Property{gid}, props...)
		schema = types.Object(props)
	}

	// Retrieve the events records.
	records, err := this.store.Events(ctx, schema, toSelect, where, types.Property{}, 0, limit)
	if err != nil {
		return nil, err
	}

	events := []Event{}
	err = records.For(func(r warehouses.Record) error {

		if r.Err != nil {
			return err
		}

		var e Event

		e.AnonymousId = r.Properties["anonymousId"].(string)

		e.Category = r.Properties["category"].(string)

		context := r.Properties["context"].(map[string]any)

		// App.
		{
			app := context["app"].(map[string]any)
			a := EventContextApp{
				Name:      app["name"].(string),
				Version:   app["version"].(string),
				Build:     app["build"].(string),
				Namespace: app["namespace"].(string),
			}
			if a != (EventContextApp{}) {
				e.Context.App = &a
			}
		}

		// Browser.
		{
			browser := context["browser"].(map[string]any)
			b := EventContextBrowser{
				Name:    browser["name"].(string),
				Other:   browser["other"].(string),
				Version: browser["version"].(string),
			}
			if b != (EventContextBrowser{}) {
				e.Context.Browser = &b
			}
		}

		// Campaign.
		{
			campaign := context["campaign"].(map[string]any)
			c := EventContextCampaign{
				Name:    campaign["name"].(string),
				Source:  campaign["source"].(string),
				Medium:  campaign["medium"].(string),
				Term:    campaign["term"].(string),
				Content: campaign["content"].(string),
			}
			if c != (EventContextCampaign{}) {
				e.Context.Campaign = &c
			}
		}

		// Device.
		{
			device := context["device"].(map[string]any)
			d := EventContextDevice{
				Id:                device["id"].(string),
				AdvertisingId:     device["advertisingId"].(string),
				AdTrackingEnabled: device["adTrackingEnabled"].(bool),
				Manufacturer:      device["manufacturer"].(string),
				Model:             device["model"].(string),
				Name:              device["name"].(string),
				Type:              device["type"].(string),
				Token:             device["token"].(string),
			}
			if d != (EventContextDevice{}) {
				e.Context.Device = &d
			}
		}

		// IP.
		e.Context.IP = context["ip"].(string)

		// Library.
		{
			library := context["library"].(map[string]any)
			l := EventContextLibrary{
				Name:    library["name"].(string),
				Version: library["version"].(string),
			}
			if l != (EventContextLibrary{}) {
				e.Context.Library = &l
			}
		}

		// Locale.
		e.Context.Locale = context["locale"].(string)

		// Location.
		{
			location := context["location"].(map[string]any)
			l := EventContextLocation{
				City:      location["city"].(string),
				Country:   location["country"].(string),
				Latitude:  location["latitude"].(float64),
				Longitude: location["longitude"].(float64),
				Speed:     location["speed"].(float64),
			}
			if l != (EventContextLocation{}) {
				e.Context.Location = &l
			}
		}

		// Network.
		{
			network := context["network"].(map[string]any)
			n := EventContextNetwork{
				Bluetooth: network["bluetooth"].(bool),
				Carrier:   network["carrier"].(string),
				Cellular:  network["cellular"].(bool),
				WiFi:      network["wifi"].(bool),
			}
			if n != (EventContextNetwork{}) {
				e.Context.Network = &n
			}
		}

		// OS.
		{
			os := context["os"].(map[string]any)
			o := EventContextOS{
				Name:    os["name"].(string),
				Version: os["version"].(string),
			}
			if o != (EventContextOS{}) {
				e.Context.OS = &o
			}
		}

		// Page.
		{
			page := context["page"].(map[string]any)
			p := EventContextPage{
				Path:     page["path"].(string),
				Referrer: page["referrer"].(string),
				Search:   page["search"].(string),
				Title:    page["title"].(string),
				URL:      page["url"].(string),
			}
			if p != (EventContextPage{}) {
				e.Context.Page = &p
			}
		}

		// Referrer.
		{
			referrer := context["referrer"].(map[string]any)
			r := EventContextReferrer{
				Id:   referrer["id"].(string),
				Type: referrer["type"].(string),
			}
			if r != (EventContextReferrer{}) {
				e.Context.Referrer = &r
			}
		}

		// Screen.
		{
			screen := context["screen"].(map[string]any)
			s := EventContextScreen{
				Width:   screen["width"].(int),
				Height:  screen["height"].(int),
				Density: screen["density"].(decimal.Decimal),
			}
			if s != (EventContextScreen{}) {
				e.Context.Screen = &s
			}
		}

		// Session.
		{
			session := context["session"].(map[string]any)
			s := EventContextSession{
				Id:    int64(session["id"].(int)),
				Start: session["start"].(bool),
			}
			if s != (EventContextSession{}) {
				e.Context.Session = &s
			}
		}

		e.Context.Timezone = context["timezone"].(string)
		e.Context.UserAgent = context["userAgent"].(string)

		e.Event = r.Properties["event"].(string)
		e.GroupId = r.Properties["groupId"].(string)
		e.MessageId = r.Properties["messageId"].(string)
		e.Name = r.Properties["name"].(string)
		// TODO(Gianluca): this is a temporary workaround for
		// the issue https://github.com/open2b/chichi/issues/403.
		// e.Properties = json.RawMessage(r.Properties["properties"].(string))
		e.ReceivedAt = r.Properties["receivedAt"].(time.Time).Format(time.RFC3339)
		e.SentAt = r.Properties["sentAt"].(time.Time).Format(time.RFC3339)
		e.Source = r.Properties["source"].(int)
		e.Timestamp = r.Properties["timestamp"].(time.Time).Format(time.RFC3339)
		// TODO(Gianluca): this is a temporary workaround for
		// the issue https://github.com/open2b/chichi/issues/403.
		// e.Traits = json.RawMessage(r.Properties["traits"].(string))
		e.Type = r.Properties["type"].(string)
		e.UserId = r.Properties["userId"].(string)

		events = append(events, e)
		return nil

	})
	if err != nil {
		return nil, err
	}
	if err = records.Err(); err != nil {
		return nil, fmt.Errorf("an error occurred closing the database: %s", err)
	}

	return events, nil
}

// Traits returns the traits of the user.
//
// It returns an errors.NotFoundError error, if the user does not exist.
// It returns an errors.UnprocessableError error with code
//
//   - NoWarehouse, if the workspace does not have a data warehouse.
//   - DataWarehouseFailed, if an error occurred with the data warehouse.
func (this *User) Traits(ctx context.Context) (map[string]any, error) {

	this.apis.mustBeOpen()

	ws := this.workspace

	// Verify that the workspace has a data warehouse.
	if this.store == nil {
		return nil, errors.Unprocessable(NoWarehouse, "workspace %d does not have a data warehouse", ws.ID)
	}

	// Build the "where" expression.
	id := types.Property{Name: "id", Type: types.Int(32)}
	where := whereExpr(id, this.id)
	if where == nil {
		return nil, errors.New("unexpected nil where")
	}

	// Retrieve the user traits as records.
	records, err := this.store.Users(ctx, types.Type{}, nil, where, types.Property{}, 0, 1)
	if err != nil {
		if err, ok := err.(*datastore.DataWarehouseError); ok {
			// TODO(marco): log the error in a log specific of the workspace.
			slog.Error("cannot get users from the data store", "workspace", ws.ID, "err", err)
			return nil, errors.Unprocessable(DataWarehouseFailed, "store connection is failed: %w", err.Err)
		}
		return nil, err
	}
	var traits map[string]any
	err = records.For(func(user warehouses.Record) error {
		if user.Err != nil {
			return err
		}
		traits = user.Properties
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err = records.Err(); err != nil {
		return nil, err
	}
	if traits == nil {
		return nil, errors.NotFound("user %d does not exist", this.id)
	}

	return traits, nil
}

func whereExpr(property types.Property, value int) *expr.BaseExpr {
	where := expr.NewBaseExpr(property.Name, expr.OperatorEqual, nil)
	switch property.Type.Kind() {
	case types.IntKind:
		where.Value = value
	case types.DecimalKind:
		where.Value = decimal.NewFromInt(int64(value))
	default:
		return nil
	}
	return where
}

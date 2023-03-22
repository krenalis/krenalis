//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package googleanalytics4

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the AppConnection interface.
var _ connector.AppConnection = &connection{}

func init() {
	connector.RegisterApp(connector.App{
		Name: "Google Analytics 4",
		Icon: icon,
		Open: open,
	})
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

type settings struct {
	MeasurementID string
	APISecret     string
}

// open opens a Google Analytics 4 connection and returns it.
func open(ctx context.Context, conf *connector.AppConfig) (connector.AppConnection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of Google Analytics 4 connection")
		}
	}
	return &c, nil
}

const (
	addToCart = iota + 1
)

// ActionTypes returns the connection's action types.
func (c *connection) ActionTypes() ([]*connector.ActionType, error) {
	actionTypes := []*connector.ActionType{
		{
			ID:          addToCart,
			Name:        "Add to cart",
			Description: "Send an Add to Cart event to Google Analytics 4",
			Endpoints:   []int{1},
			Schema: types.Object([]types.Property{
				{Name: "value", Type: types.Float()},
			}),
		},
	}
	return actionTypes, nil
}

// Groups returns the groups starting from the given cursor.
func (c *connection) Groups(cursor string, properties []connector.PropertyPath) error {
	return nil
}

// ReceiveWebhook receives a webhook request and returns its events.
// It returns the ErrWebhookUnauthorized error is the request was not authorized.
func (c *connection) ReceiveWebhook(r *http.Request) ([]connector.Event, error) {
	return nil, connector.ErrWebhookUnauthorized
}

// Resource returns the resource from a client token.
func (c *connection) Resource() (string, error) {
	return "", nil
}

// Schemas returns user and group schemas.
func (c *connection) Schemas() (types.Type, types.Type, error) {
	return types.Type{}, types.Type{}, nil
}

// SetUsers sets the users.
func (c *connection) SetUsers(users []connector.User) error {
	return errors.New("not implemented")
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the Form.
		var s settings
		if c.settings != nil {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		// Save the settings.
		s, err := c.SettingsUI(values)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, c.firehose.SetSettings(s)
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "MeasurementID", Label: "Measurement ID", Placeholder: "G-2XYZBEB6AB", Type: "text"},
			&ui.Input{Name: "APISecret", Label: "API Secret", Placeholder: "ZuHCHFZbRBi8V7u8crWFUz", Type: "text"},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil

}

// SettingsUI obtains the settings from UI values and returns them.
func (c *connection) SettingsUI(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// TODO(Gianluca): validate settings.
	return json.Marshal(&s)
}

// Users returns the users starting from the given cursor.
func (c *connection) Users(cursor string, properties []connector.PropertyPath) error {
	panic("not implemented")
}

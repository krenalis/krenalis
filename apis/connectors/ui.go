//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/apis/state"
)

// ServeActionUI serves the user interface of the provided file action and
// returns the new serialized interface to be sent back to the client. event is
// the event to be served, and values are the user-entered values.
//
// It returns the ErrUIEventNotExist error if the event does not exist.
// It returns an InvalidUIValuesError error value if the values are not valid.
// It panics if the connector has no UI.
func (connectors *Connectors) ServeActionUI(ctx context.Context, action *state.Action, event string, values []byte) ([]byte, error) {
	role := meergo.Role(action.Connection().Role)
	c := action.Connector()
	inner, err := meergo.RegisteredFile(c.Name).New(&meergo.FileConfig{
		Settings:    action.Settings,
		SetSettings: setActionSettingsFunc(connectors.state, action),
	})
	if err != nil {
		return nil, err
	}
	ui, err := inner.(meergo.UIHandler).ServeUI(ctx, event, values, role)
	if err != nil {
		return nil, err
	}
	return marshalUI(ui, role)
}

// ServeConnectionUI serves the user interface of the provided connection and
// returns the new serialized interface to be sent back to the client. event
// is the event to be served, and values are the user-entered values.
//
// It returns the ErrUIEventNotExist error if the event does not exist.
// It returns an InvalidUIValuesError error value if the values are not valid.
// It panics if the connector has no UI.
func (connectors *Connectors) ServeConnectionUI(ctx context.Context, connection *state.Connection, event string, values []byte) ([]byte, error) {
	var accountID int
	var accountCode string
	if r, ok := connection.Account(); ok {
		accountID = r.ID
		accountCode = r.Code
	}
	var inner any
	var err error
	switch c := connection.Connector(); c.Type {
	case state.App:
		inner, err = meergo.RegisteredApp(c.Name).New(&meergo.AppConfig{
			Settings:     connection.Settings,
			SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
			OAuthAccount: accountCode,
			HTTPClient:   connectors.http.ConnectionClient(connection.ID),
			Region:       meergo.PrivacyRegion(connection.Workspace().PrivacyRegion),
			WebhookURL:   webhookURL(connection, accountID)})
	case state.Database:
		var database meergo.Database
		database, err = meergo.RegisteredDatabase(c.Name).New(&meergo.DatabaseConfig{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
		defer database.Close()
		inner = database
	case state.FileStorage:
		inner, err = meergo.RegisteredFileStorage(c.Name).New(&meergo.FileStorageConfig{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.Mobile:
		inner, err = meergo.RegisteredMobile(c.Name).New(&meergo.MobileConfig{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.Server:
		inner, err = meergo.RegisteredServer(c.Name).New(&meergo.ServerConfig{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.Stream:
		inner, err = meergo.RegisteredStream(c.Name).New(&meergo.StreamConfig{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.Website:
		inner, err = meergo.RegisteredWebsite(c.Name).New(&meergo.WebsiteConfig{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	}
	if err != nil {
		return nil, err
	}
	ui, err := inner.(meergo.UIHandler).ServeUI(ctx, event, values, meergo.Role(connection.Role))
	if err != nil {
		return nil, err
	}
	return marshalUI(ui, meergo.Role(connection.Role))
}

type ConnectorConfig struct {
	Role   state.Role
	Region state.PrivacyRegion
	OAuth  struct {
		Account      string
		ClientSecret string
		AccessToken  string
	}
}

// ServeConnectorUI serves the user interface of the provided connector and
// returns the new serialized interface to be sent back to the client. event
// is the event to be served, and values are the user-entered values.
//
// It returns the ErrUIEventNotExist error if the event does not exist.
// It returns an InvalidUIValuesError error value if the values are not valid.
// It panics if the connector has no UI.
func (connectors *Connectors) ServeConnectorUI(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, event string, values []byte) ([]byte, error) {
	var inner any
	var err error
	switch c := connector; c.Type {
	case state.App:
		inner, err = meergo.RegisteredApp(c.Name).New(&meergo.AppConfig{
			OAuthAccount: conf.OAuth.Account,
			HTTPClient:   connectors.http.Client(conf.OAuth.ClientSecret, conf.OAuth.AccessToken, c.BackoffPolicy),
			Region:       meergo.PrivacyRegion(conf.Region),
		})
	case state.Database:
		var database meergo.Database
		database, err = meergo.RegisteredDatabase(c.Name).New(&meergo.DatabaseConfig{})
		defer database.Close()
		inner = database
	case state.File:
		inner, err = meergo.RegisteredFile(c.Name).New(&meergo.FileConfig{})
	case state.FileStorage:
		inner, err = meergo.RegisteredFileStorage(c.Name).New(&meergo.FileStorageConfig{})
	case state.Mobile:
		inner, err = meergo.RegisteredMobile(c.Name).New(&meergo.MobileConfig{})
	case state.Server:
		inner, err = meergo.RegisteredServer(c.Name).New(&meergo.ServerConfig{})
	case state.Stream:
		inner, err = meergo.RegisteredStream(c.Name).New(&meergo.StreamConfig{})
	case state.Website:
		inner, err = meergo.RegisteredWebsite(c.Name).New(&meergo.WebsiteConfig{})
	}
	if err != nil {
		return nil, err
	}
	ui, err := inner.(meergo.UIHandler).ServeUI(ctx, event, values, meergo.Role(conf.Role))
	if err != nil {
		return nil, err
	}
	return marshalUI(ui, meergo.Role(conf.Role))
}

// UpdatedSettings returns the settings, for the given connector, updated with
// the provided user-entered values.
//
// It returns an InvalidUIValuesError error value if the values are not valid.
// It panics if the connector has no UI.
func (connectors *Connectors) UpdatedSettings(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, uiValues []byte) ([]byte, error) {
	var inner any
	var err error
	var newSettings []byte
	setSettings := func(_ context.Context, settings []byte) error {
		if !utf8.Valid(settings) {
			return errors.New("settings is not valid UTF-8")
		}
		if len(settings) > maxSettingsLen && utf8.RuneCount(settings) > maxSettingsLen {
			return fmt.Errorf("settings is longer than %d runes", maxSettingsLen)
		}
		newSettings = settings
		return nil
	}
	switch c := connector; c.Type {
	case state.App:
		inner, err = meergo.RegisteredApp(c.Name).New(&meergo.AppConfig{
			OAuthAccount: conf.OAuth.Account,
			HTTPClient:   connectors.http.Client(conf.OAuth.ClientSecret, conf.OAuth.AccessToken, c.BackoffPolicy),
			SetSettings:  setSettings,
		})
	case state.Database:
		var database meergo.Database
		database, err = meergo.RegisteredDatabase(c.Name).New(&meergo.DatabaseConfig{SetSettings: setSettings})
		defer database.Close()
		inner = database
	case state.File:
		inner, err = meergo.RegisteredFile(c.Name).New(&meergo.FileConfig{SetSettings: setSettings})
	case state.Mobile:
		inner, err = meergo.RegisteredMobile(c.Name).New(&meergo.MobileConfig{SetSettings: setSettings})
	case state.Server:
		inner, err = meergo.RegisteredServer(c.Name).New(&meergo.ServerConfig{SetSettings: setSettings})
	case state.FileStorage:
		inner, err = meergo.RegisteredFileStorage(c.Name).New(&meergo.FileStorageConfig{SetSettings: setSettings})
	case state.Stream:
		inner, err = meergo.RegisteredStream(c.Name).New(&meergo.StreamConfig{SetSettings: setSettings})
	case state.Website:
		inner, err = meergo.RegisteredWebsite(c.Name).New(&meergo.WebsiteConfig{SetSettings: setSettings})
	}
	if err != nil {
		return nil, err
	}
	_, err = inner.(meergo.UIHandler).ServeUI(ctx, "save", uiValues, meergo.Role(conf.Role))
	if err != nil {
		return nil, err
	}
	return newSettings, nil
}

// marshalUI marshals the provided UI, in the given role, into JSON format.
// If ui is nil, it is serialized as "null".
func marshalUI(ui *meergo.UI, role meergo.Role) ([]byte, error) {

	if ui == nil {
		return []byte("null"), nil
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	b.WriteString("{")

	// Serialize the alert, if present.
	if ui.Alert != nil {
		b.WriteString(`"Alert":{"Message":`)
		err := enc.Encode(ui.Alert.Message)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"Variant":"`)
		b.WriteString(ui.Alert.Variant.String())
		b.WriteString(`"`)
		b.WriteString("}")
	}

	// Serialize the fields, if present.
	if ui.Fields != nil {

		if ui.Alert != nil {
			b.WriteString(",")
		}

		values := map[string]any{}
		if len(ui.Values) > 0 {
			err := json.Unmarshal(ui.Values, &values)
			if err != nil {
				return nil, err
			}
		}

		comma := false
		b.WriteString(`"Fields":[`)
		for _, field := range ui.Fields {
			ok, err := marshalUIComponent(&b, field, role, values, comma)
			if err != nil {
				return nil, err
			}
			if ok {
				comma = true
			}
		}
		b.WriteString(`],"Buttons":`)
		err := enc.Encode(ui.Buttons)
		if err != nil {
			return nil, err
		}
		if len(ui.Values) > 0 {
			b.WriteString(`,"Values":`)
			err = json.NewEncoder(&b).Encode(values)
			if err != nil {
				return nil, err
			}
		}

	}

	b.WriteString(`}`)

	return b.Bytes(), nil
}

// marshalUIComponent marshals component with the provided role in JSON format.
// If comma is true, it prepends a comma. Returns whether it has been marshaled.
func marshalUIComponent(b *bytes.Buffer, component meergo.Component, role meergo.Role, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if r := meergo.Role(rv.FieldByName("Role").Int()); r != meergo.Both && r != role {
		return false, nil
	}
	if comma {
		b.WriteString(`,`)
	}
	b.WriteString(`{"ComponentType":"`)
	b.WriteString(rt.Name())
	b.WriteString(`"`)
	for j := 0; j < rt.NumField(); j++ {
		name := rt.Field(j).Name
		if name == "Role" {
			continue
		}
		field := rv.Field(j)
		b.WriteString(`,"`)
		b.WriteString(name)
		b.WriteString(`":`)
		var err error
		switch field := field.Interface().(type) {
		case meergo.Component:
			_, err = marshalUIComponent(b, field, role, values, false)
		case []meergo.FieldSet:
			b.WriteByte('[')
			comma = false
			for _, set := range field {
				var ok bool
				ok, err = marshalUIFieldSet(b, set, role, values, comma)
				if ok {
					comma = true
				}
			}
			b.WriteByte(']')
		default:
			err = json.NewEncoder(b).Encode(field)
		}
		if err != nil {
			return false, err
		}
	}
	b.WriteString(`}`)
	return true, nil
}

// marshalUIFieldSet marshals fieldSet with the provided role in JSON format. If
// comma is true, it prepends a comma. Returns whether it has been marshaled.
func marshalUIFieldSet(b *bytes.Buffer, fieldSet meergo.FieldSet, role meergo.Role, values map[string]any, comma bool) (bool, error) {
	if fieldSet.Role != meergo.Both && fieldSet.Role != role {
		return false, nil
	}
	if comma {
		b.WriteByte(',')
	}
	b.WriteString(`{"Name":`)
	_ = json.NewEncoder(b).Encode(fieldSet.Name)
	b.WriteString(`,"Label":`)
	_ = json.NewEncoder(b).Encode(fieldSet.Label)
	b.WriteString(`,"Fields":[`)
	comma = false
	for _, c := range fieldSet.Fields {
		var valuesOfSet map[string]any
		switch vs := values[fieldSet.Name].(type) {
		case nil:
		case map[string]any:
			valuesOfSet = vs
		default:
			return false, fmt.Errorf("expected a map[string]any value for field set %s, got %T", fieldSet.Name, vs)
		}
		ok, err := marshalUIComponent(b, c, role, valuesOfSet, comma)
		if err != nil {
			return false, err
		}
		if ok {
			comma = true
		}
	}
	b.WriteString(`]}`)
	return true, nil
}

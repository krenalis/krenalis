//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/json"
)

type uiHandlerConnector interface {
	// ServeUI serves the connector's user interface. event is the event to be
	// served, settings are the connector's settings, and role is the
	// connection's role, it can be Source or Destination.
	//
	// The first time ServeUI is called to display the UI, event is "load" and
	// settings is nil. The connector saves the settings only when serving the
	// "save" event; for other events, it returns an updated interface without
	// saving the settings.
	//
	// If event does not exist, it returns an ErrUIEventNotExist.
	// If the settings are invalid, it returns an InvalidSettingsError error.
	ServeUI(ctx context.Context, event string, settings json.Value, role meergo.Role) (*meergo.UI, error)
}

// ServeActionUI serves the user interface of the specified file action and
// returns the serialized interface to send back to the client. event indicates
// the event to serve and settings are the format settings.
//
// It returns the ErrUIEventNotExist error if the event does not exist, an
// *InvalidSettingsError error if the settings are not valid, and an
// *UnavailableError error if the connector returns an error.
//
// It panics if the connector has no settings.
func (connectors *Connectors) ServeActionUI(ctx context.Context, action *state.Action, event string, settings json.Value) (json.Value, error) {
	role := meergo.Role(action.Connection().Role)
	format := action.Format()
	inner, err := meergo.RegisteredFile(format.Name).New(&meergo.FileEnv{
		Settings:    action.FormatSettings,
		SetSettings: setActionSettingsFunc(connectors.state, action),
	})
	if err != nil {
		return nil, err
	}
	ui, err := inner.(uiHandlerConnector).ServeUI(ctx, event, settings, role)
	if err != nil {
		return nil, connectorError(err)
	}
	return marshalUI(ui, role)
}

// ServeConnectionUI serves the user interface of the provided connection and
// returns the serialized interface to send back to the client. event specifies
// the event to serve, and settings are the connection settings.
//
// It returns the ErrUIEventNotExist error if the event does not exist, an
// *InvalidSettingsError error if the settings are not valid, and an
// *UnavailableError error if the connector returns an error.
//
// It panics if the connector has no settings.
func (connectors *Connectors) ServeConnectionUI(ctx context.Context, connection *state.Connection, event string, settings json.Value) (json.Value, error) {
	// var accountID int TODO(marco): implement webhooks
	var accountCode string
	if r, ok := connection.Account(); ok {
		// accountID = r.ID TODO(marco): implement webhooks
		accountCode = r.Code
	}
	var inner any
	var err error
	switch c := connection.Connector(); c.Type {
	case state.App:
		inner, err = meergo.RegisteredApp(c.Name).New(&meergo.AppEnv{
			Settings:     connection.Settings,
			SetSettings:  setConnectionSettingsFunc(connectors.state, connection),
			OAuthAccount: accountCode,
			HTTPClient:   connectors.http.ConnectionClient(connection),
			// WebhookURL:   webhookURL(connection, accountID) // TODO(marco): implement webhooks
		})
	case state.Database:
		var database any
		database, err = meergo.RegisteredDatabase(c.Name).New(&meergo.DatabaseEnv{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
		defer database.(databaseConnector).Close()
		inner = database
	case state.FileStorage:
		inner, err = meergo.RegisteredFileStorage(c.Name).New(&meergo.FileStorageEnv{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.SDK:
		inner, err = meergo.RegisteredSDK(c.Name).New(&meergo.SDKEnv{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.Stream:
		inner, err = meergo.RegisteredStream(c.Name).New(&meergo.StreamEnv{
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	}
	if err != nil {
		return nil, err
	}
	ui, err := inner.(uiHandlerConnector).ServeUI(ctx, event, settings, meergo.Role(connection.Role))
	if err != nil {
		return nil, connectorError(err)
	}
	return marshalUI(ui, meergo.Role(connection.Role))
}

type ConnectorConfig struct {
	Role  state.Role
	OAuth struct {
		Account      string
		ClientSecret string
		AccessToken  string
	}
}

// ServeConnectorUI serves the user interface of the provided connector and
// returns the serialized interface to send back to the client. event specifies
// the event to serve, and settings are the connector settings.
//
// It returns the ErrUIEventNotExist error if the event does not exist, an
// *InvalidSettingsError error if the settings are not valid, and an
// *UnavailableError error if the connector returns an error.
//
// It panics if the connector has no settings.
func (connectors *Connectors) ServeConnectorUI(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, event string, settings json.Value) (json.Value, error) {
	var inner any
	var err error
	switch c := connector; c.Type {
	case state.App:
		inner, err = meergo.RegisteredApp(c.Name).New(&meergo.AppEnv{
			OAuthAccount: conf.OAuth.Account,
			HTTPClient:   connectors.http.ConnectorClient(c, conf.OAuth.ClientSecret, conf.OAuth.AccessToken),
		})
	case state.Database:
		var database any
		database, err = meergo.RegisteredDatabase(c.Name).New(&meergo.DatabaseEnv{})
		defer database.(databaseConnector).Close()
		inner = database
	case state.File:
		inner, err = meergo.RegisteredFile(c.Name).New(&meergo.FileEnv{})
	case state.FileStorage:
		inner, err = meergo.RegisteredFileStorage(c.Name).New(&meergo.FileStorageEnv{})
	case state.SDK:
		inner, err = meergo.RegisteredSDK(c.Name).New(&meergo.SDKEnv{})
	case state.Stream:
		inner, err = meergo.RegisteredStream(c.Name).New(&meergo.StreamEnv{})
	}
	if err != nil {
		return nil, err
	}
	ui, err := inner.(uiHandlerConnector).ServeUI(ctx, event, settings, meergo.Role(conf.Role))
	if err != nil {
		return nil, connectorError(err)
	}
	return marshalUI(ui, meergo.Role(conf.Role))
}

// UpdatedSettings returns the inner settings, for the given connector, updated
// with the provided settings.
//
// It returns an *InvalidSettingsError error value if the settings are not valid
// and an *UnavailableError error if the connector returns an error.
//
// It panics if the connector has no settings.
func (connectors *Connectors) UpdatedSettings(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, settings json.Value) ([]byte, error) {
	var inner any
	var err error
	var updatedSettings []byte
	setSettings := func(_ context.Context, innerSettings []byte) error {
		if !utf8.Valid(innerSettings) {
			return errors.New("inner settings is not valid UTF-8")
		}
		if len(innerSettings) > maxSettingsLen && utf8.RuneCount(innerSettings) > maxSettingsLen {
			return fmt.Errorf("inner settings is longer than %d runes", maxSettingsLen)
		}
		updatedSettings = innerSettings
		return nil
	}
	switch c := connector; c.Type {
	case state.App:
		inner, err = meergo.RegisteredApp(c.Name).New(&meergo.AppEnv{
			OAuthAccount: conf.OAuth.Account,
			HTTPClient:   connectors.http.ConnectorClient(c, conf.OAuth.ClientSecret, conf.OAuth.AccessToken),
			SetSettings:  setSettings,
		})
	case state.Database:
		var database any
		database, err = meergo.RegisteredDatabase(c.Name).New(&meergo.DatabaseEnv{SetSettings: setSettings})
		defer database.(databaseConnector).Close()
		inner = database
	case state.File:
		inner, err = meergo.RegisteredFile(c.Name).New(&meergo.FileEnv{SetSettings: setSettings})
	case state.SDK:
		inner, err = meergo.RegisteredSDK(c.Name).New(&meergo.SDKEnv{SetSettings: setSettings})
	case state.FileStorage:
		inner, err = meergo.RegisteredFileStorage(c.Name).New(&meergo.FileStorageEnv{SetSettings: setSettings})
	case state.Stream:
		inner, err = meergo.RegisteredStream(c.Name).New(&meergo.StreamEnv{SetSettings: setSettings})
	}
	if err != nil {
		return nil, err
	}
	_, err = inner.(uiHandlerConnector).ServeUI(ctx, "save", settings, meergo.Role(conf.Role))
	if err != nil {
		return nil, connectorError(err)
	}
	return updatedSettings, nil
}

// marshalUI marshals the provided UI, in the given role, into JSON format.
// If ui is nil, it is serialized as "null".
func marshalUI(ui *meergo.UI, role meergo.Role) (json.Value, error) {

	if ui == nil {
		return []byte("null"), nil
	}

	var b json.Buffer
	b.WriteString("{")

	// Serialize the alert, if present.
	if ui.Alert != nil {
		b.WriteString(`"alert":{`)
		err := b.EncodeKeyValue("message", ui.Alert.Message)
		if err != nil {
			return nil, err
		}
		_ = b.EncodeKeyValue("variant", ui.Alert.Variant.String())
		b.WriteString("}")
	}

	// Serialize the fields, if present.
	if ui.Fields != nil {

		if ui.Alert != nil {
			b.WriteString(",")
		}

		settings := map[string]any{}
		if len(ui.Settings) > 0 {
			err := json.Unmarshal(ui.Settings, &settings)
			if err != nil {
				return nil, err
			}
		}

		comma := false
		b.WriteString(`"fields":[`)
		for _, field := range ui.Fields {
			ok, err := marshalUIComponent(&b, field, role, settings, comma)
			if err != nil {
				return nil, err
			}
			if ok {
				comma = true
			}
		}
		b.WriteString(`],"buttons":[`)
		for _, button := range ui.Buttons {
			b.WriteByte('{')
			_ = b.EncodeKeyValue("event", button.Event)
			_ = b.EncodeKeyValue("text", button.Text)
			_ = b.EncodeKeyValue("variant", button.Variant)
			_ = b.EncodeKeyValue("role", button.Role.String())
			b.WriteString(`}`)
		}
		b.WriteString(`]`)
		if len(ui.Settings) > 0 {
			b.WriteString(`,"settings":`)
			err := b.Encode(settings)
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
func marshalUIComponent(b *json.Buffer, component meergo.Component, role meergo.Role, settings map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if r := meergo.Role(rv.FieldByName("Role").Int()); r != meergo.Both && r != role {
		return false, nil
	}
	if comma {
		b.WriteString(`,`)
	}
	b.WriteString(`{"componentType":"`)
	b.WriteString(rt.Name())
	b.WriteString(`"`)
	for j := 0; j < rt.NumField(); j++ {
		name := rt.Field(j).Name
		if name == "Role" {
			continue
		}
		// Writes the field name in camelCase format.
		field := rv.Field(j)
		b.WriteString(`,"`)
		b.WriteByte(name[0] + ('a' - 'A'))
		b.WriteString(name[1:])
		b.WriteString(`":`)
		var err error
		switch field := field.Interface().(type) {
		case meergo.Component:
			_, err = marshalUIComponent(b, field, role, settings, false)
		case []meergo.FieldSet:
			b.WriteByte('[')
			comma = false
			for _, set := range field {
				var ok bool
				ok, err = marshalUIFieldSet(b, set, role, settings, comma)
				if ok {
					comma = true
				}
			}
			b.WriteByte(']')
		case []meergo.Option:
			b.WriteByte('[')
			for i, option := range field {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteByte('{')
				_ = b.EncodeKeyValue("text", option.Text)
				_ = b.EncodeKeyValue("value", option.Value)
				b.WriteString(`}`)
			}
			b.WriteByte(']')
		default:
			err = b.Encode(field)
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
func marshalUIFieldSet(b *json.Buffer, fieldSet meergo.FieldSet, role meergo.Role, settings map[string]any, comma bool) (bool, error) {
	if fieldSet.Role != meergo.Both && fieldSet.Role != role {
		return false, nil
	}
	if comma {
		b.WriteByte(',')
	}
	b.WriteByte('{')
	_ = b.EncodeKeyValue("name", fieldSet.Name)
	_ = b.EncodeKeyValue("label", fieldSet.Label)
	b.WriteString(`,"fields":[`)
	comma = false
	for _, c := range fieldSet.Fields {
		var valuesOfSet map[string]any
		switch vs := settings[fieldSet.Name].(type) {
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

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
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/ui"
)

// ServeConnectionUI serves the user interface of the provided connection and
// returns the new serialized interface to be sent back to the client. event
// is the event to be served, and settings are the user-entered settings.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns the ErrEventNotExist error if the event does not exist.
// It returns an *InvalidSettingsError error value if the settings are not
// valid.
func (connectors *Connectors) ServeConnectionUI(ctx context.Context, connection *state.Connection, event string, settings []byte) ([]byte, error) {
	var resourceID int
	var resourceCode string
	if r, ok := connection.Resource(); ok {
		resourceID = r.ID
		resourceCode = r.Code
	}
	role := _connector.Role(connection.Role)
	db := connectors.db
	var inner any
	var err error
	switch c := connection.Connector(); c.Type {
	case state.AppType:
		inner, err = _connector.RegisteredApp(c.Name).New(&_connector.AppConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
			Resource:    resourceCode,
			HTTPClient:  connectors.http.ConnectionClient(connection.ID),
			Region:      _connector.PrivacyRegion(connection.Workspace().PrivacyRegion),
			WebhookURL:  webhookURL(connection, resourceID)})
	case state.DatabaseType:
		var database _connector.DatabaseConnection
		database, err = _connector.RegisteredDatabase(c.Name).New(&_connector.DatabaseConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = _connector.RegisteredFile(c.Name).New(&_connector.FileConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
	case state.MobileType:
		inner, err = _connector.RegisteredMobile(c.Name).New(&_connector.MobileConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
	case state.ServerType:
		inner, err = _connector.RegisteredServer(c.Name).New(&_connector.ServerConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
	case state.StorageType:
		inner, err = _connector.RegisteredStorage(c.Name).New(&_connector.StorageConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
	case state.StreamType:
		inner, err = _connector.RegisteredStream(c.Name).New(&_connector.StreamConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
	case state.WebsiteType:
		inner, err = _connector.RegisteredWebsite(c.Name).New(&_connector.WebsiteConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setSettingsFunc(db, connection),
		})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(_connector.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	form, alert, err := connectorUI.ServeUI(ctx, event, settings)
	if err != nil {
		if err == ui.ErrEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(ui.Error); ok {
			return nil, &InvalidSettingsError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUIFormAlert(form, alert, ui.Role(role))
}

type ConnectorConfig struct {
	Role         state.Role
	Resource     string
	ClientSecret string
	AccessToken  string
	Region       state.PrivacyRegion
}

// ServeConnectorUI serves the user interface of the provided connector and
// returns the new serialized interface to be sent back to the client. event
// is the event to be served, and settings are the user-entered settings.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns the ErrEventNotExist error if the event does not exist.
// It returns an *InvalidSettingsError error value if the settings are not
// valid.
func (connectors *Connectors) ServeConnectorUI(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, event string, settings []byte) ([]byte, error) {
	var inner any
	var err error
	r := _connector.Role(conf.Role)
	switch c := connector; c.Type {
	case state.AppType:
		inner, err = _connector.RegisteredApp(c.Name).New(&_connector.AppConfig{
			Role:       r,
			Resource:   conf.Resource,
			HTTPClient: connectors.http.Client(conf.ClientSecret, conf.AccessToken),
			Region:     _connector.PrivacyRegion(conf.Region),
		})
	case state.DatabaseType:
		var database _connector.DatabaseConnection
		database, err = _connector.RegisteredDatabase(c.Name).New(&_connector.DatabaseConfig{Role: r})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = _connector.RegisteredFile(c.Name).New(&_connector.FileConfig{Role: r})
	case state.MobileType:
		inner, err = _connector.RegisteredMobile(c.Name).New(&_connector.MobileConfig{Role: r})
	case state.ServerType:
		inner, err = _connector.RegisteredServer(c.Name).New(&_connector.ServerConfig{Role: r})
	case state.StorageType:
		inner, err = _connector.RegisteredStorage(c.Name).New(&_connector.StorageConfig{Role: r})
	case state.StreamType:
		inner, err = _connector.RegisteredStream(c.Name).New(&_connector.StreamConfig{Role: r})
	case state.WebsiteType:
		inner, err = _connector.RegisteredWebsite(c.Name).New(&_connector.WebsiteConfig{Role: r})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(_connector.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	form, alert, err := connectorUI.ServeUI(ctx, event, settings)
	if err != nil {
		if err == ui.ErrEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(ui.Error); ok {
			return nil, &InvalidSettingsError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUIFormAlert(form, alert, ui.Role(r))
}

// ValidateSettings validates the user-entered settings for the provided
// connector and returns them validated and normalized.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns an *InvalidSettingsError error value if the settings are not
// valid.
func (connectors *Connectors) ValidateSettings(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, settings []byte) ([]byte, error) {
	var inner any
	var err error
	r := _connector.Role(conf.Role)
	switch c := connector; c.Type {
	case state.AppType:
		inner, err = _connector.RegisteredApp(c.Name).New(&_connector.AppConfig{
			Role:       r,
			Resource:   conf.Resource,
			HTTPClient: connectors.http.Client(conf.ClientSecret, conf.AccessToken),
		})
	case state.DatabaseType:
		var database _connector.DatabaseConnection
		database, err = _connector.RegisteredDatabase(c.Name).New(&_connector.DatabaseConfig{Role: r})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = _connector.RegisteredFile(c.Name).New(&_connector.FileConfig{Role: r})
	case state.MobileType:
		inner, err = _connector.RegisteredMobile(c.Name).New(&_connector.MobileConfig{Role: r})
	case state.ServerType:
		inner, err = _connector.RegisteredServer(c.Name).New(&_connector.ServerConfig{Role: r})
	case state.StorageType:
		inner, err = _connector.RegisteredStorage(c.Name).New(&_connector.StorageConfig{Role: r})
	case state.StreamType:
		inner, err = _connector.RegisteredStream(c.Name).New(&_connector.StreamConfig{Role: r})
	case state.WebsiteType:
		inner, err = _connector.RegisteredWebsite(c.Name).New(&_connector.WebsiteConfig{Role: r})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(_connector.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	settings, err = connectorUI.ValidateSettings(ctx, settings)
	if err != nil {
		if err, ok := err.(ui.Error); ok {
			return nil, &InvalidSettingsError{Msg: err.Error()}
		}
		return nil, err
	}
	if utf8.RuneCount(settings) > maxSettingsLen {
		return nil, fmt.Errorf("settings returned by %s are longer than %d runes", connector.Name, maxSettingsLen)
	}
	return settings, nil
}

// marshalUIFormAlert marshals form, with the provided alert and role, in JSON
// format. form and alert can be nil or not, independently of each other.
func marshalUIFormAlert(form *ui.Form, alert *ui.Alert, role ui.Role) ([]byte, error) {

	if form == nil && alert == nil {
		return []byte("null"), nil
	}

	var b bytes.Buffer
	enc := json.NewEncoder(&b)

	b.WriteString("{")

	// Serialize the form, if present.
	if form != nil {

		// Makes the keys of form.Values to have the same case as the Name field of the components.
		values := map[string]any{}
		if len(form.Values) > 0 {
			err := json.Unmarshal(form.Values, &values)
			if err != nil {
				return nil, err
			}
		}

		comma := false
		b.WriteString(`"Form":{"Fields":[`)
		for _, field := range form.Fields {
			ok, err := marshalUIComponent(&b, field, role, values, comma)
			if err != nil {
				return nil, err
			}
			if ok {
				comma = true
			}
		}
		b.WriteString(`],"Actions":`)
		err := enc.Encode(form.Actions)
		if err != nil {
			return nil, err
		}
		if len(form.Values) > 0 {
			b.WriteString(`,"Values":`)
			err = json.NewEncoder(&b).Encode(values)
			if err != nil {
				return nil, err
			}
		}
		b.WriteString("}")

	}

	// Serialize the alert, if present.
	if alert != nil {
		if form != nil {
			b.WriteString(",")
		}
		b.WriteString(`"Alert":{"Message":`)
		err := enc.Encode(alert.Message)
		if err != nil {
			return nil, err
		}
		b.WriteString(`,"Variant":"`)
		b.WriteString(alert.Variant.String())
		b.WriteString(`"`)
		b.WriteString("}")
	}

	b.WriteString(`}`)

	return b.Bytes(), nil
}

// marshalUIComponent marshals component with the provided role in JSON format.
// If comma is true, it prepends a comma. Returns whether it has been marshaled.
func marshalUIComponent(b *bytes.Buffer, component ui.Component, role ui.Role, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if role != ui.Both {
		if r := ui.Role(rv.FieldByName("Role").Int()); r != ui.Both && r != role {
			return false, nil
		}
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
		if name == "Name" && values != nil {
			adjustValuesCase(field.String(), values)
		}
		b.WriteString(`,"`)
		b.WriteString(name)
		b.WriteString(`":`)
		var err error
		switch field := field.Interface().(type) {
		case ui.Component:
			_, err = marshalUIComponent(b, field, role, values, false)
		case []ui.FieldSet:
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
func marshalUIFieldSet(b *bytes.Buffer, fieldSet ui.FieldSet, role ui.Role, values map[string]any, comma bool) (bool, error) {
	if role != ui.Both {
		if fieldSet.Role != ui.Both && fieldSet.Role != role {
			return false, nil
		}
	}
	name := fieldSet.Name
	if values != nil {
		adjustValuesCase(name, values)
	}
	if comma {
		b.WriteByte(',')
	}
	b.WriteString(`{"Name":`)
	_ = json.NewEncoder(b).Encode(name)
	b.WriteString(`,"Label":`)
	_ = json.NewEncoder(b).Encode(fieldSet.Label)
	b.WriteString(`,"Fields":[`)
	comma = false
	for _, c := range fieldSet.Fields {
		var valuesOfSet map[string]any
		switch vs := values[name].(type) {
		case nil:
		case map[string]any:
			valuesOfSet = vs
		default:
			return false, fmt.Errorf("expected a map[string]any value for field set %s, got %T", name, values[name])
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

// adjustValuesCase adjusts the case of keys of values.
func adjustValuesCase(key string, values map[string]any) {
	var found struct {
		key   string
		value any
	}
	for k, v := range values {
		if strings.EqualFold(k, key) {
			found.key = k
			found.value = v
			break
		}
	}
	if found.key == "" {
		return
	}
	delete(values, found.key)
	values[key] = found.value
}

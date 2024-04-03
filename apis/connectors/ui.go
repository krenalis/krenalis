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

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
)

// ServeActionUI serves the user interface of the provided file action and returns the
// new serialized interface to be sent back to the client. event is the event to be
// served, and settings are the user-entered settings.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns the ErrEventNotExist error if the event does not exist.
// It returns an *InvalidSettingsError error value if the settings are not
// valid.
func (connectors *Connectors) ServeActionUI(ctx context.Context, action *state.Action, event string, settings []byte) ([]byte, error) {
	role := chichi.Role(action.Connection().Role)
	c := action.Connector()
	inner, err := chichi.RegisteredFile(c.Name).New(&chichi.FileConfig{
		Role:        role,
		Settings:    action.Settings,
		SetSettings: setActionSettingsFunc(connectors.state, action),
	})
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(chichi.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	form, alert, err := connectorUI.ServeUI(ctx, event, settings)
	if err != nil {
		if err == chichi.ErrEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(chichi.Error); ok {
			return nil, &InvalidSettingsError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUIFormAlert(form, alert, chichi.Role(role))
}

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
	role := chichi.Role(connection.Role)
	var inner any
	var err error
	switch c := connection.Connector(); c.Type {
	case state.AppType:
		inner, err = chichi.RegisteredApp(c.Name).New(&chichi.AppConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
			Resource:    resourceCode,
			HTTPClient:  connectors.http.ConnectionClient(connection.ID),
			Region:      chichi.PrivacyRegion(connection.Workspace().PrivacyRegion),
			WebhookURL:  webhookURL(connection, resourceID)})
	case state.DatabaseType:
		var database chichi.Database
		database, err = chichi.RegisteredDatabase(c.Name).New(&chichi.DatabaseConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
		defer database.Close()
		inner = database
	case state.FileStorageType:
		inner, err = chichi.RegisteredFileStorage(c.Name).New(&chichi.FileStorageConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.MobileType:
		inner, err = chichi.RegisteredMobile(c.Name).New(&chichi.MobileConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.ServerType:
		inner, err = chichi.RegisteredServer(c.Name).New(&chichi.ServerConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.StreamType:
		inner, err = chichi.RegisteredStream(c.Name).New(&chichi.StreamConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	case state.WebsiteType:
		inner, err = chichi.RegisteredWebsite(c.Name).New(&chichi.WebsiteConfig{
			Role:        role,
			Settings:    connection.Settings,
			SetSettings: setConnectionSettingsFunc(connectors.state, connection),
		})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(chichi.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	form, alert, err := connectorUI.ServeUI(ctx, event, settings)
	if err != nil {
		if err == chichi.ErrEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(chichi.Error); ok {
			return nil, &InvalidSettingsError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUIFormAlert(form, alert, chichi.Role(role))
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
	r := chichi.Role(conf.Role)
	switch c := connector; c.Type {
	case state.AppType:
		inner, err = chichi.RegisteredApp(c.Name).New(&chichi.AppConfig{
			Role:       r,
			Resource:   conf.Resource,
			HTTPClient: connectors.http.Client(conf.ClientSecret, conf.AccessToken),
			Region:     chichi.PrivacyRegion(conf.Region),
		})
	case state.DatabaseType:
		var database chichi.Database
		database, err = chichi.RegisteredDatabase(c.Name).New(&chichi.DatabaseConfig{Role: r})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = chichi.RegisteredFile(c.Name).New(&chichi.FileConfig{Role: r})
	case state.FileStorageType:
		inner, err = chichi.RegisteredFileStorage(c.Name).New(&chichi.FileStorageConfig{Role: r})
	case state.MobileType:
		inner, err = chichi.RegisteredMobile(c.Name).New(&chichi.MobileConfig{Role: r})
	case state.ServerType:
		inner, err = chichi.RegisteredServer(c.Name).New(&chichi.ServerConfig{Role: r})
	case state.StreamType:
		inner, err = chichi.RegisteredStream(c.Name).New(&chichi.StreamConfig{Role: r})
	case state.WebsiteType:
		inner, err = chichi.RegisteredWebsite(c.Name).New(&chichi.WebsiteConfig{Role: r})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(chichi.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	form, alert, err := connectorUI.ServeUI(ctx, event, settings)
	if err != nil {
		if err == chichi.ErrEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(chichi.Error); ok {
			return nil, &InvalidSettingsError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUIFormAlert(form, alert, chichi.Role(r))
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
	r := chichi.Role(conf.Role)
	switch c := connector; c.Type {
	case state.AppType:
		inner, err = chichi.RegisteredApp(c.Name).New(&chichi.AppConfig{
			Role:       r,
			Resource:   conf.Resource,
			HTTPClient: connectors.http.Client(conf.ClientSecret, conf.AccessToken),
		})
	case state.DatabaseType:
		var database chichi.Database
		database, err = chichi.RegisteredDatabase(c.Name).New(&chichi.DatabaseConfig{Role: r})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = chichi.RegisteredFile(c.Name).New(&chichi.FileConfig{Role: r})
	case state.MobileType:
		inner, err = chichi.RegisteredMobile(c.Name).New(&chichi.MobileConfig{Role: r})
	case state.ServerType:
		inner, err = chichi.RegisteredServer(c.Name).New(&chichi.ServerConfig{Role: r})
	case state.FileStorageType:
		inner, err = chichi.RegisteredFileStorage(c.Name).New(&chichi.FileStorageConfig{Role: r})
	case state.StreamType:
		inner, err = chichi.RegisteredStream(c.Name).New(&chichi.StreamConfig{Role: r})
	case state.WebsiteType:
		inner, err = chichi.RegisteredWebsite(c.Name).New(&chichi.WebsiteConfig{Role: r})
	}
	if err != nil {
		return nil, err
	}
	connectorUI, ok := inner.(chichi.UI)
	if !ok {
		return nil, ErrNoUserInterface
	}
	settings, err = connectorUI.ValidateSettings(ctx, settings)
	if err != nil {
		if err, ok := err.(chichi.Error); ok {
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
func marshalUIFormAlert(form *chichi.Form, alert *chichi.Alert, role chichi.Role) ([]byte, error) {

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
func marshalUIComponent(b *bytes.Buffer, component chichi.Component, role chichi.Role, values map[string]any, comma bool) (bool, error) {
	rv := reflect.ValueOf(component).Elem()
	rt := rv.Type()
	if role != chichi.Both {
		if r := chichi.Role(rv.FieldByName("Role").Int()); r != chichi.Both && r != role {
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
		case chichi.Component:
			_, err = marshalUIComponent(b, field, role, values, false)
		case []chichi.FieldSet:
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
func marshalUIFieldSet(b *bytes.Buffer, fieldSet chichi.FieldSet, role chichi.Role, values map[string]any, comma bool) (bool, error) {
	if role != chichi.Both {
		if fieldSet.Role != chichi.Both && fieldSet.Role != role {
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

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
	"unicode/utf8"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
)

// ServeActionUI serves the user interface of the provided file action and
// returns the new serialized interface to be sent back to the client. event is
// the event to be served, and values are the user-entered values.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns the ErrEventNotExist error if the event does not exist.
// It returns an *InvalidUIValuesError error value if the values are not valid.
func (connectors *Connectors) ServeActionUI(ctx context.Context, action *state.Action, event string, values []byte) ([]byte, error) {
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
	uih, ok := inner.(chichi.UIHandler)
	if !ok {
		return nil, ErrNoUserInterface
	}
	ui, err := uih.ServeUI(ctx, event, values)
	if err != nil {
		if err == chichi.ErrUIEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(chichi.InvalidUIValuesError); ok {
			return nil, &InvalidUIValuesError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUI(ui, role)
}

// ServeConnectionUI serves the user interface of the provided connection and
// returns the new serialized interface to be sent back to the client. event
// is the event to be served, and values are the user-entered values.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns the ErrEventNotExist error if the event does not exist.
// It returns an *InvalidUIValuesError error value if the values are not valid.
func (connectors *Connectors) ServeConnectionUI(ctx context.Context, connection *state.Connection, event string, values []byte) ([]byte, error) {
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
	uih, ok := inner.(chichi.UIHandler)
	if !ok {
		return nil, ErrNoUserInterface
	}
	ui, err := uih.ServeUI(ctx, event, values)
	if err != nil {
		if err == chichi.ErrUIEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(chichi.InvalidUIValuesError); ok {
			return nil, &InvalidUIValuesError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUI(ui, role)
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
// is the event to be served, and values are the user-entered values.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns the ErrEventNotExist error if the event does not exist.
// It returns an *InvalidUIValuesError error value if the values are not valid.
func (connectors *Connectors) ServeConnectorUI(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, event string, values []byte) ([]byte, error) {
	var inner any
	var err error
	role := chichi.Role(conf.Role)
	switch c := connector; c.Type {
	case state.AppType:
		inner, err = chichi.RegisteredApp(c.Name).New(&chichi.AppConfig{
			Role:       role,
			Resource:   conf.Resource,
			HTTPClient: connectors.http.Client(conf.ClientSecret, conf.AccessToken),
			Region:     chichi.PrivacyRegion(conf.Region),
		})
	case state.DatabaseType:
		var database chichi.Database
		database, err = chichi.RegisteredDatabase(c.Name).New(&chichi.DatabaseConfig{Role: role})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = chichi.RegisteredFile(c.Name).New(&chichi.FileConfig{Role: role})
	case state.FileStorageType:
		inner, err = chichi.RegisteredFileStorage(c.Name).New(&chichi.FileStorageConfig{Role: role})
	case state.MobileType:
		inner, err = chichi.RegisteredMobile(c.Name).New(&chichi.MobileConfig{Role: role})
	case state.ServerType:
		inner, err = chichi.RegisteredServer(c.Name).New(&chichi.ServerConfig{Role: role})
	case state.StreamType:
		inner, err = chichi.RegisteredStream(c.Name).New(&chichi.StreamConfig{Role: role})
	case state.WebsiteType:
		inner, err = chichi.RegisteredWebsite(c.Name).New(&chichi.WebsiteConfig{Role: role})
	}
	if err != nil {
		return nil, err
	}
	uih, ok := inner.(chichi.UIHandler)
	if !ok {
		return nil, ErrNoUserInterface
	}
	ui, err := uih.ServeUI(ctx, event, values)
	if err != nil {
		if err == chichi.ErrUIEventNotExist {
			return nil, ErrEventNotExist
		}
		if err, ok := err.(chichi.InvalidUIValuesError); ok {
			return nil, &InvalidUIValuesError{Msg: err.Error()}
		}
		return nil, err
	}
	return marshalUI(ui, role)
}

// ValidateUIValues validates the user-entered values for the provided connector
// and returns the new connector's settings.
//
// It returns the ErrNoUserInterface error if the connector does not have a user
// interface.
// It returns an *InvalidUIValuesError error value if the values are not valid.
func (connectors *Connectors) ValidateUIValues(ctx context.Context, connector *state.Connector, conf *ConnectorConfig, values []byte) ([]byte, error) {
	var inner any
	var err error
	r := chichi.Role(conf.Role)
	var savedSettings []byte
	setSettings := func(_ context.Context, settings []byte) error {
		savedSettings = settings
		return nil
	}
	switch c := connector; c.Type {
	case state.AppType:
		inner, err = chichi.RegisteredApp(c.Name).New(&chichi.AppConfig{
			Role:        r,
			Resource:    conf.Resource,
			HTTPClient:  connectors.http.Client(conf.ClientSecret, conf.AccessToken),
			SetSettings: setSettings,
		})
	case state.DatabaseType:
		var database chichi.Database
		database, err = chichi.RegisteredDatabase(c.Name).New(&chichi.DatabaseConfig{Role: r, SetSettings: setSettings})
		defer database.Close()
		inner = database
	case state.FileType:
		inner, err = chichi.RegisteredFile(c.Name).New(&chichi.FileConfig{Role: r, SetSettings: setSettings})
	case state.MobileType:
		inner, err = chichi.RegisteredMobile(c.Name).New(&chichi.MobileConfig{Role: r, SetSettings: setSettings})
	case state.ServerType:
		inner, err = chichi.RegisteredServer(c.Name).New(&chichi.ServerConfig{Role: r, SetSettings: setSettings})
	case state.FileStorageType:
		inner, err = chichi.RegisteredFileStorage(c.Name).New(&chichi.FileStorageConfig{Role: r, SetSettings: setSettings})
	case state.StreamType:
		inner, err = chichi.RegisteredStream(c.Name).New(&chichi.StreamConfig{Role: r, SetSettings: setSettings})
	case state.WebsiteType:
		inner, err = chichi.RegisteredWebsite(c.Name).New(&chichi.WebsiteConfig{Role: r, SetSettings: setSettings})
	}
	if err != nil {
		return nil, err
	}
	uih, ok := inner.(chichi.UIHandler)
	if !ok {
		return nil, ErrNoUserInterface
	}
	_, err = uih.ServeUI(ctx, "save", values)
	if err != nil {
		if err, ok := err.(chichi.InvalidUIValuesError); ok {
			return nil, &InvalidUIValuesError{Msg: err.Error()}
		}
		return nil, err
	}
	if savedSettings == nil {
		return nil, fmt.Errorf("%s connector has not set the settings", connector.Name)
	}
	if utf8.RuneCount(savedSettings) > maxSettingsLen {
		return nil, fmt.Errorf("settings returned by %s are longer than %d runes", connector.Name, maxSettingsLen)
	}
	return savedSettings, nil
}

// marshalUI marshals the provided UI, in the given role, into JSON format.
// If ui is nil, it is serialized as "null".
func marshalUI(ui *chichi.UI, role chichi.Role) ([]byte, error) {

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

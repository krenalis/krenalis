//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package clickhouse implements the ClickHouse connector.
// (https://clickhouse.com/docs/)
package clickhouse

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"unicode/utf8"

	"chichi/connector"
	"chichi/connector/types"
	"chichi/connector/ui"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the UI interface.
var _ connector.UI = (*connection)(nil)

func init() {
	connector.RegisterDatabase(connector.Database{
		Name:              "ClickHouse",
		SourceDescription: "import users and groups from a ClickHouse database",
		Icon:              icon,
	}, open)
}

// open opens a ClickHouse connection and returns it.
func open(ctx context.Context, conf *connector.DatabaseConfig) (*connection, error) {
	c := connection{ctx: ctx, conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of ClickHouse connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	conf     *connector.DatabaseConfig
	settings *settings
}

// Query executes the given query and returns the resulting rows and properties.
func (c *connection) Query(query string) (connector.Rows, []types.Property, error) {
	conn, err := clickhouse.Open(c.settings.options())
	if err != nil {
		return nil, nil, err
	}
	rows, err := conn.Query(c.ctx, query)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	columnTypes := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	properties := make([]types.Property, len(columnTypes))
	for i, c := range columnTypes {
		typ, nullable, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
			_ = conn.Close()
			return nil, nil, err
		}
		properties[i] = types.Property{
			Name:     c.Name(),
			Type:     typ,
			Nullable: nullable,
		}
	}
	return rows, properties, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the UI.
		var s settings
		if c.settings == nil {
			s.Port = 9000
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		s, err := c.ValidateSettings(values)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		err = c.conf.SetSettings(s)
		if err != nil {
			return nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), nil
	default:
		return nil, nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Label: "Port", Placeholder: "9000", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 64},
			&ui.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&ui.Input{Name: "database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil, nil
}

// ValidateSettings validates the settings received from the UI and returns them
// in a format suitable for storage.
func (c *connection) ValidateSettings(values []byte) ([]byte, error) {
	var s settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return nil, err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, ui.Errorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, ui.Errorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 64 {
		return nil, ui.Errorf("username length in bytes must be in range [1,64]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return nil, ui.Errorf("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 64 {
		return nil, ui.Errorf("database length in bytes must be in range [1,64]")
	}
	err = testConnection(c.ctx, &s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&s)
}

type settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// options returns the connection options, from s.
func (s *settings) options() *clickhouse.Options {
	return &clickhouse.Options{
		Addr: []string{net.JoinHostPort(s.Host, strconv.Itoa(s.Port))},
		Auth: clickhouse.Auth{
			Database: s.Database,
			Username: s.Username,
			Password: s.Password,
		},
	}
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *settings) error {
	conn, err := clickhouse.Open(settings.options())
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping(ctx)
}

// propertyType returns the property type of the column type and a boolean
// indicating if it is nullable.
func propertyType(t driver.ColumnType) (types.Type, bool, error) {
	typ, nullable := columnType(t.DatabaseTypeName())
	if !typ.Valid() {
		return types.Type{}, false, connector.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
	}
	return typ, nullable, nil
}

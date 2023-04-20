//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package postgresql

// This package is the PostgreSQL connector.
// (https://www.postgresql.org/docs/15/)

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"
	"chichi/pkg/pgx/pgconn"
	_ "chichi/pkg/pgx/stdlib"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	connector.RegisterDatabase(connector.Database{
		Name:              "PostgreSQL",
		SourceDescription: "import users and groups from a PostgreSQL database",
		Icon:              icon,
	}, open)
}

// open opens a PostgreSQL connection and returns it.
func open(ctx context.Context, conf *connector.DatabaseConfig) (*connection, error) {
	c := connection{ctx: ctx, firehose: conf.Firehose}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of PostgreSQL connection")
		}
	}
	return &c, nil
}

type connection struct {
	ctx      context.Context
	settings *settings
	firehose connector.Firehose
}

// Query executes the given query and returns the resulting rows.
func (c *connection) Query(query string) (types.Type, connector.Rows, error) {
	db, err := sql.Open("pgx", c.settings.dsn())
	if err != nil {
		return types.Type{}, nil, err
	}
	db.SetMaxIdleConns(0)
	rows, err := db.QueryContext(c.ctx, query)
	if err != nil {
		_ = db.Close()
		if err, ok := err.(*pgconn.PgError); ok {
			return types.Type{}, nil, connector.NewDatabaseQueryError(err.Message)
		}
		return types.Type{}, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		_ = db.Close()
	}
	properties := make([]types.Property, len(columnTypes))
	for i, c := range columnTypes {
		typ, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
			_ = db.Close()
			return types.Type{}, nil, err
		}
		nullable, ok := c.Nullable()
		properties[i] = types.Property{
			Name:     c.Name(),
			Type:     typ,
			Nullable: nullable || !ok,
		}
	}
	return types.Object(properties), rows, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, *ui.Alert, error) {

	switch event {
	case "load":
		// Load the UI.
		var s settings
		if c.settings == nil {
			s.Port = 5432
		} else {
			s = *c.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		s, err := c.SettingsUI(values)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil
			}
			return nil, ui.DangerAlert(err.Error()), nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil
		}
		err = c.firehose.SetSettings(s)
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
			&ui.Input{Name: "port", Label: "Port", Placeholder: "5432", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 63},
			&ui.Input{Name: "password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&ui.Input{Name: "database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 63},
		},
		Values: values,
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
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
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, ui.Errorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, ui.Errorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := len(s.Username); n < 1 || n > 63 {
		return nil, ui.Errorf("username length in bytes must be in range [1,63]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
		return nil, ui.Errorf("password length must be in range [1,100]")
	}
	// Validate Database.
	if n := len(s.Database); n < 1 || n > 63 {
		return nil, ui.Errorf("database length in bytes must be in range [1,63]")
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

// dsn returns the connection string, from s, in the URL format.
func (s *settings) dsn() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(s.Username, s.Password),
		Host:   net.JoinHostPort(s.Host, strconv.Itoa(s.Port)),
		Path:   "/" + url.PathEscape(s.Database),
	}
	return u.String()
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *settings) error {
	db, err := sql.Open("pgx", settings.dsn())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column type t.
func propertyType(t *sql.ColumnType) (types.Type, error) {
	switch t.DatabaseTypeName() {
	case "BOOL":
		return types.Boolean(), nil
	case "BYTEA", "TEXT":
		return types.Text(), nil
	case "CHAR", "VARCHAR":
		length, ok := t.Length()
		if !ok {
			return types.Type{}, errors.New("cannot get length")
		}
		return types.Text(types.Chars(length)), nil
	case "DATE":
		return types.Date(""), nil // TODO(marco) set the layout
	case "FLOAT4":
		return types.Float32(), nil
	case "FLOAT8":
		return types.Float(), nil
	case "INT2":
		return types.Int16(), nil
	case "INT4":
		return types.Int(), nil
	case "INT8":
		return types.Int64(), nil
	case "JSON", "JSONB":
		return types.JSON(), nil
	case "NUMERIC":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, errors.New("cannot get decimal size")
		}
		if precision > types.MaxDecimalPrecision || scale > types.MaxDecimalScale {
			return types.Type{}, fmt.Errorf("PostreSQL type %s(%d,%d) is not supported",
				t.DatabaseTypeName(), precision, scale)
		}
		return types.Decimal(int(precision), int(scale)), nil
	case "TIME":
		return types.Time(), nil
	case "TIMESTAMP", "TIMESTAMPTZ":
		return types.DateTime(""), nil // TODO(marco) set the layout
	case "UUID":
		return types.UUID(), nil
	}
	return types.Type{}, connector.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}

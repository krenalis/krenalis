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

	"chichi/apis"
	"chichi/apis/types"
	"chichi/connector"
	"chichi/connector/ui"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connector icon.
var icon []byte

// Make sure it implements the DatabaseConnection interface.
var _ connector.DatabaseConnection = &connection{}

func init() {
	apis.RegisterDatabaseConnector("PostgreSQL", New)
}

// New returns a new PostgreSQL connection.
func New(ctx context.Context, conf *connector.DatabaseConfig) (connector.DatabaseConnection, error) {
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

// Connector returns the connector.
func (c *connection) Connector() *connector.Connector {
	return &connector.Connector{
		Name: "PostgreSQL",
		Type: connector.DatabaseType,
		Icon: icon,
	}
}

// Query executes the given query and returns the resulting rows.
func (c *connection) Query(query string) ([]connector.Column, connector.Rows, error) {
	db, err := sql.Open("pgx", c.settings.dsn())
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxIdleConns(0)
	rows, err := db.QueryContext(c.ctx, query)
	if err != nil {
		_ = db.Close()
		if err, ok := err.(*pgconn.PgError); ok {
			return nil, nil, connector.NewDatabaseQueryError(err.Message)
		}
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		_ = db.Close()
	}
	columns := make([]connector.Column, len(columnTypes))
	for i, c := range columnTypes {
		typ, err := propertyType(c)
		if err != nil {
			_ = rows.Close()
			_ = db.Close()
			return nil, nil, err
		}
		columns[i] = connector.Column{
			Name: c.Name(),
			Type: typ,
		}
	}
	return columns, rows, nil
}

// ServeUI serves the connector's user interface.
func (c *connection) ServeUI(event string, values []byte) (*ui.Form, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if c.settings == nil {
			s.Port = 5432
		} else {
			s = *c.settings
		}
	case "test", "save":
		// Test the connection and save the settings if required.
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
			return nil, ui.Errorf("connection failed: %s", err)
		}
		if event == "test" {
			return nil, nil
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, err
		}
		return nil, c.firehose.SetSettings(b)
	default:
		return nil, ui.ErrEventNotExist
	}

	form := &ui.Form{
		Fields: []ui.Component{
			&ui.Input{Name: "host", Value: s.Host, Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&ui.Input{Name: "port", Value: s.Port, Label: "Port", Placeholder: "5432", Type: "number", MinLength: 1, MaxLength: 5},
			&ui.Input{Name: "username", Value: s.Username, Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 63},
			&ui.Input{Name: "password", Value: s.Password, Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 100},
			&ui.Input{Name: "database", Value: s.Database, Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 63},
		},
		Actions: []ui.Action{
			{Event: "test", Text: "Test Connection", Variant: "neutral"},
			{Event: "save", Text: "Save", Variant: "primary"},
		},
	}

	return form, nil
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
		return types.Time(""), nil // TODO(marco) set the layout
	case "TIMESTAMP", "TIMESTAMPTZ":
		return types.DateTime(""), nil // TODO(marco) set the layout
	case "UUID":
		return types.UUID(), nil
	}
	return types.Type{}, connector.NewNotSupportedTypeError(t.Name(), t.DatabaseTypeName())
}

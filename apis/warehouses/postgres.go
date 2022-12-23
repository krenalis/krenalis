//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package warehouses

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chichi/apis/types"
	"chichi/connector/ui"

	"github.com/shopspring/decimal"
)

var _ Warehouse = &postgreSQL{}

type postgreSQL struct {
	mu       sync.Mutex // for the db and closed fields
	db       *sql.DB
	closed   bool
	settings *settings
}

// openPostgres the data warehouse with the given settings.
// If settings is nil, ServeUI is the only callable method.
func openPostgres(settings []byte) (*postgreSQL, error) {
	warehouse := &postgreSQL{}
	err := json.Unmarshal(settings, &warehouse.settings)
	if err != nil {
		return nil, err
	}
	return warehouse, nil
}

// Close closes the warehouse. It will not allow any new queries to run, and it
// waits for the current ones to finish.
func (warehouse *postgreSQL) Close() error {
	var err error
	warehouse.mu.Lock()
	if warehouse.db != nil {
		err = warehouse.db.Close()
		warehouse.db = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	return err
}

// Exec executes a query without returning any rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	result, err := db.ExecContext(ctx, query, args...)
	return result, wrapError(err)
}

// Type returns the type of the warehouse.
func (warehouse *postgreSQL) Type() Type {
	return PostgreSQL
}

// ServeUI serves the data warehouse's user interface.
func (warehouse *postgreSQL) ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, []byte, error) {

	var s settings

	switch event {
	case "load":
		// Load the UI.
		if warehouse.settings == nil {
			s.Port = 5432
		} else {
			s = *warehouse.settings
		}
		values, _ = json.Marshal(s)
	case "test", "save":
		// Test the connection and save the settings if required.
		err := json.Unmarshal(values, &s)
		if err != nil {
			return nil, nil, nil, err
		}
		// Validate Host.
		if n := len(s.Host); n == 0 || n > 253 {
			return nil, nil, nil, ui.Errorf("host length in bytes must be in range [1,253]")
		}
		// Validate Port.
		if s.Port < 1 || s.Port > 65536 {
			return nil, nil, nil, ui.Errorf("port must be in range [1,65536]")
		}
		// Validate Username.
		if n := len(s.Username); n < 1 || n > 63 {
			return nil, nil, nil, ui.Errorf("username length in bytes must be in range [1,63]")
		}
		// Validate Password.
		if n := utf8.RuneCountInString(s.Password); n < 1 || n > 100 {
			return nil, nil, nil, ui.Errorf("password length must be in range [1,100]")
		}
		// Validate Database.
		if n := len(s.Database); n < 1 || n > 63 {
			return nil, nil, nil, ui.Errorf("database length in bytes must be in range [1,63]")
		}
		err = testConnection(ctx, &s)
		if err != nil {
			if event == "test" {
				return nil, ui.WarningAlert(err.Error()), nil, nil
			}
			return nil, ui.DangerAlert(err.Error()), nil, nil
		}
		if event == "test" {
			return nil, ui.SuccessAlert("Connection established"), nil, nil
		}
		b, err := json.Marshal(&s)
		if err != nil {
			return nil, nil, nil, err
		}
		return nil, ui.SuccessAlert("Settings saved"), b, nil
	default:
		return nil, nil, nil, ui.ErrEventNotExist
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

	return form, nil, nil, nil
}

// Query executes a query that returns rows. args are the placeholders.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	return rows, wrapError(err)
}

// QueryRow executes a query that should return at most one row.
// If the query fails, it returns an Error value.
func (warehouse *postgreSQL) QueryRow(ctx context.Context, query string, args ...any) Row {
	db, err := warehouse.connection()
	if err != nil {
		return postgreSQLRow{err: err}
	}
	row := db.QueryRowContext(ctx, query, args...)
	return postgreSQLRow{row: row}
}

// Users returns the users, with only the properties in schema, ordered by
// order if order is not the zero Property, and in range [first,first+limit]
// with first >= 0 and 0 < limit <= 1000.
//
// If a query to the warehouse fails, it returns an Error value.
// If an argument is not valid, it panics.
func (warehouse *postgreSQL) Users(ctx context.Context, schema types.Schema, order types.Property, first, limit int) ([][]any, error) {

	db, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	properties := schema.Properties()

	// Build the query.
	var query strings.Builder
	query.WriteString(`SELECT "`)
	for i, p := range properties {
		if i > 0 {
			query.WriteString(`", "`)
		}
		if !types.IsValidPropertyName(p.Name) {
			panic(fmt.Sprintf("invalid property name: %q", p.Name))
		}
		query.WriteString(p.Name)
	}
	query.WriteString(`" FROM users`)
	if order.Name != "" {
		if !types.IsValidPropertyName(order.Name) {
			panic(fmt.Sprintf("invalid property name: %q", order.Name))
		}
		query.WriteString(" ORDER BY ")
		query.WriteString(order.Name)
	}
	query.WriteString(" LIMIT ")
	query.WriteString(strconv.Itoa(limit))
	if first > 0 {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(first))
	}

	// Execute the query.
	var users [][]any
	rows, err := db.QueryContext(ctx, query.String())
	if err != nil {
		return nil, wrapError(err)
	}
	for rows.Next() {
		user := make([]any, len(properties))
		for i := range user {
			typ := properties[i].Type
			switch typ.PhysicalType() {
			case types.PtBoolean:
				var v bool
				user[i] = &v
			case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
				var v int
				user[i] = &v
			case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
				var v uint
				user[i] = &v
			case types.PtFloat, types.PtFloat32:
				var v float64
				user[i] = &v
			case types.PtDecimal:
				var v decimal.Decimal
				user[i] = &v
			case types.PtDateTime, types.PtDate:
				var v time.Time
				user[i] = &v
			case types.PtTime, types.PtYear:
				var v int
				user[i] = &v
			case types.PtUUID, types.PtJSON, types.PtText, types.PtArray, types.PtObject, types.PtMap:
				var v string
				user[i] = &v
			}
		}
		if err = rows.Scan(user...); err != nil {
			_ = rows.Close()
			return nil, wrapError(err)
		}
		users = append(users, user)
	}
	if err = rows.Err(); err != nil {
		return nil, wrapError(err)
	}
	err = rows.Close()
	if err != nil {
		log.Printf("cannot close rows: %s", err)
	}
	if users == nil {
		users = [][]any{}
	}

	return users, nil
}

// connection returns the database connection.
func (warehouse *postgreSQL) connection() (*sql.DB, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.closed {
		return nil, wrapError(errors.New("warehouse is closed"))
	}
	if warehouse.db != nil {
		return warehouse.db, nil
	}
	db, err := sql.Open("pgx", warehouse.settings.dsn())
	if err != nil {
		return nil, wrapError(err)
	}
	warehouse.db = db
	return db, nil
}

// postgreSQLRow implements the Row interface.
type postgreSQLRow struct {
	row *sql.Row
	err error
}

func (row postgreSQLRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	err := row.row.Scan(dest...)
	return wrapError(err)
}

func (row postgreSQLRow) Err() error {
	if row.err != nil {
		return row.err
	}
	err := row.row.Err()
	return wrapError(err)
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

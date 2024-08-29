//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"github.com/meergo/meergo/apis/datastore/warehouses"

	"github.com/ClickHouse/clickhouse-go/v2"
	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var (
	//go:embed tables/destinations_users.sql
	createDestinationUsersTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/users.sql
	createUsersTable string
)

var _ warehouses.Warehouse = &ClickHouse{}

type ClickHouse struct {
	mu       sync.Mutex // for the conn field
	conn     chDriver.Conn
	settings *chSettings
}

type chSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

// Open opens a ClickHouse data warehouse with the given settings.
// It returns a SettingsError error if the settings are not valid.
func Open(settings []byte) (warehouses.Warehouse, error) {
	var s chSettings
	err := json.Unmarshal(settings, &s)
	if err != nil {
		return nil, warehouses.SettingsErrorf("cannot unmarshal settings: %s", err)
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return nil, warehouses.SettingsErrorf("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return nil, warehouses.SettingsErrorf("port must be in range [1,65536]")
	}
	// Validate Username.
	if s.Username == "" {
		return nil, warehouses.SettingsErrorf("username cannot be empty")
	}
	// Validate Database.
	if s.Database == "" {
		return nil, warehouses.SettingsErrorf("database cannot be empty")
	}
	return &ClickHouse{settings: &s}, nil
}

// AlterSchema alters the user schema.
func (warehouse *ClickHouse) AlterSchema(ctx context.Context, userColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) error {
	panic("TODO: not implemented")
}

// AlterSchemaQueries returns the queries of a schema altering operation.
func (warehouse *ClickHouse) AlterSchemaQueries(ctx context.Context, userColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) ([]string, error) {
	panic("TODO: not implemented")
}

// Close closes the data warehouse.
func (warehouse *ClickHouse) Close() error {
	if warehouse.conn == nil {
		return nil
	}
	err := warehouse.conn.Close()
	warehouse.conn = nil
	return err
}

// Delete deletes rows from the specified table that match the provided where
// expression.
func (warehouse *ClickHouse) Delete(ctx context.Context, table string, where warehouses.Expr) error {
	if where == nil {
		return errors.New("where is nil")
	}
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	var s strings.Builder
	s.WriteString("DELETE FROM `" + table + "` WHERE ")
	err = renderExpr(&s, where)
	if err != nil {
		return fmt.Errorf("cannot build WHERE expression: %s", err)
	}
	err = db.Exec(ctx, s.String())
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// IdentityResolutionExecution returns information about the execution of the
// Identity Resolution.
func (warehouse *ClickHouse) IdentityResolutionExecution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	panic("not implemented")
}

// Init initializes the data warehouse by creating the supporting tables.
func (warehouse *ClickHouse) Init(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	tables := []string{
		createDestinationUsersTable,
		createEventsTable,
		createUsersTable,
	}
	for _, table := range tables {
		err := conn.Exec(ctx, table)
		if err != nil {
			return warehouses.Error(err)
		}
	}
	return nil
}

// Merge performs a table merge operation.
func (warehouse *ClickHouse) Merge(ctx context.Context, table warehouses.Table, rows [][]any, deleted []any) error {
	return errors.New("not implemented yet")
}

// MergeIdentities merge existing identities, deletes them and inserts new ones.
func (warehouse *ClickHouse) MergeIdentities(ctx context.Context, columns []warehouses.Column, rows []map[string]any) error {
	panic("TODO: not implemented")
}

// Ping checks the connection to the data warehouse.
func (warehouse *ClickHouse) Ping(ctx context.Context) error {
	conn, err := warehouse.connection()
	if err != nil {
		return err
	}
	err = conn.Ping(ctx)
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// Query executes a query and returns the results as Rows.
func (warehouse *ClickHouse) Query(ctx context.Context, query warehouses.RowQuery, withCount bool) (warehouses.Rows, int, error) {
	panic("not implemented")
}

// RunIdentityResolution runs the Identity Resolution.
func (warehouse *ClickHouse) RunIdentityResolution(ctx context.Context, identifiers, userColumns []warehouses.Column, userPrimarySources map[string]int) error {
	panic("TODO: not implemented")
}

// SetDestinationUser sets the destination user for an action.
func (warehouse *ClickHouse) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	panic("TODO: not implemented")
}

// Settings returns the data warehouse settings.
func (warehouse *ClickHouse) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

// Truncate truncates the specified table.
func (warehouse *ClickHouse) Truncate(ctx context.Context, table string) error {
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	err = db.Exec(ctx, "TRUNCATE TABLE `"+table+"`")
	if err != nil {
		return warehouses.Error(err)
	}
	return nil
}

// connection returns the ClickHouse connection.
func (warehouse *ClickHouse) connection() (clickhouse.Conn, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.conn != nil {
		return warehouse.conn, nil
	}
	conn, err := clickhouse.Open(warehouse.settings.options())
	if err != nil {
		return nil, warehouses.Error(err)
	}
	warehouse.conn = conn
	return conn, nil
}

// options returns s as a *clickhouse.Options value.
func (s *chSettings) options() *clickhouse.Options {
	return &clickhouse.Options{
		Addr: []string{net.JoinHostPort(s.Host, strconv.Itoa(s.Port))},
		Auth: clickhouse.Auth{
			Database: s.Database,
			Username: s.Username,
			Password: s.Password,
		},
	}
}

// quoteValue quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexAny(s, "\\'")
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		b.WriteByte('\\')
		b.WriteByte(s[p])
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}

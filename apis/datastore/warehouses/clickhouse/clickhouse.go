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
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // workaround for clickhouse-go issue #162

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"

	"github.com/ClickHouse/clickhouse-go/v2"
	chDriver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

var (
	//go:embed tables/destinations_users.sql
	createDestinationUsersTable string
	//go:embed tables/events.sql
	createEventsTable string
	//go:embed tables/groups.sql
	createGroupsTable string
	//go:embed tables/users.sql
	createUsersTable string
)

var _ warehouses.Warehouse = &ClickHouse{}

type ClickHouse struct {
	mu       sync.Mutex // for the conn and closed fields
	conn     chDriver.Conn
	closed   bool
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

// AlterSchema alters the users schemas by applying the given operations.
func (warehouse *ClickHouse) AlterSchema(ctx context.Context, operations []warehouses.AlterSchemaOperation) error {
	panic("TODO: not implemented")
}

// AlterSchemaQueries returns the queries relative to the given operations.
func (warehouse *ClickHouse) AlterSchemaQueries(ctx context.Context, operations []warehouses.AlterSchemaOperation) ([]string, error) {
	panic("TODO: not implemented")
}

// Close closes the warehouse.
func (warehouse *ClickHouse) Close() error {
	var err error
	warehouse.mu.Lock()
	if warehouse.conn != nil {
		err = warehouse.conn.Close()
		warehouse.conn = nil
		warehouse.closed = true
	}
	warehouse.mu.Unlock()
	return err
}

// DeleteConnectionIdentities deletes the identities of a connection.
func (warehouse *ClickHouse) DeleteConnectionIdentities(ctx context.Context, connection int) error {
	panic("not implemented")
}

// DestinationUsers returns the destination users of the action.
func (warehouse *ClickHouse) DestinationUsers(ctx context.Context, action int, propertyValue string) ([]string, error) {
	panic("TODO: not implemented")
}

// DuplicatedDestinationUsers retrieves duplicated destination users.
func (warehouse *ClickHouse) DuplicatedDestinationUsers(ctx context.Context, action int) (string, string, bool, error) {
	panic("TODO: not implemented")
}

// DuplicatedUsers returns the GIDs of two duplicated users.
func (warehouse *ClickHouse) DuplicatedUsers(ctx context.Context, property string) (int, int, bool, error) {
	panic("TODO: not implemented")
}

// IdentitiesWriter returns an IdentitiesWriter.
func (warehouse *ClickHouse) IdentitiesWriter(ctx context.Context, schema types.Type, connection int, fromEvent bool, ack warehouses.IdentitiesAckFunc) warehouses.IdentitiesWriter {
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
		createGroupsTable,
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
func (warehouse *ClickHouse) Merge(ctx context.Context, table warehouses.MergeTable, rows []map[string]any, deleted map[string]any) error {
	return errors.New("not implemented yet")
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

// SetDestinationUser sets the destination user for an action.
func (warehouse *ClickHouse) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	panic("TODO: not implemented")
}

// Settings returns the data warehouse settings.
func (warehouse *ClickHouse) Settings() []byte {
	s, _ := json.Marshal(warehouse.settings)
	return s
}

type warehouseTable struct {
	Name   string
	Schema types.Type
}

// Tables returns the tables of the data warehouse.
// It returns only the tables 'users', 'users_identities', 'groups',
// 'groups_identities' and 'events'.
func (warehouse *ClickHouse) tables(ctx context.Context) ([]*warehouseTable, error) {

	// TODO(Gianluca): this method has been kept (and made unexported) as it
	// may be useful in the future when we will complete the implementation of
	// the ClickHouse driver.
	//
	// This is related to https://github.com/open2b/chichi/issues/582.

	// Get the connection.
	conn, err := warehouse.connection()
	if err != nil {
		return nil, err
	}

	// Read columns.
	query := "SELECT c.table_name, c.column_name, c.data_type, c.column_comment\n" +
		"FROM information_schema.columns c\n" +
		"INNER JOIN information_schema.tables t ON c.table_name = t.table_name AND c.table_schema = t.table_schema\n" +
		"WHERE t.table_schema = '" + warehouse.settings.Database + "' AND t.table_type = 'BASE TABLE' AND" +
		" ( t.table_name IN ('users', 'users_identities', 'groups', 'groups_identities', 'events') )\n" +
		"ORDER BY c.table_name, c.ordinal_position"

	type clickHouseTable struct {
		Name    string
		Columns []types.Property
	}

	var table *clickHouseTable
	var tables []*clickHouseTable

	rows, err := conn.Query(ctx, query)
	if err != nil {
		return nil, warehouses.Error(err)
	}
	defer rows.Close()
	for rows.Next() {
		var tableName, columnName, typ, comment string
		if err = rows.Scan(&tableName, &columnName, &typ, &comment); err != nil {
			return nil, warehouses.Error(err)
		}
		if strings.HasPrefix(columnName, "__") && strings.HasSuffix(columnName, "__") { // used internally by Chichi.
			continue
		}
		if !types.IsValidPropertyName(columnName) {
			return nil, warehouses.Errorf("column name %q is not supported", columnName)
		}
		column := types.Property{
			Name:        columnName,
			Description: comment,
		}
		column.Type, column.Nullable = columnType(typ)
		if !column.Type.Valid() {
			return nil, warehouses.Errorf("type %q of column %s is not supported", typ, column.Name)
		}
		if table == nil || tableName != table.Name {
			table = &clickHouseTable{Name: tableName}
			tables = append(tables, table)
		}
		table.Columns = append(table.Columns, column)
	}
	if err = rows.Close(); err != nil {
		return nil, warehouses.Error(err)
	}
	if err := rows.Err(); err != nil {
		return nil, warehouses.Error(err)
	}

	// Transform the ClickHouse columns in properties.
	whTables := make([]*warehouseTable, len(tables))
	for i, t := range tables {
		props, err := warehouses.ColumnsToProperties(t.Columns)
		if err != nil {
			return nil, warehouses.Error(err)
		}
		schema, err := types.ObjectOf(props)
		if err != nil {
			return nil, warehouses.Error(err)
		}
		whTables[i] = &warehouseTable{
			Name:   t.Name,
			Schema: schema,
		}
	}

	return whTables, nil
}

// Records returns an iterator over the results of the query.
func (warehouse *ClickHouse) Records(ctx context.Context, query warehouses.RecordsQuery) (warehouses.Records, int, error) {
	panic("not implemented")
}

// RunWorkspaceIdentityResolution runs the Workspace Identity Resolution.
func (warehouse *ClickHouse) RunWorkspaceIdentityResolution(ctx context.Context, connections []int, identifiers []types.Property, usersSchema types.Type) error {
	panic("TODO: not implemented")
}

// connection returns the database connection.
func (warehouse *ClickHouse) connection() (clickhouse.Conn, error) {
	warehouse.mu.Lock()
	defer warehouse.mu.Unlock()
	if warehouse.closed {
		return nil, errors.New("warehouse is closed")
	}
	if warehouse.settings == nil {
		return nil, errors.New("there are no settings")
	}
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

// quoteValue quotes value and writes it into b.
func quoteValue(b *strings.Builder, value any) {
	if value == nil {
		b.WriteString("NULL")
		return
	}
	switch v := value.(type) {
	case bool:
		if v {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int16:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint16:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(v, 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(v), 'G', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
	case netip.Addr:
		quoteString(b, v.String())
	case string:
		quoteString(b, v)
	case time.Time:
		b.WriteByte('\'')
		b.WriteString(v.Format("2006-01-02 15:04:05"))
		b.WriteByte('\'')
	default:
		panic(fmt.Errorf("unsupported type '%T'", v))
	}
}

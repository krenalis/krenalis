//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

// Package mysql implements the MySQL connector.
// (https://dev.mysql.com/doc/refman/8.0/en/)
package mysql

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"

	"github.com/go-sql-driver/mysql"
)

// Connector icon.
var icon = "<svg></svg>"

// Make sure it implements the Database and UIHandler interfaces.
var _ interface {
	meergo.Database
	meergo.UIHandler
} = (*MySQL)(nil)

func init() {
	meergo.RegisterDatabase(meergo.DatabaseInfo{
		Name:        "MySQL",
		SampleQuery: "SELECT *\nFROM users\nWHERE ${last_change_time}\nLIMIT ${limit}\n",
		Icon:        icon,
	}, New)
}

// New returns a new MySQL connector instance.
func New(conf *meergo.DatabaseConfig) (*MySQL, error) {
	c := MySQL{conf: conf}
	if len(conf.Settings) > 0 {
		err := json.Unmarshal(conf.Settings, &c.settings)
		if err != nil {
			return nil, errors.New("cannot unmarshal settings of MySQL connector")
		}
	}
	return &c, nil
}

type MySQL struct {
	conf     *meergo.DatabaseConfig
	settings *Settings
	db       *sql.DB
}

// Close closes the database.
func (my *MySQL) Close() error {
	if my.db == nil {
		return nil
	}
	return my.db.Close()
}

// Columns returns the columns of the given table.
func (my *MySQL) Columns(ctx context.Context, table string) ([]types.Property, error) {
	var err error
	table, err = quoteTable(table)
	if err != nil {
		return nil, err
	}
	rows, columns, err := my.query(ctx, "SELECT * FROM "+table+" LIMIT 0")
	if err != nil {
		return nil, err
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}
	return columns, nil
}

// LastChangeTimeCondition returns the query condition used for the
// last_change_time placeholder in the form "column >= value" or, if column is
// empty, a true value.
func (my *MySQL) LastChangeTimeCondition(column string, typ types.Type, value any) string {
	if column == "" {
		return "TRUE"
	}
	var err error
	column, err = quoteColumn(column)
	if err != nil {
		panic(err)
	}
	b := strings.Builder{}
	b.WriteString(column)
	b.WriteString(` >= `)
	_ = quoteValue(&b, value, typ)
	return b.String()
}

// Query executes the given query and returns the resulting rows and columns.
func (my *MySQL) Query(ctx context.Context, query string) (meergo.Rows, []types.Property, error) {
	return my.query(ctx, query)
}

// ServeUI serves the connector's user interface.
func (my *MySQL) ServeUI(ctx context.Context, event string, values []byte, role meergo.Role) (*meergo.UI, error) {

	switch event {
	case "load":
		var s Settings
		if my.settings == nil {
			s.Port = 3306
		} else {
			s = *my.settings
		}
		values, _ = json.Marshal(s)
	case "save":
		return nil, my.saveValues(ctx, values, false)
	case "test":
		return nil, my.saveValues(ctx, values, true)
	default:
		return nil, meergo.ErrUIEventNotExist
	}

	ui := &meergo.UI{
		Fields: []meergo.Component{
			&meergo.Input{Name: "Host", Label: "Host", Placeholder: "example.com", Type: "text", MinLength: 1, MaxLength: 253},
			&meergo.Input{Name: "Port", Label: "Port", Placeholder: "3306", Type: "number", OnlyIntegerPart: true, MinLength: 1, MaxLength: 5},
			&meergo.Input{Name: "Username", Label: "Username", Placeholder: "username", Type: "text", MinLength: 1, MaxLength: 16},
			&meergo.Input{Name: "Password", Label: "Password", Placeholder: "password", Type: "password", MinLength: 1, MaxLength: 200},
			&meergo.Input{Name: "Database", Label: "Database name", Placeholder: "database", Type: "text", MinLength: 1, MaxLength: 64},
		},
		Values: values,
		Buttons: []meergo.Button{
			{Event: "test", Text: "Test connection", Variant: "neutral"},
		},
	}

	return ui, nil
}

// Upsert inserts or updates the rows provided in the specified table.
func (my *MySQL) Upsert(ctx context.Context, table meergo.Table, rows []map[string]any) error {

	name, err := quoteTable(table.Name)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(name)
	b.WriteString(" (")
	for i, column := range table.Columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('`')
		b.WriteString(column.Name)
		b.WriteByte('`')
	}
	b.WriteString(") VALUES ")
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("(")
		for j, column := range table.Columns {
			if j > 0 {
				b.WriteByte(',')
			}
			if v, ok := row[column.Name]; ok {
				if err = quoteValue(&b, v, column.Type); err != nil {
					return err
				}
			} else {
				b.WriteString(`DEFAULT`)
			}
		}
		b.WriteByte(')')
	}
	b.WriteString(` ON DUPLICATE KEY UPDATE `)
	i := 0
	for _, column := range table.Columns {
		if column.Name == table.Key {
			continue
		}
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('`')
		b.WriteString(column.Name)
		b.WriteString("` = VALUES(`")
		b.WriteString(column.Name)
		b.WriteString("`)")
		i++
	}
	query := b.String()

	if err = my.openDB(); err != nil {
		return err
	}
	_, err = my.db.ExecContext(ctx, query)

	return err
}

// openDB opens the database. If the database is already open it does nothing.
func (my *MySQL) openDB() error {
	if my.db != nil {
		return nil
	}
	mysqlConnector, err := mysql.NewConnector(my.settings.config())
	if err != nil {
		return err
	}
	db := sql.OpenDB(mysqlConnector)
	db.SetMaxIdleConns(0)
	my.db = db
	return nil
}

// query executes the given query and returns the resulting rows and columns.
func (my *MySQL) query(ctx context.Context, query string) (meergo.Rows, []types.Property, error) {
	if err := my.openDB(); err != nil {
		return nil, nil, err
	}
	rows, err := my.db.QueryContext(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		_ = rows.Close()
		return nil, nil, err
	}
	columns := make([]types.Property, len(columnTypes))
	for i, column := range columnTypes {
		typ, err := propertyType(column)
		if err != nil {
			_ = rows.Close()
			return nil, nil, fmt.Errorf("cannot get type for property %q: %s", column.Name(), err)
		}
		// Unlike what happens with PostgreSQL, the MySQL driver is able to
		// determine whether a column returned by the query is nullable or not.
		nullable, ok := column.Nullable()
		columns[i] = types.Property{
			Name:     column.Name(),
			Type:     typ,
			Nullable: nullable || !ok,
		}
	}
	return rows, columns, nil
}

// saveValues saves the user-entered values as settings. If test is true, it
// validates only the values without saving it.
func (my *MySQL) saveValues(ctx context.Context, values []byte, test bool) error {
	var s Settings
	err := json.Unmarshal(values, &s)
	if err != nil {
		return err
	}
	// Validate Host.
	if n := len(s.Host); n == 0 || n > 253 {
		return meergo.NewInvalidUIValuesError("host length in bytes must be in range [1,253]")
	}
	// Validate Port.
	if s.Port < 1 || s.Port > 65536 {
		return meergo.NewInvalidUIValuesError("port must be in range [1,65536]")
	}
	// Validate Username.
	if n := utf8.RuneCountInString(s.Username); n < 1 || n > 16 {
		return meergo.NewInvalidUIValuesError("username length must be in range [1,16]")
	}
	// Validate Password.
	if n := utf8.RuneCountInString(s.Password); n < 1 || n > 200 {
		return meergo.NewInvalidUIValuesError("password length must be in range [1,200]")
	}
	// Validate Database.
	if n := utf8.RuneCountInString(s.Database); n < 1 || n > 64 {
		return meergo.NewInvalidUIValuesError("database length must be in range [1,64]")
	}
	err = testConnection(ctx, &s)
	if err != nil || test {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = my.conf.SetSettings(ctx, b)
	if err != nil {
		return err
	}
	my.settings = &s
	return nil
}

type Settings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
}

func (s *Settings) config() *mysql.Config {
	c := mysql.NewConfig()
	c.User = s.Username
	c.Passwd = s.Password
	c.DBName = s.Database
	c.AllowOldPasswords = true
	c.ParseTime = true
	return c
}

// testConnection tests a connection with the given settings.
// Returns an error if the connection cannot be established.
func testConnection(ctx context.Context, settings *Settings) error {
	mysqlConnector, err := mysql.NewConnector(settings.config())
	if err != nil {
		return err
	}
	db := sql.OpenDB(mysqlConnector)
	defer db.Close()
	db.SetMaxIdleConns(0)
	return db.PingContext(ctx)
}

// propertyType returns the property type of the column with type t.
func propertyType(t *sql.ColumnType) (types.Type, error) {
	switch t.DatabaseTypeName() {
	case "BLOB":
		return types.Text(), nil
	case "DATE":
		return types.Date(), nil
	case "DATETIME":
		return types.DateTime(), nil
	case "DECIMAL":
		precision, scale, ok := t.DecimalSize()
		if !ok {
			return types.Type{}, errors.New("cannot get decimal size")
		}
		return types.Decimal(int(precision), int(scale)), nil
	case "DOUBLE":
		return types.Float(64), nil
	case "ENUM":
		return types.Text(), nil
	// TODO(marco): SET can be implemented as an Array(Type), but the driver only returns the first element of the set.
	//case "SET":
	//return types.Array(types.Text()), nil
	case "FLOAT":
		return types.Float(32), nil
	case "UNSIGNED MEDIUMINT":
		return types.Uint(24), nil
	case "MEDIUMINT":
		return types.Int(24), nil
	case "JSON":
		return types.JSON(), nil
	case "UNSIGNED INT":
		return types.Uint(32), nil
	case "INT":
		return types.Int(32), nil
	case "UNSIGNED BIGINT":
		return types.Uint(64), nil
	case "BIGINT":
		return types.Int(64), nil
	case "UNSIGNED SMALLINT":
		return types.Uint(16), nil
	case "SMALLINT":
		return types.Int(16), nil
	case "VARCHAR", "CHAR":
		length, ok := t.Length()
		if !ok {
			return types.Text(), nil
		}
		return types.Text().WithCharLen(int(length)), nil
	case "VARBINARY", "BINARY":
		length, ok := t.Length()
		if !ok {
			return types.Text(), nil
		}
		return types.Text().WithByteLen(int(length)), nil
	case "TEXT":
		return types.Text(), nil
	case "TIME":
		return types.Time(), nil
	case "TIMESTAMP":
		return types.DateTime(), nil
	case "UNSIGNED TINYINT":
		return types.Uint(8), nil
	case "TINYINT":
		return types.Int(8), nil
	case "YEAR":
		return types.Year(), nil
	}
	return types.Type{}, meergo.NewUnsupportedColumnTypeError(t.Name(), t.DatabaseTypeName())
}

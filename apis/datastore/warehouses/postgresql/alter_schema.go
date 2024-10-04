//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/types"

	"github.com/jackc/pgx/v5"
)

// AlterSchema alters the user schema.
func (warehouse *PostgreSQL) AlterSchema(ctx context.Context, userColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) error {

	// Start an AlterSchema operation on the data warehouse, then defer its
	// ending.
	opID, err := warehouse.startOperation(ctx, alterSchema)
	if err != nil {
		return err
	}
	defer func() {
		err := warehouse.endOperation(ctx, opID, time.Now().UTC())
		if err != nil {
			go func() {
				slog.Error("cannot end data warehouse operation", "id", opID, "err", err)
			}()
		}
	}()

	// Retrieve the current version of the "users" table.
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return err
	}

	// Determine the alter schema queries.
	queries, err := alterSchemaQueries("_users_"+strconv.Itoa(usersVersion), userColumns, operations)
	if err != nil {
		return err
	}

	// Execute the alter schema queries within a transaction.
	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		for _, query := range queries {
			_, err := tx.Exec(ctx, query)
			if err != nil {
				return warehouses.Error(err)
			}
		}
		return nil
	})

	return err
}

// AlterSchemaQueries returns the queries of a schema altering operation.
func (warehouse *PostgreSQL) AlterSchemaQueries(ctx context.Context, userColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) ([]string, error) {
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return nil, err
	}
	queries, err := alterSchemaQueries("_users_"+strconv.Itoa(usersVersion), userColumns, operations)
	if err != nil {
		return nil, err
	}
	queries = append([]string{"BEGIN"}, queries...)
	queries = append(queries, "COMMIT")
	for i, q := range queries {
		queries[i] = q + ";"
	}
	return queries, nil
}

// alterSchemaQueries returns the queries that perform the given operations.
// usersTableName is the current name of the users table, for example
// "_users_42". operations must contain at least one operation.
//
// In case of incompatible type, returns a *warehouses.UnsupportedColumnType
// error.
func alterSchemaQueries(usersTableName string, userColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) ([]string, error) {

	// The operations are performed in this order:
	//
	// (1) DROP VIEW.
	// (2) DROP columns.
	// (3) RENAME columns (each in its own ALTER TABLE, see the PostgreSQL syntax for ALTER TABLE).
	// (4) ADD columns.
	// (5) CREATE VIEW.

	// Note that it is necessary to discard the view and rebuild it from
	// scratch, as the table needs to be altered and it would be impossible to
	// alter a column if there is a view that references it.

	var queries []string

	// DROP VIEW.
	queries = append(queries, "DROP VIEW \"users\"")

	// ALTER TABLE ... DROP COLUMN.
	{
		var toDrop []string
		for _, op := range operations {
			if op.Operation == warehouses.OperationDropColumn {
				toDrop = append(toDrop, op.Column)
			}
		}
		if len(toDrop) > 0 {
			for _, table := range []string{usersTableName, "_user_identities"} {
				b := strings.Builder{}
				b.WriteString("ALTER TABLE " + quoteIdent(table) + "\n\t")
				for i, c := range toDrop {
					if i > 0 {
						b.WriteString(",\n\t")
					}
					b.WriteString(`DROP COLUMN "` + c + `"`)
				}
				queries = append(queries, b.String())
			}
		}
	}

	// ALTER TABLE ... RENAME COLUMN.
	for _, op := range operations {
		if op.Operation == warehouses.OperationRenameColumn {
			queries = append(queries, `ALTER TABLE `+quoteIdent(usersTableName)+"\n\tRENAME COLUMN \""+op.Column+`" TO "`+op.NewColumn+`"`)
			queries = append(queries, `ALTER TABLE "_user_identities"`+"\n\tRENAME COLUMN \""+op.Column+`" TO "`+op.NewColumn+`"`)
		}
	}

	// ALTER TABLE ... ADD COLUMN.
	{
		var toAdd []warehouses.AlterSchemaOperation
		for _, op := range operations {
			if op.Operation == warehouses.OperationAddColumn {
				toAdd = append(toAdd, op)
			}
		}
		if len(toAdd) > 0 {
			for _, table := range []string{usersTableName, "_user_identities"} {
				b := strings.Builder{}
				b.WriteString("ALTER TABLE " + quoteIdent(table) + "\n\t")
				for i, op := range toAdd {
					if i > 0 {
						b.WriteString(",\n\t")
					}
					typ, ok := typeToPostgresType(op.Type)
					if !ok {
						return nil, warehouses.NewUnsupportedColumnType(op.Column, op.Type)
					}
					b.WriteString(`ADD COLUMN "` + op.Column + `" ` + typ)
				}
				queries = append(queries, b.String())
			}
		}
	}

	// CREATE VIEW "users".
	queries = append(queries, createViewQuery(usersTableName, userColumns, false))

	return queries, nil
}

// createViewQuery returns the CREATE (OR REPLACE) VIEW query that creates the
// "users" view on the "users" table with the given name.
// userColumns contains the columns of such table.
// replace indicates if the query that creates the VIEW should have the "OR
// REPLACE" clause to replace the view if it already exist.
func createViewQuery(usersTableName string, userColumns []warehouses.Column, replace bool) string {
	b := strings.Builder{}
	b.WriteString(`CREATE `)
	if replace {
		b.WriteString(`OR REPLACE `)
	}
	b.WriteString(`VIEW "users" AS SELECT` + "\n")
	metaProps := []string{"__id__", "__last_change_time__"}
	for i, p := range metaProps {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("\t\"")
		b.WriteString(p)
		b.WriteRune('"')
	}
	for _, c := range userColumns {
		b.WriteString(",\n\t\"")
		b.WriteString(c.Name)
		b.WriteRune('"')
	}
	b.WriteString("\nFROM ")
	b.WriteString(quoteIdent(usersTableName))
	return b.String()
}

func typeToPostgresType(t types.Type) (string, bool) {
	switch t.Kind() {
	case types.BooleanKind:
		return "boolean", true
	case types.IntKind:
		min, max := t.IntRange()
		switch t.BitSize() {
		case 16:
			if min > types.MinInt16 || max < types.MaxInt16 {
				return "", false
			}
			return "smallint", true
		case 32:
			if min > types.MinInt32 || max < types.MaxInt32 {
				return "", false
			}
			return "integer", true
		case 64:
			if min > types.MinInt64 || max < types.MaxInt64 {
				return "", false
			}
			return "bigint", true
		default:
			return "", false
		}
	case types.UintKind:
		return "", false
	case types.FloatKind:
		if t.IsReal() {
			return "", false
		}
		min, max := t.FloatRange()
		switch t.BitSize() {
		case 32:
			if min > -math.MaxFloat32 || max < math.MaxFloat32 {
				return "", false
			}
			return "real", true
		case 64:
			if min > -math.MaxFloat64 || max < math.MaxFloat64 {
				return "", false
			}
			return "double precision", true
		}
	case types.DecimalKind:
		// TODO(Gianluca): for decimal types specifying a minimum and a maximum
		// value, see https://github.com/meergo/meergo/issues/578.
		p := t.Precision()
		s := t.Scale()
		if p < 1 || p > 76 || s > 37 {
			return "", false
		}
		return fmt.Sprintf("decimal(%d, %d)", p, s), true
	case types.DateTimeKind:
		return "timestamp without time zone", true
	case types.DateKind:
		return "date", true
	case types.TimeKind:
		return "time without time zone", true
	case types.YearKind:
		return "", false
	case types.UUIDKind:
		return "uuid", true
	case types.JSONKind:
		return "jsonb", true
	case types.InetKind:
		return "inet", true
	case types.TextKind:
		if _, ok := t.ByteLen(); ok {
			return "", false
		}
		typ := "varchar"
		if l, ok := t.CharLen(); ok {
			typ += "(" + strconv.Itoa(l) + ")"
		}
		return typ, true
	case types.ArrayKind:
		typ, ok := typeToPostgresType(t.Elem())
		if !ok {
			return "", false
		}
		return typ + "[]", true
	case types.MapKind:
		return "", false
	}
	return "", false
}

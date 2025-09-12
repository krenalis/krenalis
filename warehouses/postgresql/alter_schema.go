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
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/backoff"
	"github.com/meergo/meergo/types"

	"github.com/jackc/pgx/v5"
)

// AlterUserSchema alters the user schema.
func (warehouse *PostgreSQL) AlterUserSchema(ctx context.Context, opID string, columns []meergo.Column, operations []meergo.AlterOperation) error {
	status, err := warehouse.executeOperation(ctx, opID, alterUserSchema)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		return status.executionError
	}
	err = warehouse.alterUserSchema(ctx, columns, operations)
	bo := backoff.New(200)
	bo.SetCap(time.Second)
	for bo.Next(ctx) {
		err2 := warehouse.setOperationAsCompleted(ctx, opID, err)
		if err2 != nil {
			slog.Error("cannot set alter user columns operation as completed, retrying", "err", err2, "operationError", err)
			continue
		}
		if err != nil {
			return meergo.NewOperationError(err)
		}
		return nil
	}
	return ctx.Err()
}

func (warehouse *PostgreSQL) alterUserSchema(ctx context.Context, columns []meergo.Column, operations []meergo.AlterOperation) error {

	// Retrieve the current version of the "users" table.
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return err
	}

	// Determine the alter schema queries.
	queries := alterUserSchemaQueries("_users_"+strconv.Itoa(usersVersion), columns, operations)

	// Execute the alter schema queries within a transaction.
	err = warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		for _, query := range queries {
			_, err := tx.Exec(ctx, query)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

// PreviewAlterUserSchema provides a preview of an alter user schema operation
// by returning the queries that would be executed on the warehouse to perform a
// given alter schema.
func (warehouse *PostgreSQL) PreviewAlterUserSchema(ctx context.Context, columns []meergo.Column, operations []meergo.AlterOperation) ([]string, error) {
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return nil, err
	}
	queries := alterUserSchemaQueries("_users_"+strconv.Itoa(usersVersion), columns, operations)
	queries = append([]string{"BEGIN"}, queries...)
	queries = append(queries, "COMMIT")
	for i, q := range queries {
		queries[i] = q + ";"
	}
	return queries, nil
}

// alterUserSchemaQueries returns the queries that perform the given operations.
// usersTableName is the current name of the users table, for example
// "_users_42". operations must contain at least one operation.
func alterUserSchemaQueries(usersTableName string, columns []meergo.Column, operations []meergo.AlterOperation) []string {

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
	queries = append(queries, `DROP VIEW "users"`)

	// ALTER TABLE ... DROP COLUMN.
	{
		var toDrop []string
		for _, op := range operations {
			if op.Operation == meergo.OperationDropColumn {
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
					b.WriteString(`DROP COLUMN ` + quoteIdent(c))
				}
				queries = append(queries, b.String())
			}
		}
	}

	// ALTER TABLE ... RENAME COLUMN.
	for _, op := range operations {
		if op.Operation == meergo.OperationRenameColumn {
			queries = append(queries, `ALTER TABLE `+quoteIdent(usersTableName)+"\n\tRENAME COLUMN "+quoteIdent(op.Column)+` TO `+quoteIdent(op.NewColumn))
			queries = append(queries, `ALTER TABLE "_user_identities"`+"\n\tRENAME COLUMN "+quoteIdent(op.Column)+` TO `+quoteIdent(op.NewColumn))
		}
	}

	// ALTER TABLE ... ADD COLUMN.
	{
		var toAdd []meergo.AlterOperation
		for _, op := range operations {
			if op.Operation == meergo.OperationAddColumn {
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
					typ := typeToPostgresType(op.Type)
					b.WriteString("ADD COLUMN ")
					b.WriteString(quoteIdent(op.Column))
					b.WriteByte(' ')
					b.WriteString(typ)
				}
				queries = append(queries, b.String())
			}
		}
	}

	// CREATE VIEW "users".
	queries = append(queries, createViewQuery(usersTableName, columns, false))

	return queries
}

// createViewQuery returns the CREATE (OR REPLACE) VIEW query that creates the
// "users" view on the "users" table with the given name.
// userColumns contains the columns of such table.
// replace indicates if the query that creates the VIEW should have the "OR
// REPLACE" clause to replace the view if it already exist.
func createViewQuery(usersTableName string, userColumns []meergo.Column, replace bool) string {
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
		b.WriteString(",\n\t")
		b.WriteString(quoteIdent(c.Name))
	}
	b.WriteString("\nFROM ")
	b.WriteString(quoteIdent(usersTableName))
	return b.String()
}

// typeToPostgresType returns the PostgreSQL type corresponding to the given
// type.Type, which is a type supported in the user schema. These types are
// specified in the file 'core/datastore/README.md',
func typeToPostgresType(t types.Type) string {
	switch t.Kind() {
	case types.TextKind:
		var charLen int
		if l, ok := t.ByteLen(); ok {
			charLen = l // we represent N bytes len as N chars len in PostgreSQL.
		}
		if l, ok := t.CharLen(); ok {
			if charLen == 0 {
				charLen = l
			} else {
				charLen = min(l, charLen)
			}
		}
		if charLen > 0 {
			return "character varying(" + strconv.Itoa(charLen) + ")"
		}
		return "character varying"
	case types.BooleanKind:
		return "boolean"
	case types.IntKind:
		switch t.BitSize() {
		case 8:
			return "smallint"
		case 16:
			return "smallint"
		case 24:
			return "integer"
		case 32:
			return "integer"
		case 64:
			return "bigint"
		}
	case types.UintKind:
		switch t.BitSize() {
		case 8:
			return "smallint"
		case 16:
			return "integer"
		case 24:
			return "integer"
		case 32:
			return "bigint"
		case 64:
			return "numeric(20, 0)"
		}
	case types.FloatKind:
		switch t.BitSize() {
		case 32:
			return "real"
		case 64:
			return "double precision"
		}
	case types.DecimalKind:
		return fmt.Sprintf("numeric(%d, %d)", t.Precision(), t.Scale())
	case types.DateTimeKind:
		return "timestamp without time zone"
	case types.DateKind:
		return "date"
	case types.TimeKind:
		return "time without time zone"
	case types.YearKind:
		return "smallint"
	case types.UUIDKind:
		return "uuid"
	case types.JSONKind:
		return "jsonb"
	case types.InetKind:
		return "inet"
	case types.ArrayKind:
		typ := typeToPostgresType(t.Elem())
		return typ + "[]"
	case types.MapKind:
		return "jsonb"
	}
	panic(fmt.Sprintf("unexpected type %s", t))
}

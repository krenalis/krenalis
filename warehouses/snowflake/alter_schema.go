//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/types"
)

// AlterUserColumns alters the columns of the user tables.
func (warehouse *Snowflake) AlterUserColumns(ctx context.Context, opID string, userColumns []meergo.Column, operations []meergo.AlterOperation) error {
	status, err := warehouse.executeOperation(ctx, opID, alterUserColumns2)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		return status.executionError
	}
	err = warehouse.alterUserColumns(ctx, userColumns, operations)
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

func (warehouse *Snowflake) alterUserColumns(ctx context.Context, userColumns []meergo.Column, operations []meergo.AlterOperation) error {

	// Retrieve the current version of the "users" table.
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return err
	}

	// Determine the alter schema queries.
	queries := alterUserColumnsQueries("_USERS_"+strconv.Itoa(usersVersion), userColumns, operations)

	// Execute the alter schema queries within a transaction.
	err = warehouse.execTransaction(ctx, func(tx *sql.Tx) error {
		for _, query := range queries {
			_, err := tx.Exec(query)
			if err != nil {
				return snowflake(err)
			}
		}
		return nil
	})

	return err
}

// AlterUserColumnsQueries returns the queries that alter the columns of the
// user tables.
func (warehouse *Snowflake) AlterUserColumnsQueries(ctx context.Context, userColumns []meergo.Column, operations []meergo.AlterOperation) ([]string, error) {
	usersVersion, err := warehouse.usersVersion(ctx)
	if err != nil {
		return nil, err
	}
	queries := alterUserColumnsQueries("_USERS_"+strconv.Itoa(usersVersion), userColumns, operations)
	queries = append([]string{"BEGIN"}, queries...)
	queries = append(queries, "COMMIT")
	for i, q := range queries {
		queries[i] = q + ";"
	}
	return queries, nil
}

// alterUserColumnsQueries returns the queries that perform the given
// operations. usersTableName is the current name of the users table, for
// example "_users_42". operations must contain at least one operation.
func alterUserColumnsQueries(usersTableName string, userColumns []meergo.Column, operations []meergo.AlterOperation) []string {

	// The operations are performed in this order:
	//
	// (1) DROP VIEW.
	// (2) DROP columns.
	// (3) RENAME columns ?????
	// (4) ADD columns.
	// (5) CREATE VIEW.

	// Note that it is necessary to discard the view and rebuild it from
	// scratch, as the table needs to be altered and it would be impossible to
	// alter a column if there is a view that references it.

	var queries []string

	// DROP VIEW.
	queries = append(queries, `DROP VIEW "USERS"`)

	// ALTER TABLE ... DROP COLUMN.
	{
		var toDrop []string
		for _, op := range operations {
			if op.Operation == meergo.OperationDropColumn {
				toDrop = append(toDrop, op.Column)
			}
		}
		if len(toDrop) > 0 {
			for _, table := range []string{usersTableName, "_USER_IDENTITIES"} {
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
			queries = append(queries, `ALTER TABLE "_USER_IDENTITIES"`+"\n\tRENAME COLUMN "+quoteIdent(op.Column)+` TO `+quoteIdent(op.NewColumn))
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
			for _, table := range []string{usersTableName, "_USER_IDENTITIES"} {
				b := strings.Builder{}
				b.WriteString("ALTER TABLE " + quoteIdent(table) + "\n\t")
				for i, op := range toAdd {
					if i == 0 {
						b.WriteString("ADD COLUMN ")
					} else {
						b.WriteString(",\n\t           ")
					}
					typ := typeToSnowflakeType(op.Type)
					b.WriteString(quoteIdent(op.Column))
					b.WriteByte(' ')
					b.WriteString(typ)
				}
				queries = append(queries, b.String())
			}
		}
	}

	// CREATE VIEW "USERS".
	queries = append(queries, createViewQuery(usersTableName, userColumns, false))

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
	b.WriteString(`VIEW "USERS" AS SELECT` + "\n")
	metaProps := []string{"__ID__", "__LAST_CHANGE_TIME__"}
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

// typeToSnowflakeType returns the Snowflake type corresponding to the given
// type.Type, which is a type supported in the user schema. These types are
// specified in the file 'core/datastore/README.md',
func typeToSnowflakeType(t types.Type) string {
	switch t.Kind() {
	case types.BooleanKind:
		return "BOOLEAN"
	case types.IntKind:
		switch t.BitSize() {
		case 8:
			return "TINYINT" // Alias for "NUMBER(38, 0)".
		case 16:
			return "SMALLINT" // Alias for "NUMBER(38, 0)".
		case 24:
			return "NUMBER(7,0)"
		case 32:
			return "INT" // Alias for "NUMBER(38, 0)".
		case 64:
			return "BIGINT" // Alias for "NUMBER(38, 0)".
		}
	case types.UintKind:
		switch t.BitSize() {
		case 8:
			return "SMALLINT" // Alias for "NUMBER(38, 0)".
		case 16:
			return "INT" // Alias for "NUMBER(38, 0)".
		case 24:
			return "NUMBER(8,0)"
		case 32:
			return "BIGINT" // Alias for "NUMBER(38, 0)".
		case 64:
			return "NUMBER(20,0)"
		}
	case types.FloatKind:
		switch t.BitSize() {
		case 32:
			return "FLOAT"
		case 64:
			return "FLOAT"
		}
	case types.DecimalKind:
		return fmt.Sprintf("NUMBER(%d,%d)", t.Precision(), t.Scale())
	case types.DateTimeKind:
		return "TIMESTAMP_NTZ"
	case types.DateKind:
		return "DATE"
	case types.TimeKind:
		return "TIME"
	case types.YearKind:
		return "NUMBER(4,0)"
	case types.UUIDKind:
		return "VARCHAR"
	case types.JSONKind:
		return "VARIANT"
	case types.InetKind:
		return "VARCHAR"
	case types.TextKind:
		var charLen int
		if l, ok := t.ByteLen(); ok {
			charLen = l
		}
		if l, ok := t.CharLen(); ok {
			if charLen == 0 {
				charLen = l
			} else {
				charLen = min(l, charLen)
			}
		}
		if charLen > 0 {
			return "VARCHAR(" + strconv.Itoa(charLen) + ")"
		}
		return "VARCHAR"
	case types.ArrayKind:
		return "ARRAY"
	case types.MapKind:
		return "VARIANT"
	}
	panic(fmt.Sprintf("unexpected type %s", t))
}

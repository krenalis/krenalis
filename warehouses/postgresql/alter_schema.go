// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/jackc/pgx/v5"
)

// AlterProfileSchema alters the profile schema.
func (warehouse *PostgreSQL) AlterProfileSchema(ctx context.Context, opID string, columns []warehouses.Column, operations []warehouses.AlterOperation) error {
	status, err := warehouse.executeOperation(ctx, opID, alterProfileSchema)
	if err != nil {
		return err
	}
	if status.alreadyCompleted {
		return status.executionError
	}
	err = warehouse.alterProfileSchema(ctx, columns, operations)
	bo := backoff.New(200)
	bo.SetCap(time.Second)
	for bo.Next(ctx) {
		err2 := warehouse.setOperationAsCompleted(ctx, opID, err)
		if err2 != nil {
			slog.Error("cannot set alter profile columns operation as completed, retrying", "err", err2, "operationError", err)
			continue
		}
		if err != nil {
			return warehouses.NewOperationError(err)
		}
		return nil
	}
	return ctx.Err()
}

func (warehouse *PostgreSQL) alterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) error {

	// Retrieve the current version of the "meergo_profiles" table.
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return err
	}

	// Determine the alter schema queries.
	queries := alterProfileSchemaQueries("meergo_profiles_"+strconv.Itoa(profilesVersion), columns, operations)

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

// PreviewAlterProfileSchema provides a preview of an alter profile schema
// operation by returning the queries that would be executed on the warehouse to
// perform a given alter schema.
func (warehouse *PostgreSQL) PreviewAlterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) ([]string, error) {
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return nil, err
	}
	queries := alterProfileSchemaQueries("meergo_profiles_"+strconv.Itoa(profilesVersion), columns, operations)
	queries = append([]string{"BEGIN"}, queries...)
	queries = append(queries, "COMMIT")
	for i, q := range queries {
		queries[i] = q + ";"
	}
	return queries, nil
}

// alterProfileSchemaQueries returns the queries that perform the given
// operations. profilesTableName is the current name of the profiles table, for
// example "meergo_profiles_42". operations must contain at least one operation.
func alterProfileSchemaQueries(profilesTableName string, columns []warehouses.Column, operations []warehouses.AlterOperation) []string {

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
	queries = append(queries, `DROP VIEW "profiles"`)

	// ALTER TABLE ... DROP COLUMN.
	{
		var toDrop []string
		for _, op := range operations {
			if op.Operation == warehouses.OperationDropColumn {
				toDrop = append(toDrop, op.Column)
			}
		}
		if len(toDrop) > 0 {
			for _, table := range []string{profilesTableName, "meergo_identities"} {
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
		if op.Operation == warehouses.OperationRenameColumn {
			queries = append(queries, `ALTER TABLE `+quoteIdent(profilesTableName)+"\n\tRENAME COLUMN "+quoteIdent(op.Column)+` TO `+quoteIdent(op.NewColumn))
			queries = append(queries, `ALTER TABLE "meergo_identities"`+"\n\tRENAME COLUMN "+quoteIdent(op.Column)+` TO `+quoteIdent(op.NewColumn))
		}
	}

	// ALTER TABLE ... ADD COLUMN.
	{
		var toAdd []warehouses.AlterOperation
		for _, op := range operations {
			if op.Operation == warehouses.OperationAddColumn {
				toAdd = append(toAdd, op)
			}
		}
		if len(toAdd) > 0 {
			for _, table := range []string{profilesTableName, "meergo_identities"} {
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

	// CREATE VIEW "profiles".
	queries = append(queries, createViewQuery(profilesTableName, columns, false))

	return queries
}

// createViewQuery returns the CREATE (OR REPLACE) VIEW query that creates the
// "profiles" view on the "meergo_profiles" table with the given name.
// profileColumns contains the columns of such table.
// replace indicates if the query that creates the VIEW should have the "OR
// REPLACE" clause to replace the view if it already exists.
func createViewQuery(profilesTableName string, profileColumns []warehouses.Column, replace bool) string {
	b := strings.Builder{}
	b.WriteString(`CREATE `)
	if replace {
		b.WriteString(`OR REPLACE `)
	}
	b.WriteString(`VIEW "profiles" AS SELECT` + "\n")
	metaProps := []string{"_mpid", "_updated_at"}
	for i, p := range metaProps {
		if i > 0 {
			b.WriteString(",\n")
		}
		b.WriteString("\t\"")
		b.WriteString(p)
		b.WriteRune('"')
	}
	for _, c := range profileColumns {
		b.WriteString(",\n\t")
		b.WriteString(quoteIdent(c.Name))
	}
	b.WriteString("\nFROM ")
	b.WriteString(quoteIdent(profilesTableName))
	return b.String()
}

// typeToPostgresType returns the PostgreSQL type corresponding to the given
// type.Type, which is a type supported in the profile schema. These types are
// specified in the file 'core/datastore/README.md',
func typeToPostgresType(t types.Type) string {
	switch t.Kind() {
	case types.StringKind:
		var maxLength int
		if l, ok := t.MaxBytes(); ok {
			maxLength = l // we represent N bytes len as N chars len in PostgreSQL.
		}
		if l, ok := t.MaxLength(); ok {
			if maxLength == 0 {
				maxLength = l
			} else {
				maxLength = min(l, maxLength)
			}
		}
		if maxLength > 0 {
			return "character varying(" + strconv.Itoa(maxLength) + ")"
		}
		return "character varying"
	case types.BooleanKind:
		return "boolean"
	case types.IntKind:
		if t.IsUnsigned() {
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
		} else {
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
	case types.IPKind:
		return "inet"
	case types.ArrayKind:
		typ := typeToPostgresType(t.Elem())
		return typ + "[]"
	case types.MapKind:
		return "jsonb"
	}
	panic(fmt.Sprintf("unexpected type %s", t))
}

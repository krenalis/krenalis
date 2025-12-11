// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// AlterProfileSchema alters the profile schema.
func (warehouse *Snowflake) AlterProfileSchema(ctx context.Context, opID string, columns []warehouses.Column, operations []warehouses.AlterOperation) error {
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

func (warehouse *Snowflake) alterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) error {

	// Retrieve the current version of the "meergo_profiles" table.
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return err
	}

	// Determine the alter schema queries.
	queries := alterProfileSchemaQueries("MEERGO_PROFILES_"+strconv.Itoa(profilesVersion), columns, operations)

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

// PreviewAlterProfileSchema provides a preview of an alter profile schema
// operation by returning the queries that would be executed on the warehouse to
// perform a given alter schema.
func (warehouse *Snowflake) PreviewAlterProfileSchema(ctx context.Context, columns []warehouses.Column, operations []warehouses.AlterOperation) ([]string, error) {
	profilesVersion, err := warehouse.profilesVersion(ctx)
	if err != nil {
		return nil, err
	}
	queries := alterProfileSchemaQueries("MEERGO_PROFILES_"+strconv.Itoa(profilesVersion), columns, operations)
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
	// (3) RENAME columns ?????
	// (4) ADD columns.
	// (5) CREATE VIEW.

	// Note that it is necessary to discard the view and rebuild it from
	// scratch, as the table needs to be altered and it would be impossible to
	// alter a column if there is a view that references it.

	var queries []string

	// DROP VIEW.
	queries = append(queries, `DROP VIEW "PROFILES"`)

	// ALTER TABLE ... DROP COLUMN.
	{
		var toDrop []string
		for _, op := range operations {
			if op.Operation == warehouses.OperationDropColumn {
				toDrop = append(toDrop, op.Column)
			}
		}
		if len(toDrop) > 0 {
			for _, table := range []string{profilesTableName, "MEERGO_IDENTITIES"} {
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
			queries = append(queries, `ALTER TABLE "MEERGO_IDENTITIES"`+"\n\tRENAME COLUMN "+quoteIdent(op.Column)+` TO `+quoteIdent(op.NewColumn))
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
			for _, table := range []string{profilesTableName, "MEERGO_IDENTITIES"} {
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

	// CREATE VIEW "PROFILES".
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
	b.WriteString(`VIEW "PROFILES" AS SELECT` + "\n")
	metaProps := []string{"_MPID", "_UPDATED_AT"}
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

// typeToSnowflakeType returns the Snowflake type corresponding to the given
// type.Type, which is a type supported in the profile schema. These types are
// specified in the file 'core/datastore/README.md',
func typeToSnowflakeType(t types.Type) string {
	switch t.Kind() {
	case types.StringKind:
		var maxLength int
		if l, ok := t.MaxBytes(); ok {
			maxLength = l
		}
		if l, ok := t.MaxLength(); ok {
			if maxLength == 0 {
				maxLength = l
			} else {
				maxLength = min(l, maxLength)
			}
		}
		if maxLength > 0 {
			return "VARCHAR(" + strconv.Itoa(maxLength) + ")"
		}
		return "VARCHAR"
	case types.BooleanKind:
		return "BOOLEAN"
	case types.IntKind:
		if t.IsUnsigned() {
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
		} else {
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
	case types.IPKind:
		return "VARCHAR"
	case types.ArrayKind:
		return "ARRAY"
	case types.MapKind:
		return "VARIANT"
	}
	panic(fmt.Sprintf("unexpected type %s", t))
}

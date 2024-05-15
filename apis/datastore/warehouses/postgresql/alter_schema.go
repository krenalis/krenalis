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
	"math"
	"strconv"
	"strings"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/types"
)

// AlterSchema alters the users schema.
func (warehouse *PostgreSQL) AlterSchema(ctx context.Context, usersColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) error {
	queries, err := alterSchemaQueries(usersColumns, operations)
	if err != nil {
		return err
	}
	db, err := warehouse.connection()
	if err != nil {
		return err
	}
	err = db.Transaction(ctx, func(tx *postgres.Tx) error {
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
func (warehouse *PostgreSQL) AlterSchemaQueries(ctx context.Context, usersColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) ([]string, error) {
	queries, err := alterSchemaQueries(usersColumns, operations)
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

// addColumnClause returns the PostgreSQL clause "ADD COLUMN" for a column with
// the given type and nullable constraint.
func addColumnClause(column string, colType types.Type, nullable bool) (string, error) {
	var typ, defaultExpr string
	typ, defaultExpr, ok := typeToPostgresType(colType)
	if !ok {
		return "", fmt.Errorf("the type of the column %q is not supported by the PostgreSQL driver", column)
	}
	var additional string
	if !nullable {
		additional = " NOT NULL DEFAULT " + defaultExpr
	}
	return `ADD COLUMN "` + column + `" ` + typ + additional, nil
}

// alterSchemaQueries returns the queries that perform the given operations.
// operations must contain at least one operation.
func alterSchemaQueries(usersColumns []warehouses.Column, operations []warehouses.AlterSchemaOperation) ([]string, error) {

	// The operations are performed in this order:
	//
	// (1) DROP VIEW.
	// (2) DROP columns.
	// (3) RENAME columns (each in its own ALTER TABLE, see the PostgreSQL syntax for ALTER TABLE).
	// (4) ADD columns.
	// (5) CREATE VIEW.

	var queries []string

	// DROP VIEW.
	queries = append(queries, "DROP VIEW \"users\"")
	queries = append(queries, "DROP VIEW \"users_identities\"")

	// ALTER TABLE ... DROP COLUMN.
	{
		var toDrop []string
		for _, op := range operations {
			if op.Operation == warehouses.OperationDropColumn {
				toDrop = append(toDrop, op.Column)
			}
		}
		if len(toDrop) > 0 {
			for _, table := range []string{"_users", "_users_identities"} {
				b := strings.Builder{}
				b.WriteString("ALTER TABLE \"" + table + "\"\n\t")
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
			queries = append(queries, `ALTER TABLE "_users"`+"\n\tRENAME COLUMN \""+op.Column+`" TO "`+op.NewColumn+`"`)
			queries = append(queries, `ALTER TABLE "_users_identities"`+"\n\tRENAME COLUMN \""+op.Column+`" TO "`+op.NewColumn+`"`)
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
			for _, table := range []string{"_users", "_users_identities"} {
				b := strings.Builder{}
				b.WriteString("ALTER TABLE \"" + table + "\"\n\t")
				for i, op := range toAdd {
					if i > 0 {
						b.WriteString(",\n\t")
					}
					add, err := addColumnClause(op.Column, op.Type, op.Nullable)
					if err != nil {
						return nil, warehouses.UnsupportedAlterSchemaErr(err.Error())
					}
					b.WriteString(add)
				}
				queries = append(queries, b.String())
			}
		}
	}

	// CREATE VIEW "users".
	{
		b := strings.Builder{}
		b.WriteString(`CREATE VIEW "users" AS SELECT` + "\n")
		for i, c := range usersColumns {
			if i > 0 {
				b.WriteString(",\n")
			}
			b.WriteString("\t\"")
			b.WriteString(c.Name)
			b.WriteRune('"')
		}
		b.WriteString("\n" + `FROM "_users"`)
		queries = append(queries, b.String())
	}

	// CREATE VIEW "users_identities"
	{
		b := strings.Builder{}
		b.WriteString(`CREATE VIEW "users_identities" AS SELECT` + "\n")
		metaProps := []string{"__identity_key__", "__connection__", "__identity_id__",
			"__displayed_property__", "__anonymous_ids__", "__last_change_time__", "__gid__"}
		for i, p := range metaProps {
			if i > 0 {
				b.WriteString(",\n")
			}
			b.WriteString("\t\"")
			b.WriteString(p)
			b.WriteRune('"')
		}
		for _, c := range usersColumns {
			if c.Name == "__id__" {
				continue
			}
			b.WriteString(",\n\t\"")
			b.WriteString(c.Name)
			b.WriteRune('"')
		}
		b.WriteString("\n" + `FROM "_users_identities"`)
		queries = append(queries, b.String())
	}

	return queries, nil
}

func typeToPostgresType(t types.Type) (string, string, bool) {
	switch t.Kind() {
	case types.BooleanKind:
		return "boolean", "false", true
	case types.IntKind:
		min, max := t.IntRange()
		switch t.BitSize() {
		case 16:
			if min > types.MinInt16 || max < types.MaxInt16 {
				return "", "", false
			}
			return "smallint", "0", true
		case 32:
			if min > types.MinInt32 || max < types.MaxInt32 {
				return "", "", false
			}
			return "integer", "0", true
		case 64:
			if min > types.MinInt64 || max < types.MaxInt64 {
				return "", "", false
			}
			return "bigint", "0", true
		default:
			return "", "", false
		}
	case types.UintKind:
		return "", "", false
	case types.FloatKind:
		if t.IsReal() {
			return "", "", false
		}
		min, max := t.FloatRange()
		switch t.BitSize() {
		case 32:
			if min > -math.MaxFloat32 || max < math.MaxFloat32 {
				return "", "", false
			}
			return "real", "0", true
		case 64:
			if min > -math.MaxFloat64 || max < math.MaxFloat64 {
				return "", "", false
			}
			return "double precision", "0", true
		}
	case types.DecimalKind:
		// TODO(Gianluca): for decimal types specifying a minimum and a maximum
		// value, see https://github.com/open2b/chichi/issues/578.
		p := t.Precision()
		s := t.Scale()
		if p < 1 || p > 76 || s > 37 {
			return "", "", false
		}
		return fmt.Sprintf("decimal(%d, %d)", p, s), "0", true
	case types.DateTimeKind:
		return "timestamp without time zone", "'0001-01-01 00:00:00'", true
	case types.DateKind:
		return "date", "'0001-01-01'", true
	case types.TimeKind:
		return "time without time zone", "'00:00:00'", true
	case types.YearKind:
		return "", "", false
	case types.UUIDKind:
		return "uuid", "'00000000-0000-0000-0000-000000000000'", true
	case types.JSONKind:
		return "jsonb", "null", true
	case types.InetKind:
		return "", "", false
	case types.TextKind:
		if len(t.Values()) > 0 {
			return "", "", false
		}
		if t.Regexp() != nil {
			return "", "", false
		}
		if _, ok := t.ByteLen(); ok {
			return "", "", false
		}
		typ := "varchar"
		if l, ok := t.CharLen(); ok {
			typ += "(" + strconv.Itoa(l) + ")"
		}
		return typ, "''", true
	case types.ArrayKind:
		if t.MinItems() > 0 || t.MaxItems() < types.MaxItems {
			return "", "", false
		}
		if t.Elem().Kind() == types.ArrayKind {
			return "", "", false
		}
		typ, _, ok := typeToPostgresType(t.Elem())
		if !ok {
			return "", "", false
		}
		return typ + "[]", "'{}'", true
	case types.ObjectKind:
		return "", "", false
	case types.MapKind:
		return "", "", false
	}
	return "", "", false
}

// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2013-2017 Open2b
//

package sql

import (
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"chichi/pkg/open2b/decimal"

	"github.com/open2b/nuts"
)

const (
	Ignore int = iota
	Replace
)

var expressionReg = regexp.MustCompile(`{~?[1-9]*[0-9]}`)
var columnNameReg = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_\.]+$`)
var orderReg = regexp.MustCompile(`^(-)?(?:(.+)\.)?(.+)$`)
var tableExpressionReg = regexp.MustCompile(`^(\w+)(?:\s+[aA][sS]\s+(\w+))?$`)

// const columnLiteralReg   = regexp.MustCompile(`^(?:TRUE|FALSE|\d+(?:\.\d+)?)$`)
// const joinReg            = regexp.MustCompile(`^(?:inner|(?:full|left|right)\s+outer)$`)

type Table struct {
	connection Connection
	scheme     scheme
	statement  string
}

type Column []any
type Columns []any
type Group []any
type Having []any
type Join []any
type Inner struct {
	Table any
	On    On
}
type Left struct {
	Table any
	On    On
}
type Order []any
type OrderExpr struct {
	Expr []any
	Desc bool
}
type On map[string]any
type Right struct {
	Table any
	On    On
}
type Set map[string]any
type Value []any
type Where map[string]any

type Params struct {
	Columns  Columns
	Distinct bool
	Join     Join
	Where    Where
	Group    Group
	Having   Having
	Order    Columns
	Limit    int
	Offset   int
}

func newTable(connection Connection, scheme scheme) *Table {
	return &Table{connection, scheme, ""}
}

func (table *Table) Add(row map[string]any, onDuplicate any) (int, error) {

	var columns = nuts.SortedKeys(row)
	var values = make([]string, len(row))

	for i, column := range columns {
		values[i] = valueStatement(row[column])
		columns[i] = QuoteColumn(column)
	}

	var onDuplicateStatement string
	table.statement, onDuplicateStatement = table.onDuplicate(onDuplicate)

	table.statement += table.scheme.quotedName +
		" (" + strings.Join(columns, ",") + ") VALUES (" + strings.Join(values, ",") + ")" +
		onDuplicateStatement

	res, err := table.connection.Exec(table.statement)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (table *Table) Alter(alter []any) error {

	var values = make([]string, len(alter))
	for i, value := range alter {
		values[i] = valueStatement(value)
	}

	table.statement = "ALTER TABLE " + table.scheme.quotedName + "\n    " + strings.Join(values, ",\n")

	_, err := table.connection.Exec(table.statement)

	return err
}

func (table *Table) Columns() ([]string, error) {

	table.statement = "SHOW COLUMNS FROM " + table.scheme.quotedName

	rows, err := table.connection.Query(table.statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns = make([]string, 1)

	if rows.Next() {
		for rows.Next() {
			var column string
			err = rows.Scan(&column)
			if err != nil {
				return nil, err
			}
			columns = append(columns, column)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return columns, nil
}

func (table *Table) Count(where map[string]any) (int, error) {

	table.statement = "SELECT COUNT(*)\nFROM " + table.scheme.quotedName + WhereStatement(where)

	var count int
	err := table.connection.QueryRow(table.statement).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (table *Table) Delete(where Where) (int, error) {
	return table.deleteLimit(where, nil, 0)
}

func (table *Table) DeleteLimit(where Where, order []any, limit int) (int, error) {
	if limit <= 0 {
		panic("open2b/sql: Limit must be greater than zero")
	}
	return table.deleteLimit(where, order, limit)
}

func (table *Table) deleteLimit(where Where, order []any, limit int) (int, error) {

	var whereStatement = WhereStatement(where)

	var orderStatement = ""
	if len(order) > 0 {
		orderStatement = OrderStatement(order)
	}

	var limitStatement = ""
	if limit > 0 {
		limitStatement = "\nLIMIT " + strconv.Itoa(limit)
	}

	if whereStatement == "" && limitStatement == "" {
		table.statement = "TRUNCATE TABLE " + table.scheme.quotedName
	} else {
		table.statement = "DELETE FROM " + table.scheme.quotedName + whereStatement + limitStatement + orderStatement
	}

	res, err := table.connection.Exec(table.statement)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

func (table *Table) Drop() error {
	table.statement = "DROP TABLE " + table.scheme.quotedName
	_, err := table.connection.Exec(table.statement)
	return err
}

func (table *Table) Exists(where map[string]any) (bool, error) {

	table.statement = "SELECT 1\nFROM " + table.scheme.quotedName + WhereStatement(where) + "\nLIMIT 1"

	var exists bool
	err := table.connection.QueryRow(table.statement).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return exists, nil
}

func (table *Table) Get(where map[string]any, columns []any) (map[string]any, error) {

	columnsStatement := ColumnsStatement(columns)
	var query = NewQuery(table,
		"SELECT "+columnsStatement+"\nFROM "+table.scheme.quotedName+WhereStatement(where)+"\nLIMIT 1")

	var rows, err = query.Rows()
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, nil
	}

	return rows[0], nil
}

func (table *Table) Insert(columns []string, rows [][]any, onDuplicate any) (int, error) {

	var onDuplicateStatement string
	table.statement, onDuplicateStatement = table.onDuplicate(onDuplicate)

	var statement = make([]string, 0, 19+len(rows)*(len(columns)*2+2))
	statement = append(statement, table.statement, table.scheme.quotedName, " (")

	for _, column := range columns {
		statement = append(statement, QuoteColumn(column), ",")
	}
	if len(rows) == 1 {
		statement[len(statement)-1] = ") VALUES "
	} else {
		statement[len(statement)-1] = ") VALUES\n"
	}

	for _, row := range rows {
		statement = append(statement, "(")
		for _, value := range row {
			statement = append(statement, valueStatement(value), ",")
		}
		statement[len(statement)-1] = ")"
		statement = append(statement, ",\n")
	}
	statement[len(statement)-1] = onDuplicateStatement

	// trasforma lo statement in stringa
	table.statement = strings.Join(statement, "")

	res, err := table.connection.Exec(table.statement)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (table *Table) onDuplicate(expression any) (statement string, onDuplicate string) {

	statement = "INSERT INTO "

	if expression != nil {

		switch expr := expression.(type) {
		case int:
			switch expr {
			case Ignore:
				statement = "INSERT IGNORE INTO "
			case Replace:
				statement = "REPLACE "
			default:
				panic("open2b/sql: OnDuplicate must be 'Ignore', 'Replace' or a statement")
			}
		default:
			onDuplicate = "\nON DUPLICATE KEY UPDATE "
			for column, value := range expr.(Set) {
				onDuplicate += QuoteColumn(column) + " = " + valueStatement(value) + ","
			}
			onDuplicate = onDuplicate[:len(onDuplicate)-1]
		}

	}

	return
}

func GroupStatement(groups []any) string {

	if groups == nil {
		return ""
	}

	if len(groups) == 0 {
		panic("open2b/sql: Group slice can not be empty")
	}

	var groupStatement = "\nGROUP BY "

	for i, group := range groups {
		if groupString, ok := group.(string); ok {
			groupStatement += QuoteColumn(groupString)
		} else {
			panic(fmt.Sprintf("open2b/sql: Group element of type '%T' is not valid", group))
		}
		if i < len(groups)-1 {
			groupStatement += ", "
		}
	}

	return groupStatement
}

func OrderStatement(orders []any) string {

	if orders == nil {
		return ""
	}

	if len(orders) == 0 {
		panic("open2b/sql: Order slice can not be empty")
	}

	var orderStatement = "\nORDER BY "

	for i, order := range orders {
		if orderString, ok := order.(string); ok {
			if m := orderReg.FindStringSubmatch(orderString); m != nil {
				if m[2] != "" {
					orderStatement += QuoteColumn(m[2]) + "."
				}
				orderStatement += QuoteColumn(m[3])
				if m[1] != "" {
					orderStatement += " DESC"
				}
			} else {
				panic(fmt.Sprintf("open2b/sql: Order element %q is not valid", orderString))
			}
		} else if orderExpr, ok := order.(OrderExpr); ok {
			if len(orderExpr.Expr) == 0 {
				panic("open2b/sql: Order expression can not be an empty slice")
			}
			if expression, ok := orderExpr.Expr[0].(string); ok {
				if len(orderExpr.Expr) == 1 {
					orderStatement += expression
				} else {
					orderStatement += bindExpression(expression, orderExpr.Expr[1:])
				}
				if orderExpr.Desc {
					orderStatement += " DESC"
				}
			} else {
				panic("open2b/sql: First element of an order must be a string")
			}
		} else {
			panic(fmt.Sprintf("open2b/sql: Order element of type '%T' is not valid", order))
		}
		if i < len(orders)-1 {
			orderStatement += ", "
		}
	}

	return orderStatement
}

func (table *Table) Select(columns []any, where map[string]any, order []any, limit, first int) Query {
	return table._select(Params{Columns: columns, Where: where, Order: order, Limit: limit, Offset: first})
}

func (table *Table) SelectDistinct(columns []any, where map[string]any, order []any, limit, first int) Query {
	return table._select(Params{Columns: columns, Distinct: true, Where: where, Order: order, Limit: limit, Offset: first})
}

func (table *Table) SelectQuery(query Params) Query {
	return table._select(query)
}

func (table *Table) _select(query Params) Query {

	// SELECT
	var statement string
	if query.Distinct {
		statement = "SELECT DISTINCT "
	} else {
		statement = "SELECT "
	}
	statement += ColumnsStatement(query.Columns)

	// FROM
	statement += "\nFROM "
	if len(query.Join) > 0 {
		statement += FromStatement(table.scheme.quotedName, query.Join)
	} else {
		statement += table.scheme.quotedName
	}

	// WHERE
	statement += WhereStatement(query.Where)

	// GROUP BY
	if len(query.Group) > 0 {
		statement += GroupStatement(query.Group)
	}

	// ORDER BY
	if len(query.Order) > 0 {
		statement += OrderStatement(query.Order)
	}

	statement += LimitFirstStatement(query.Limit, query.Offset)

	return NewQuery(table, statement)
}

func (table *Table) SelectColumn(column any, where Where) ([]any, error) {

	var columnsStatement = ColumnsStatement([]any{column})

	table.statement = "SELECT " + columnsStatement + "\nFROM " + table.scheme.quotedName + WhereStatement(where)

	innerRows, err := table.connection.Query(table.statement)
	if err != nil {
		return nil, err
	}
	defer innerRows.Close()

	columns, err := innerRows.Columns()
	if err != nil {
		return nil, err
	}

	var innerValue any

	switch table.scheme.columns[columns[0]] {
	case 'b':
		var value bool
		innerValue = &value
	case 'i':
		var value int
		innerValue = &value
	case 'I':
		var value int64
		innerValue = &value
	default:
		var value string
		innerValue = &value
	}

	var values []any

	for innerRows.Next() {
		err = innerRows.Scan(innerValue)
		if err != nil {
			return nil, err
		}
		var value any
		switch val := innerValue.(type) {
		case *string:
			value = *val
		case *int:
			value = *val
		case *int64:
			value = *val
		case *bool:
			value = *val
		}
		values = append(values, value)
	}

	if err = innerRows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

func (table *Table) SelectIntColumn(column any, where Where) ([]int, error) {

	var columnsStatement = ColumnsStatement([]any{column})

	table.statement = "SELECT " + columnsStatement + "\nFROM " + table.scheme.quotedName + WhereStatement(where)

	var values []int

	err := table.connection.QueryScan(table.statement, func(rows *Rows) error {
		var value int
		for rows.Next() {
			if err := rows.Scan(&value); err != nil {
				return err
			}
			values = append(values, value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func (table *Table) SelectInt64Column(column any, where Where) ([]int64, error) {

	var columnsStatement = ColumnsStatement([]any{column})

	table.statement = "SELECT " + columnsStatement + "\nFROM " + table.scheme.quotedName + WhereStatement(where)

	var values []int64

	err := table.connection.QueryScan(table.statement, func(rows *Rows) error {
		var value int64
		for rows.Next() {
			if err := rows.Scan(&value); err != nil {
				return err
			}
			values = append(values, value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func (table *Table) SelectStringColumn(column any, where Where) ([]string, error) {

	var columnsStatement = ColumnsStatement([]any{column})

	table.statement = "SELECT " + columnsStatement + "\nFROM " + table.scheme.quotedName + WhereStatement(where)

	var values []string

	err := table.connection.QueryScan(table.statement, func(rows *Rows) error {
		var value string
		for rows.Next() {
			if err := rows.Scan(&value); err != nil {
				return err
			}
			values = append(values, value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return values, nil
}

func Map[K comparable, V any](table *Table, kColumn, vColumn any, where Where) (map[K]V, error) {
	var m map[K]V
	table.statement = "SELECT " + ColumnsStatement([]any{kColumn, vColumn}) +
		"\nFROM " + table.scheme.quotedName + WhereStatement(where)
	err := table.connection.QueryScan(table.statement, func(rows *Rows) error {
		m = map[K]V{}
		var k K
		var v V
		for rows.Next() {
			if err := rows.Scan(&k, &v); err != nil {
				return err
			}
			m[k] = v
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (table *Table) Statement() string {
	return table.statement
}

func (table *Table) SetStatement(statement string) {
	table.statement = statement
}

func (table *Table) QuotedName() string {
	return table.scheme.quotedName
}

func (table *Table) Update(set map[string]any, where map[string]any) (int, error) {

	table.statement = "UPDATE " + table.scheme.quotedName + "\nSET "

	for column, value := range set {
		table.statement += QuoteColumn(column) + " = " + valueStatement(value) + ","
	}
	table.statement = table.statement[:len(table.statement)-1] + WhereStatement(where)

	res, err := table.connection.Exec(table.statement)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

func bindExpression(expression string, parameters []any) string {
	return expressionReg.ReplaceAllStringFunc(expression, func(placeholder string) string {
		var value string
		if placeholder[1] == '~' {
			position, _ := strconv.Atoi(placeholder[2 : len(placeholder)-1])
			if position >= len(parameters) {
				panic(fmt.Sprintf("open2b/sql: Placeholder {~%d} does not exist as parameter", position))
			}
			value = fmt.Sprintf("%s", parameters[position])
		} else {
			position, _ := strconv.Atoi(placeholder[1 : len(placeholder)-1])
			if position >= len(parameters) {
				panic(fmt.Sprintf("open2b/sql: Placeholder {%d} does not exist as parameter", position))
			}
			value = Quote(parameters[position])
		}
		return value
	})
}

func FromStatement(table string, joinList Join) string {

	var statement = table

	for _, j := range joinList {

		var table any
		var on On

		if join, ok := j.(Left); ok {
			table = join.Table
			on = join.On
			statement += "\n\tLEFT JOIN "
		} else if join, ok := j.(Right); ok {
			table = join.Table
			on = join.On
			statement += "\n\tRIGHT JOIN "
		} else if join, ok := j.(Inner); ok {
			table = join.Table
			on = join.On
			statement += "\n\tINNER JOIN "
		} else {
			panic("open2b/sql: unsupported join type")
		}

		if t, ok := table.(*Table); ok {
			statement += t.QuotedName()
		} else if match := tableExpressionReg.FindStringSubmatch(table.(string)); match != nil {
			if match[2] == "" {
				statement += QuoteTable(match[0])
			} else {
				statement += QuoteTable(match[1]) + " AS `" + match[2] + "`"
			}
		}

		if len(on) > 0 {
			for onLeft, onRight := range on {
				statement += " ON " + QuoteColumn(onLeft) + " = " + QuoteColumn(onRight.(string))
			}
		}

	}

	return statement
}

func ColumnsStatement(columns Columns) string {

	if columns == nil {
		return "*"
	}

	if len(columns) == 0 {
		panic("open2b/sql: The list of columns is empty")
	}

	var statement string

	for _, column := range columns {
		if column == nil {
			statement += "NULL"
		} else {
			switch column := column.(type) {
			case string:
				if columnNameReg.MatchString(column) {
					statement += QuoteColumn(column)
				} else {
					statement += column
				}
			case int:
				statement += strconv.Itoa(column)
			case bool:
				if column {
					statement += "TRUE"
				} else {
					statement += "FALSE"
				}
			case Column:
				if len(column) == 0 {
					panic("open2b/sql: Column syntax is not valid")
				}
				if expression, ok := column[0].(string); ok {
					if strings.ContainsAny(expression, "{") {
						statement += bindExpression(expression, column[1:])
					} else {
						statement += expression
					}
				} else {
					panic("open2b/sql: Column syntax is not valid")
				}
			}
		}
		statement += ", "
	}

	return statement[:len(statement)-2]
}

// func (table *Table) fromStatement(tables []any) {

//     var fromStatement string

//     table = tables[0]
//     tables = tables[1:]

//     if table != nil {
//         switch table.(type) {
//         case string:
//             if m, err regexp.MatchString(tableExpressionReg, table.(string)); m {
//                 fromStatement = quoteTable(m[1])
//                 if len(m) > 2 {
//                     fromStatement += " AS `" + m[2]
//                 }
//             }
//         }
//     }
//     if fromStatement == "" {
//         panic("open2b/sql: TABLE_SYNTAX")
//     }

//     if len(tables) > 0 {
//         for i, tableReference := range tables {
//             var join, table, onLeft, onRight string
//             switch tableReference.(type) {
//             // case []*string:
//             //     if len(tableReference.([]string)) != 4 {
//             //         panic("open2b/sql: JOIN_SYNTAX")
//             //     }
//             //     join    := tableReference[0]
//             //     table   := tableReference[1]
//             //     onLeft  := tableReference[2]
//             //     onRight := tableReference[3]
//             //     if join == "" || ! joinReg.MatchString(join) {
//             //         panic("open2b/sql: JOIN_SYNTAX")
//             //     }
//             //     if onLeft == "" || onRight == "" {
//             //         if onLeft == "" || onLeft !~ /^[\w\.]+$/ ||  {
//             //             panic("open2b/sql: JOIN_SYNTAX")
//             //         }
//             //         die $self->dbException("JOIN_SYNTAX") if ! defined $onRight || $onRight !~ /^[\w\.]+$/;
//             //     }
//             //     $fromStatement .= "\n\t".uc($join)." JOIN ";
//             case string:
//                 table = tableReference
//                 fromStatement += ", "
//             default:
//                 panic("open2b/sql: JOIN_SYNTAX")
//             }
//             if table == "" {
//                 panic("open2b/sql: JOIN_SYNTAX")
//             }
//             if ref table == "Open2b::DB::mysql::Table" {
//                 fromStatement += table.scheme.quotedName
//             } else if ( $table =~ /^(\w+)(?:\s+[aA][sS]\s+(\w+))?$/ ) {
//                 $fromStatement .= $connection->QuoteTable($1).( defined $2 ? " AS `".$2."`" : "" );
//             } else {
//                 die $self->dbException("TABLE_SYNTAX");
//             }
//             if ( defined $onLeft ) {
//                 $fromStatement .= " ON ".$connection->QuoteColumn($onLeft)." = ".$connection->QuoteColumn($onRight);
//             }
//         }
//     }

//     return fromStatement
// }

func valueStatement(value any) string {

	var statement = ""

	if valueSlice, ok := value.(Value); ok {
		if len(valueSlice) == 0 {
			panic("open2b/sql: Value element can not be an empty slice")
		}
		if expression, ok := valueSlice[0].(string); ok {
			if len(valueSlice) == 1 {
				statement += expression
			} else {
				statement += bindExpression(expression, valueSlice[1:])
			}
		} else {
			panic("open2b/sql: First element of a value must be a string")
		}
	} else {
		statement = Quote(value)
	}

	return statement
}

func WhereStatement(expressions map[string]any) string {
	if len(expressions) == 0 {
		return ""
	}
	return "\nWHERE " + whereStatementOp(expressions, "AND")
}

func whereStatementOp(expressions map[string]any, op string) string {

	var where = make([]string, 0, len(expressions))
	var keys = nuts.Keys(expressions)
	sort.Strings(keys)

	for _, expression := range keys {

		if expression == "TRUE" || expression == "FALSE" {

			where = append(where, expression)

		} else if columnNameReg.MatchString(expression) {

			var value = expressions[expression]
			if value == nil {
				where = append(where, QuoteColumn(expression)+" IS NULL")
			} else {
				switch value.(type) {
				case string, bool, int, int32, int64, uint, uint64, uint32, float32, float64, *decimal.Dec, time.Time:
					where = append(where, QuoteColumn(expression)+" = "+Quote(value))
				case []int:
					var valueSlice = value.([]int)
					if len(valueSlice) == 0 {
						panic("open2b/sql: Expression value can not be an empty slice")
					} else if len(valueSlice) == 1 {
						where = append(where, QuoteColumn(expression)+" = "+strconv.FormatInt(int64(valueSlice[0]), 10))
					} else {
						var expr = QuoteColumn(expression) + " IN ("
						for _, val := range valueSlice {
							expr += strconv.FormatInt(int64(val), 10) + ","
						}
						where = append(where, expr[0:len(expr)-1]+")")
					}
				case []int64:
					var valueSlice = value.([]int64)
					if len(valueSlice) == 0 {
						panic("open2b/sql: Expression value can not be an empty slice")
					} else if len(valueSlice) == 1 {
						where = append(where, QuoteColumn(expression)+" = "+strconv.FormatInt(valueSlice[0], 10))
					} else {
						var expr = QuoteColumn(expression) + " IN ("
						for _, val := range valueSlice {
							expr += strconv.FormatInt(val, 10) + ","
						}
						where = append(where, expr[0:len(expr)-1]+")")
					}
				case []string:
					var valueSlice = value.([]string)
					if len(valueSlice) == 0 {
						panic("open2b/sql: Expression value can not be an empty slice")
					} else if len(valueSlice) == 1 {
						where = append(where, QuoteColumn(expression)+" = "+Quote(valueSlice[0]))
					} else {
						var expr = QuoteColumn(expression) + " IN ("
						for _, val := range valueSlice {
							expr += Quote(val) + ","
						}
						where = append(where, expr[0:len(expr)-1]+")")
					}
				default:
					if valueSlice, ok := value.([]any); ok {
						if len(valueSlice) == 0 {
							panic("open2b/sql: Expression value can not be an empty slice")
						} else if len(valueSlice) == 1 {
							where = append(where, QuoteColumn(expression)+" = "+Quote(valueSlice[0]))
						} else {
							var expr = QuoteColumn(expression) + " IN ("
							for _, val := range valueSlice {
								expr += Quote(val) + ","
							}
							where = append(where, expr[0:len(expr)-1]+")")
						}
					} else {
						panic(fmt.Sprintf("open2b/sql: Expression value of type '%T' is not allowed", value))
					}
				}
			}

		} else {

			var value = expressions[expression]
			if expression[0:3] == "OR " || expression[0:4] == "AND " {
				if value != nil {
					if orExpressions, ok := value.(map[string]any); ok {
						var op = "OR"
						if expression[:3] == "AND" {
							op = "AND"
						}
						where = append(where, "( "+whereStatementOp(orExpressions, op)+" )")
					} else {
						panic(fmt.Sprintf("open2b/sql: OR and AND expressions can not give a value of type '%T'", value))
					}
				}
			} else {
				if strings.Contains(expression, "{") {
					if valueSlice, ok := value.([]any); ok {
						where = append(where, bindExpression(expression, valueSlice))
					} else {
						where = append(where, bindExpression(expression, []any{value}))
					}
				} else {
					where = append(where, expression)
				}
			}

		}

	}

	return strings.Join(where, " "+op+" ")
}

func LimitFirstStatement(limit, first int) string {
	var statement = ""
	if limit > 0 {
		statement = "\nLIMIT "
		if first > 0 {
			statement += strconv.Itoa(first) + ", " + strconv.Itoa(limit)
		} else {
			statement += strconv.Itoa(limit)
		}
	} else if first > 0 {
		statement = "\nLIMIT " + strconv.Itoa(first) + ", 18446744073709551615"
	}
	return statement
}

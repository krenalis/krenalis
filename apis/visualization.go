//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"chichi/pkg/open2b/sql"
	o2bsql "chichi/pkg/open2b/sql"
)

type Visualization struct {
	*Properties
}

// ExecuteQuery executes the given JSON query.
func (visualization *Visualization) ExecuteQuery(ctx context.Context, jsonQuery JSONQuery) (columns []string, data [][]any, query string, err error) {

	// Generate the SQL query from the JSON query.
	query, columns, err = visualization.jsonQueryToSQLQuery(jsonQuery)
	if err != nil {
		return nil, nil, "", err
	}

	// Run the SQL query.
	data, err = visualization.runQuery(ctx, query)
	if err != nil {
		return nil, nil, "", err
	}

	return columns, data, query, nil
}

type JSONQuery struct {
	Graph     []string
	Filters   []Condition
	GroupBy   []string
	DateRange string
	DateFrom  string
	DateTo    string
}

type Condition struct {
	Field    string
	Operator string
	Value    string
	Domain   string `json:",omitempty"`
}

// jsonQueryToSQLQuery converts a JSON query into a SQL query. Also returns the
// columns.
func (visualization *Visualization) jsonQueryToSQLQuery(jq JSONQuery) (string, []string, error) {

	var (
		columns  []string
		wheres   []string
		groupBys []string
	)

	// Filters.
	for _, filter := range jq.Filters {
		wheres = append(wheres, conditionToSQL(filter))
	}

	// DateRange, DateFrom and DateTo.
	if jq.DateRange != "" && (jq.DateFrom != "" || jq.DateTo != "") {
		return "", nil, fmt.Errorf("cannot have both 'DateRange' and 'DateFrom'/'DateTo'")
	}
	if jq.DateRange != "" {
		var from, to time.Time
		now := time.Now()
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		switch jq.DateRange {
		case "Today":
			from = midnight
		case "Yesterday":
			to = midnight
			from = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
		case "Past7Days":
			from = time.Date(now.Year(), now.Month(), now.Day()-7, 0, 0, 0, 0, now.Location())
		case "Past31Days":
			from = time.Date(now.Year(), now.Month(), now.Day()-31, 0, 0, 0, 0, now.Location())
		case "Past12Months":
			from = time.Date(now.Year()-1, now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "":
			// Nothing to do.
		default:
			return "", nil, fmt.Errorf("date range %q not supported", jq.DateRange)
		}
		if to != (time.Time{}) {
			wheres = append(wheres, "`timestamp` <= "+timeToClickHouseDate(to))
		}
		if from != (time.Time{}) {
			wheres = append(wheres, "`timestamp` >= "+timeToClickHouseDate(from))
		}
	}
	if jq.DateFrom != "" {
		wheres = append(wheres, "`timestamp` >= '"+jq.DateFrom+"'")
	}
	if jq.DateTo != "" {
		wheres = append(wheres, "`timestamp` <= '"+jq.DateTo+"'")
	}

	// GroupBy.
	for _, groupBy := range jq.GroupBy {
		switch groupBy {
		case "Day":
			groupBys = append(groupBys, "toDayOfMonth(`timestamp`)")
			columns = append(columns, "toDayOfMonth(`timestamp`)")
		case "Month":
			groupBys = append(groupBys, "toMonth(`timestamp`)")
			columns = append(columns, "toMonth(`timestamp`)")
		case "Year":
			groupBys = append(groupBys, "toYear(`timestamp`)")
			columns = append(columns, "toYear(`timestamp`)")
		case "":
			return "", nil, fmt.Errorf("field GroupBy is mandatory")
		default:
			groupBys = append(groupBys, fmt.Sprintf("`%s`", groupBy))
			columns = append(columns, groupBy)
		}
	}

	// Graph.
	if len(jq.Graph) == 0 {
		return "", nil, errors.New("field Graph cannot be empty")
	}
	switch jq.Graph[0] {
	case "Count":
		if len(jq.Graph) <= 1 {
			return "", nil, errors.New("graph 'Count' requires at least parameter")
		}
		columns = append(columns, "COUNT(*)")
		switch jq.Graph[1] {
		case "Pageview":
			wheres = append(wheres, "`event` = 'view'")
		case "Click":
			wheres = append(wheres, "`event` = 'click'")
		case "Smart Event":
			where, ok, err := visualization.smartEvent(jq.Graph[2])
			if err != nil {
				return "", nil, err
			}
			if !ok {
				return "", nil, fmt.Errorf("the Smart Event %q does not exist", jq.Graph[2])
			}
			wheres = append(wheres, where)
		default:
			return "", nil, fmt.Errorf("%q not supported", jq.Graph[1])
		}
	case "Count Unique":
		if len(jq.Graph) <= 1 {
			return "", nil, errors.New("graph 'Count Unique' requires at least one parameter")
		}
		columns = append(columns, "COUNT(DISTINCT `user`)")
		switch jq.Graph[1] {
		case "Pageview":
			wheres = append(wheres, "`event` = 'view'")
		case "Click":
			wheres = append(wheres, "`event` = 'click'")
		case "Smart Event":
			where, ok, err := visualization.smartEvent(jq.Graph[2])
			if err != nil {
				return "", nil, err
			}
			if !ok {
				return "", nil, fmt.Errorf("the Smart Event %q does not exist", jq.Graph[2])
			}
			wheres = append(wheres, where)
		default:
			return "", nil, fmt.Errorf("%q not supported", jq.Graph[1])
		}
	default:
		return "", nil, fmt.Errorf("graph type %q not supported", jq.Graph[0])
	}

	query := "SELECT " + strings.Join(columns, ", ") +
		" FROM `chichi`.`events` " +
		" WHERE " + strings.Join(wheres, " AND ")

	if len(groupBys) > 0 {
		query += " " + " GROUP BY " + strings.Join(groupBys, ", ")
	}

	return query, columns, nil
}

// timeToClickHouseDate represents t in a datetime string compatible with
// ClickHouse.
func timeToClickHouseDate(t time.Time) string {
	return "'" + t.Format("2006-01-02 15:04:05") + "'"
}

// smartEvent returns the 'WHERE' expression relative to the Smart Event with
// the given name.
//
// If such name does not correspond to any Smart Event, this function returns
// false.
func (visualization *Visualization) smartEvent(which any) (string, bool, error) {
	smartEvents, err := visualization.SmartEvents.Find()
	if err != nil {
		return "", false, err
	}
	var smartEventName string
	var smartEventID int
	switch which := which.(type) {
	case string:
		smartEventName = which
	case int:
		smartEventID = which
	default:
		panic(fmt.Sprintf("unexpected type %T", which))
	}
	var smartEvent SmartEvent
	for _, event := range smartEvents {
		if (smartEventID > 0 && event.ID == smartEventID) ||
			(smartEventName != "" && event.Name == smartEventName) {
			smartEvent = event
			break
		}
	}
	if smartEvent.ID == 0 { // not found.
		return "", false, nil
	}
	where := smartEventToBooleanExpression(smartEvent)
	return where, true, nil
}

// smartEventToBooleanExpression returns the SQL boolean expression
// corresponding to the given Smart Event.
func smartEventToBooleanExpression(event SmartEvent) string {
	where := &strings.Builder{}
	switch event.Event {
	case "click":
		where.WriteString("`event` = 'click'")
	case "view":
		where.WriteString("`event` = 'view'")
	default:
		panic(fmt.Sprintf("unexpected %q", event.Event))
	}
	if len(event.Pages) > 0 {
		where.WriteString(" AND (")
		for i, page := range event.Pages {
			if i > 0 {
				where.WriteString(" OR ")
			}
			where.WriteString("(")
			where.WriteString(conditionToSQL(page))
			if page.Domain != "" {
				where.WriteString(" AND `url` = " + o2bsql.Quote(page.Domain))
			}
			where.WriteString(")")
		}
		where.WriteString(")")
	}
	if len(event.Buttons) > 0 {
		where.WriteString(" AND (")
		for i, button := range event.Buttons {
			if i > 0 {
				where.WriteString(" OR ")
			}
			where.WriteString("(" + conditionToSQL(button))
			if button.Domain != "" {
				where.WriteString(" AND `url` = " + o2bsql.Quote(button.Domain))
			}
			where.WriteString(")")
		}
		where.WriteString(")")
	}
	return where.String()
}

// conditionToSQL returns an SQL expression corresponding to the given
// Condition.
func conditionToSQL(condition Condition) string {
	// TODO(Gianluca): escape/check every value before putting it into the SQL.
	quotedField := sql.QuoteColumn(condition.Field)
	quotedValue := sql.Quote(condition.Value)
	switch condition.Operator {
	case "StartsWith":
		return fmt.Sprintf("ilike(%s, '%s%%')", quotedField, condition.Value)
	case "EndsWith":
		return fmt.Sprintf("ilike(%s, '%%%s')", quotedField, condition.Value)
	case "Contains":
		return fmt.Sprintf("ilike(%s, '%%%s%%')", quotedField, condition.Value)
	case "NotContains":
		return fmt.Sprintf("not ilike(%s, '%%%s%%')", quotedField, condition.Value)
	}
	where := quotedField
	switch condition.Operator {
	case "Equal":
		where += " = "
	case "NotEqual":
		where += " <> "
	case "GreaterThan":
		where += "> "
	case "GreaterEqualThan":
		where += ">= "
	case "LessThan":
		where += "> "
	case "LessEqualThan":
		where += "> "
	default:
		panic(fmt.Errorf("%q not supported", condition.Operator))
	}
	where += " " + quotedValue
	if condition.Domain != "" {
		where += fmt.Sprintf(" AND `url` = %s", sql.Quote(condition.Domain))
	}
	return where
}

// runQuery runs the given query and returns its results as a [][]any.
func (visualization *Visualization) runQuery(ctx context.Context, query string) ([][]any, error) {
	rows, err := visualization.chDB.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columnTypes := rows.ColumnTypes()
	columnsLen := len(columnTypes)
	result := [][]any{}
	for rows.Next() {
		sqlRow := make([]any, columnsLen)
		for j, column := range columnTypes {
			switch column.DatabaseTypeName() {
			case "DateTime":
				var value time.Time
				sqlRow[j] = &value
			case "String":
				var value string
				sqlRow[j] = &value
			case "UInt8":
				var value uint8
				sqlRow[j] = &value
			case "UInt16":
				var value uint16
				sqlRow[j] = &value
			case "UInt64":
				var value uint64
				sqlRow[j] = &value
			default:
				panic(fmt.Sprintf("BUG: handling of database type %q not implemented", column.DatabaseTypeName()))
			}
		}
		err := rows.Scan(sqlRow...)
		if err != nil {
			return nil, err
		}
		row := make([]any, len(sqlRow))
		for i, pr := range sqlRow {
			switch v := pr.(type) {
			case interface{ Value() (driver.Value, error) }:
				value, err := v.Value()
				if err != nil {
					panic(err)
				}
				row[i] = value
			case *time.Time:
				row[i] = (*v).String()
			case *string:
				row[i] = *v
			case *uint8:
				row[i] = *v
			case *uint16:
				row[i] = *v
			case *uint64:
				row[i] = *v
			default:
				panic("unexpected")
			}
		}
		result = append(result, row)
	}
	return result, nil
}

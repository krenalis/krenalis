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
	"fmt"
	"strings"
	"time"

	"chichi/pkg/open2b/sql"
)

type Visualization struct {
	*Properties
}

// ExecuteJSONQuery executes the given JSON query.
// Returns the columns of the executed query, the query results (as a [][]any)
// and the query itself as a string.
// If the given JSON query is invalid, returns an InvalidJSONQueryError error.
// If the JSON query refers to a Smart Event which cannot be found (for example
// it does not exist or it does not belong to the current property), a
// SmartEventNotFoundError error is returned.
func (visualization *Visualization) ExecuteJSONQuery(ctx context.Context, jsonQuery JSONQuery) (columns []string, data [][]any, query string, err error) {

	// Generate the SQL query from the JSON query.
	query, columns, err = visualization.jsonQueryToSQL(jsonQuery)
	if err != nil {
		return nil, nil, "", err
	}

	// Run the SQL query on ClickHouse.
	data, err = visualization.runClickHouseQuery(ctx, query)
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

// invalidJSONQuery returns an InvalidJSONQueryError error.
func invalidJSONQuery(format string, a ...any) error {
	return InvalidJSONQueryError(fmt.Sprintf(format, a...))
}

// InvalidJSONQueryError is an error that occurs in case of an invalid JSON
// query.
type InvalidJSONQueryError string

func (err InvalidJSONQueryError) Error() string {
	return fmt.Sprintf("invalid JSON query: %s", string(err))
}

// SmartEventNotFoundError is an error representing a Smart Event which has not
// been found.
type SmartEventNotFoundError string

func (err SmartEventNotFoundError) Error() string {
	return fmt.Sprintf("Smart Event %q not found", string(err))
}

// jsonQueryToSQL converts a JSON query into a SQL query. Also returns the
// columns. If the JSON query is not valid, returns an InvalidJSONQueryError.
// If the JSON query refers to a Smart Event which cannot be found (for example
// it does not exist or it does not belong to the current property), a
// SmartEventNotFoundError error is returned.
func (visualization *Visualization) jsonQueryToSQL(jq JSONQuery) (string, []string, error) {

	var (
		columns  []string
		wheres   []string
		groupBys []string
	)

	// Filters.
	for _, filter := range jq.Filters {
		where, err := conditionToSQL(filter)
		if err != nil {
			return "", nil, err
		}
		wheres = append(wheres, where)
	}

	// DateRange, DateFrom and DateTo.
	if jq.DateRange != "" && (jq.DateFrom != "" || jq.DateTo != "") {
		return "", nil, invalidJSONQuery("cannot have both 'DateRange' and 'DateFrom'/'DateTo'")
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
			return "", nil, invalidJSONQuery("date range %q not supported", jq.DateRange)
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
			return "", nil, invalidJSONQuery("field GroupBy is mandatory")
		default:
			quotedColumn := sql.QuoteColumn(groupBy)
			groupBys = append(groupBys, quotedColumn)
			columns = append(columns, quotedColumn)
		}
	}

	// Graph.
	if len(jq.Graph) == 0 {
		return "", nil, invalidJSONQuery("field Graph cannot be empty")
	}
	switch jq.Graph[0] {
	case "Count":
		if len(jq.Graph) <= 1 {
			return "", nil, invalidJSONQuery("graph 'Count' requires at least parameter")
		}
		columns = append(columns, "COUNT(*)")
		switch jq.Graph[1] {
		case "Pageview":
			wheres = append(wheres, "`event` = 'pageview'")
		case "Click":
			wheres = append(wheres, "`event` = 'click'")
		case "Smart Event":
			where, ok, err := visualization.smartEventToSQL(jq.Graph[2])
			if err != nil {
				return "", nil, err
			}
			if !ok {
				return "", nil, SmartEventNotFoundError(jq.Graph[2])
			}
			wheres = append(wheres, where)
		default:
			return "", nil, invalidJSONQuery("%q not supported", jq.Graph[1])
		}
	case "Count Unique":
		if len(jq.Graph) <= 1 {
			return "", nil, invalidJSONQuery("graph 'Count Unique' requires at least one parameter")
		}
		columns = append(columns, "COUNT(DISTINCT `user`)")
		switch jq.Graph[1] {
		case "Pageview":
			wheres = append(wheres, "`event` = 'pageview'")
		case "Click":
			wheres = append(wheres, "`event` = 'click'")
		case "Smart Event":
			where, ok, err := visualization.smartEventToSQL(jq.Graph[2])
			if err != nil {
				return "", nil, err
			}
			if !ok {
				return "", nil, SmartEventNotFoundError(jq.Graph[2])
			}
			wheres = append(wheres, where)
		default:
			return "", nil, invalidJSONQuery("%q not supported", jq.Graph[1])
		}
	default:
		return "", nil, invalidJSONQuery("graph type %q not supported", jq.Graph[0])
	}

	query := "SELECT " + strings.Join(columns, ", ") +
		" FROM `events` " +
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

// smartEventToSQL returns the SQL 'WHERE' expression relative to a Smart Event,
// which can be identified both by its identifier (which is an int) or by its
// name (which is a string).
//
// If the Smart Event cannot be found, this method returns "", false and nil.
func (visualization *Visualization) smartEventToSQL(which any) (string, bool, error) {
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
	where := &strings.Builder{}
	switch smartEvent.Event {
	case "click":
		where.WriteString("`event` = 'click'")
	case "pageview":
		where.WriteString("`event` = 'pageview'")
	default:
		panic(fmt.Sprintf("unexpected %q", smartEvent.Event))
	}
	if len(smartEvent.Pages) > 0 {
		where.WriteString(" AND (")
		for i, page := range smartEvent.Pages {
			if i > 0 {
				where.WriteString(" OR ")
			}
			where.WriteString("(")
			cond, err := conditionToSQL(page)
			if err != nil {
				panic(fmt.Sprintf("Smart Event contains an invalid condition: %s", err))
			}
			where.WriteString(cond)
			if page.Domain != "" {
				where.WriteString(" AND `domain` = " + sql.Quote(page.Domain))
			}
			where.WriteString(")")
		}
		where.WriteString(")")
	}
	if len(smartEvent.Buttons) > 0 {
		where.WriteString(" AND (")
		for i, button := range smartEvent.Buttons {
			if i > 0 {
				where.WriteString(" OR ")
			}
			cond, err := conditionToSQL(button)
			if err != nil {
				panic(fmt.Sprintf("Smart Event contains an invalid condition: %s", err))
			}
			where.WriteString("(" + cond)
			if button.Domain != "" {
				where.WriteString(" AND `domain` = " + sql.Quote(button.Domain))
			}
			where.WriteString(")")
		}
		where.WriteString(")")
	}
	return where.String(), true, nil
}

// conditionToSQL returns an SQL expression corresponding to the given
// Condition. Returns an InvalidJSONQueryError in case of an invalid condition.
func conditionToSQL(condition Condition) (string, error) {
	quotedField := sql.QuoteColumn(condition.Field)
	switch condition.Operator {
	case "StartsWith":
		return fmt.Sprintf("ilike(%s, '%s%%')", quotedField, condition.Value), nil
	case "EndsWith":
		return fmt.Sprintf("ilike(%s, '%%%s')", quotedField, condition.Value), nil
	case "Contains":
		return fmt.Sprintf("ilike(%s, '%%%s%%')", quotedField, condition.Value), nil
	case "NotContains":
		return fmt.Sprintf("not ilike(%s, '%%%s%%')", quotedField, condition.Value), nil
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
		where += "<= "
	default:
		return "", invalidJSONQuery("operator %q not supported", condition.Operator)
	}
	where += " " + sql.Quote(condition.Value)
	if condition.Domain != "" {
		where += fmt.Sprintf(" AND `domain` = %s", sql.Quote(condition.Domain))
	}
	return where, nil
}

// runClickHouseQuery runs the given query on the ClickHouse database and
// returns its results as a [][]any.
func (visualization *Visualization) runClickHouseQuery(ctx context.Context, query string) ([][]any, error) {
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

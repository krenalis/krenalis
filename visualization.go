//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func (server *Server) serveVisualization(w http.ResponseWriter, r *http.Request) {

	// Parse the request.
	var jsonQuery JSONQuery
	err := json.NewDecoder(r.Body).Decode(&jsonQuery)
	if err != nil {
		w.Header().Add("X-Error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Generate the SQL query from the JSON query.
	query, columns, err := jsonQueryToSQLQuery(jsonQuery)
	if err != nil {
		w.Header().Add("X-Error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Run the SQL query.
	data, err := server.runQuery(query)
	if err != nil {
		w.Header().Add("X-Error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot run query: %s", err)
		return
	}

	// Send the results to the client.
	w.Header().Add("Content-Type", "application/json")
	var response struct {
		Columns []string
		Data    [][]any
		Query   string
	}
	response.Columns = columns
	response.Data = data
	response.Query = query
	_ = json.NewEncoder(w).Encode(response)
}

type JSONQuery struct {
	Graph     []string
	Filters   []JSONQueryFilter
	GroupBy   []string
	DateRange string
	DateFrom  string
	DateTo    string
}

type JSONQueryFilter struct {
	Column     string
	Comparison string
	Target     string
}

// jsonQueryToSQLQuery converts a JSON query into a SQL query. Also returns the
// columns.
func jsonQueryToSQLQuery(jq JSONQuery) (string, []string, error) {

	var (
		columns  []string
		wheres   []string
		groupBys []string
	)

	// Filters.
	for _, filter := range jq.Filters {
		if filter.Comparison == "Contains" {
			where := fmt.Sprintf("ilike(`%s`, '%%%s%%')", filter.Column, filter.Target)
			wheres = append(wheres, where)
			continue
		}
		where := filter.Column
		switch filter.Comparison {
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
			return "", nil, fmt.Errorf("%q not supported", filter.Comparison)
		}
		where += " " + filter.Target
		wheres = append(wheres, where)
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
			from = time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location()) // TODO(Gianluca): check if "now.Day()-1" is correct.
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
		if len(jq.Graph) != 2 {
			return "", nil, errors.New("graph 'Count' requires one parameter")
		}
		columns = append(columns, "COUNT(*)")
		switch jq.Graph[1] {
		case "Pageview":
			wheres = append(wheres, "`event` = 'view'")
		case "Click":
			wheres = append(wheres, "`event` = 'click'")
		default:
			return "", nil, fmt.Errorf("%q not supported", jq.Graph[1])
		}
	case "Count Unique":
		if len(jq.Graph) != 2 {
			return "", nil, errors.New("graph 'Number of Sessions' requires one parameter")
		}
		columns = append(columns, "COUNT(DISTINCT `user`)")
		switch jq.Graph[1] {
		case "Pageview":
			wheres = append(wheres, "`event` = 'view'")
		case "Click":
			wheres = append(wheres, "`event` = 'click'")
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

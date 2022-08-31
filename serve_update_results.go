//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

func (server *Server) serveUpdateResults(w http.ResponseWriter, r *http.Request) {

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
	query, err := jsonQueryToSQLQuery(jsonQuery)
	if err != nil {
		w.Header().Add("X-Error", err.Error())
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Run the SQL query.
	result, err := server.runQuery(query)
	if err != nil {
		w.Header().Add("X-Error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot run query: %s", err)
		return
	}

	// Send the results to the client.
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("X-Query", query)
	_ = json.NewEncoder(w).Encode(result)
}

type JSONQuery struct {
	GraphOn   string
	Filters   []JSONQueryFilter
	GroupBy   string
	DateRange string
}

type JSONQueryFilter struct {
	Column     string
	Comparison string
	Target     string
}

// jsonQueryToSQLQuery converts a JSON query into a SQL query.
func jsonQueryToSQLQuery(jq JSONQuery) (string, error) {

	columns := []string{}
	wheres := []string{}

	switch jq.GraphOn {
	case "PageView":
		wheres = append(wheres, "`event` = 'view'")
	case "Click":
		wheres = append(wheres, "`event` = 'click'")
	default:
		return "", fmt.Errorf("%q not supported", jq.GraphOn)
	}

	for _, filter := range jq.Filters {
		where := filter.Column
		switch filter.Comparison {
		case "Equal":
			where += " = "
		case "NotEqual":
			where += " <> "
		default:
			return "", fmt.Errorf("%q not supported", filter.Comparison)
		}
		where += " " + filter.Target
		wheres = append(wheres, where)
	}

	// dateRange
	{
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
		case "Past12Months":
			from = time.Date(now.Year()-1, now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		case "":
			// Nothing to do.
		default:
			return "", fmt.Errorf("date range %q not supported", jq.DateRange)
		}
		if to != (time.Time{}) {
			wheres = append(wheres, "`timestamp` <= "+timeToClickHouseDate(to))
		}
		if from != (time.Time{}) {
			wheres = append(wheres, "`timestamp` >= "+timeToClickHouseDate(from))
		}
	}

	where := " WHERE " + strings.Join(wheres, " AND ")

	var groupBy string
	switch jq.GroupBy {
	case "":
		// Nothing to do.
	case "Day":
		groupBy = "GROUP BY toDayOfMonth(`timestamp`)"
		columns = []string{"toDayOfMonth(`timestamp`)", "COUNT(toDayOfMonth(`timestamp`))"}
	case "Month":
		groupBy = "GROUP BY toMonth(`timestamp`)"
		columns = []string{"toMonth(`timestamp`)", "COUNT(toMonth(`timestamp`))"}
	default:
		groupBy = fmt.Sprintf("GROUP BY `%s`", jq.GroupBy)
		columns = []string{jq.GroupBy, "COUNT(" + jq.GroupBy + ")"}
	}

	query := "SELECT " + strings.Join(columns, ", ") + " FROM `chichi`.`events` " + where + " " + groupBy
	return query, nil
}

// timeToClickHouseDate represents t in a datetime string compatible with
// ClickHouse.
func timeToClickHouseDate(t time.Time) string {
	return "'" + t.Format("2006-01-02 15:04:05") + "'"
}

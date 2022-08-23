//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)

	// Read the configuration file.
	settings, err := parseINIFile()
	if err != nil {
		log.Printf("[error] cannot read configuration file: %s", err)
		return
	}

	// Open a connection to the database.
	db, err := sql.Open("mysql", settings.DB.Username+":"+settings.DB.Password+"@"+settings.DB.Address+"/"+settings.DB.Database+
		"?clientFoundRows=true&charset=utf8mb4,utf8&parseTime=true&allowOldPasswords=true")
	if err != nil {
		log.Printf("[error] cannot connect to the database: %v", err)
		return
	}
	defer db.Close()

	// Serve the request as CGI.
	err = cgi.Serve(newServer(db))
	if err != nil {
		log.Fatal(err)
	}

}

type Server struct {
	db *sql.DB
}

func newServer(db *sql.DB) *Server {
	return &Server{db: db}
}

// ServeHTTP serves the POST requests.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rpath := r.URL.Path
	rpath = strings.TrimSuffix(rpath, "/")
	switch {

	case strings.HasSuffix(rpath, "/log-event"):
		var event *Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		err = server.logEvent(event)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("[error] cannot log event: %s", err)
			return
		}

	case strings.HasSuffix(rpath, "/run-query"):
		var query string
		err := json.NewDecoder(r.Body).Decode(&query)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		result, err := server.runQuery(query)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("[error] cannot log event: %s", err)
			return
		}
		_ = json.NewEncoder(w).Encode(result)

	default:
		http.NotFound(w, r)
		return

	}
}

// logEvent logs the given event on the database.
func (server *Server) logEvent(e *Event) error {
	query := "INSERT INTO `events`\n" +
		"(`timestamp`, `language`, `browser`, `url`, `referrer`, `target`, `event`, `text`, `title`, `session`)\n" +
		"VALUES\n" +
		"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err := server.db.Exec(query, e.Timestamp, e.Language, e.Browser, e.URL, e.Referrer, e.Target, e.Event, e.Text, e.Title, e.Session)
	return err
}

// runQuery runs the given query and returns its results as a [][]any.
func (server *Server) runQuery(query string) ([][]any, error) {
	rows, err := server.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	columnsLen := len(columnTypes)
	result := [][]any{}
	for rows.Next() {
		sqlRow := make([]any, columnsLen)
		for j, column := range columnTypes {
			switch column.DatabaseTypeName() {
			case "TINYINT", "INT":
				if _, nullable := column.Nullable(); nullable {
					var value sql.NullInt64
					sqlRow[j] = &value
				} else {
					var value int
					sqlRow[j] = &value
				}
			case "BIGINT":
				if _, nullable := column.Nullable(); nullable {
					var value sql.NullInt64
					sqlRow[j] = &value
				} else {
					var value int64
					sqlRow[j] = &value
				}
			case "DATE":
				var value time.Time
				sqlRow[j] = &value
			case "DATETIME":
				var value time.Time
				sqlRow[j] = &value
			default:
				if _, nullable := column.Nullable(); nullable {
					var value sql.NullString
					sqlRow[j] = &value
				} else {
					var value string
					sqlRow[j] = &value
				}
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
			}
		}
		result = append(result, row)
	}
	return result, nil
}

type Event struct {
	Browser   string
	Event     string // "visit", "click", ...
	Language  string // "it-IT"
	Referrer  string // "https://example.com"
	Session   string
	Target    string // "https://example.com"
	Text      string // "Add to cart"
	Timestamp time.Time
	Title     string // "Product X"
	URL       string // "https://example.com"
}

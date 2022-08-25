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
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

type Server struct {
	db *sql.DB
}

func newServer(db *sql.DB) *Server {
	return &Server{db: db}
}

func (server *Server) serveLogEvent(w http.ResponseWriter, r *http.Request) {
	var event *Event
	err := json.NewDecoder(r.Body).Decode(&event)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	query := "INSERT INTO `events`\n" +
		"(`timestamp`, `language`, `browser`, `url`, `referrer`, `target`, `event`, `text`, `title`, `session`)\n" +
		"VALUES\n" +
		"(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = server.db.Exec(query, event.Timestamp, event.Language, event.Browser, event.URL, event.Referrer, event.Target, event.Event, event.Text, event.Title, event.Session)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("[error] cannot log event: %s", err)
		return
	}
}

func (server *Server) serveRunQuery(w http.ResponseWriter, r *http.Request) {
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

func (server *Server) serveWithESBuild(w http.ResponseWriter, r *http.Request) {
	file, err := filepath.Abs("admin/src/index.js")
	if err != nil {
		panic(err)
	}
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{file},
		Format:            api.FormatESModule,
		JSXMode:           api.JSXModeAutomatic,
		Loader:            map[string]api.Loader{".js": api.LoaderJSX},
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            "out",
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             false,
	})
	if result.Errors != nil || result.Warnings != nil {
		printMessages := func(messages []api.Message) {
			for _, msg := range messages {
				log.Printf(" -> %v", msg)
			}
			log.Fatal("errors/warnings when executing esbuild, cannot proceed")
		}
		if result.Errors != nil {
			printMessages(result.Errors)
		}
		printMessages(result.Warnings)
	}
	base := path.Base(r.URL.Path)
	for _, out := range result.OutputFiles {
		if strings.HasSuffix(out.Path, base) {
			w.Header().Add("Content-Type", "text/javascript")
			w.Write(out.Contents)
			return
		}
	}
	http.NotFound(w, r)
}

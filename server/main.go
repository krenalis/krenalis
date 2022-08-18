//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/cgi"
	"os"

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
	err = cgi.Serve(newServer())
	if err != nil {
		log.Fatal(err)
	}

}

type Server struct{}

func newServer() *Server {

	return &Server{}
}

// ServeHTTP serves the POST requests.
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ...
}

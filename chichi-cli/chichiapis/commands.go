//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichiapis

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ProcessedEvent struct {
	Source int
	Server int
	Stream int
	Header *MessageHeader
	Data   []byte
	Err    string
}

type MessageHeader struct {
	ReceivedAt string
	RemoteAddr string
	Method     string
	Proto      string
	URL        string
	Headers    http.Header
}

// Connection represents a connection.
type Connection struct {
	ID   int
	Name string
}

func DisableConnection(connection int) {
	path := "api/connections/" + strconv.Itoa(connection) + "/status"
	err := callAPI("POST", path, strings.NewReader(`{"enabled":false}`), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func EnableConnection(connection int) {
	path := "api/connections/" + strconv.Itoa(connection) + "/status"
	err := callAPI("POST", path, strings.NewReader(`{"enabled":true}`), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func ListConnections() {
	var connections []*Connection
	err := callAPI("GET", "api/connections/", nil, &connections)
	if err != nil {
		log.Fatal(err)
	}
	for _, connection := range connections {
		fmt.Printf("%-10v %s\n", connection.ID, connection.Name)
	}
}

// Property represents a connection property.
type Property struct {
	Name  string
	Type  string
	Label string
}

func ListConnectionsProperties(connection int) {
	var properties []*Property
	err := callAPI("GET", "api/connections/"+strconv.Itoa(connection)+"/properties", nil, &properties)
	if err != nil {
		log.Fatal(err)
	}
	for _, property := range properties {
		fmt.Printf("%-50s %-40s %s\n", property.Label, property.Name, property.Type)
	}
}

func ImportUsersFromConnection(connection int, reimport bool) {
	path := "api/connections/" + strconv.Itoa(connection)
	if reimport {
		path += "/reimport"
	} else {
		path += "/import"
	}
	err := callAPI("POST", path, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func ExportUsersToConnection(connection int) {
	path := "api/connections/" + strconv.Itoa(connection) + "/export"
	err := callAPI("POST", path, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func AddEventListener(size, source, server, stream int) string {
	req := struct {
		Size   int `json:"size"`
		Source int `json:"source"`
		Server int `json:"server"`
		Stream int `json:"stream"`
	}{size, source, server, stream}
	var res struct {
		ID string
	}
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(req)
	err := callAPI("PUT", "api/event-listeners/", &b, &res)
	if err != nil {
		log.Fatal(err)
	}
	return res.ID
}

func Events(listener string) ([]ProcessedEvent, int) {
	var res struct {
		Events    []ProcessedEvent
		Discarded int
	}
	err := callAPI("GET", "api/event-listeners/"+url.PathEscape(listener)+"/events", nil, &res)
	if err != nil {
		log.Fatal(err)
	}
	return res.Events, res.Discarded
}

func RemoveEventListener(listener string) {
	err := callAPI("DELETE", "api/event-listeners/"+url.PathEscape(listener), nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceConnectWarehouse(typ string, settings []byte) {
	req := struct {
		Type     string
		Settings json.RawMessage
	}{typ, settings}
	b := &bytes.Buffer{}
	_ = json.NewEncoder(b).Encode(req)
	err := callAPI("POST", "api/workspace/connect-warehouse", b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceDisconnectWarehouse() {
	err := callAPI("POST", "api/workspace/disconnect-warehouse", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceInitWarehouse() {
	err := callAPI("POST", "api/workspace/init-warehouse", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceReloadSchemas() {
	err := callAPI("POST", "api/workspace/reload-schemas", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func Reload(connection int) error {
	err := callAPI("POST", "api/connections/"+strconv.Itoa(connection)+"/reload", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

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
	"os"
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

func getWorkspace() string {
	workspaceID := os.Getenv("CHICHI_CLI_WORKSPACE")
	if workspaceID == "" {
		log.Fatal("workspace id not found. Define the CHICHI_CLI_WORKSPACE environment variable")
	}
	return workspaceID
}

func DisableConnection(connection int) {
	path := "api/workspaces/" + getWorkspace() + "/connections/" + strconv.Itoa(connection) + "/status"
	err := callAPI("POST", path, strings.NewReader(`{"enabled":false}`), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func EnableConnection(connection int) {
	path := "api/workspaces/" + getWorkspace() + "/connections/" + strconv.Itoa(connection) + "/status"
	err := callAPI("POST", path, strings.NewReader(`{"enabled":true}`), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func ListConnections() {
	var connections []*Connection
	err := callAPI("GET", "api/workspaces/"+getWorkspace()+"/connections/", nil, &connections)
	if err != nil {
		log.Fatal(err)
	}
	for _, connection := range connections {
		fmt.Printf("%-10v %s\n", connection.ID, connection.Name)
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
	err := callAPI("PUT", "api/workspaces/"+getWorkspace()+"/event-listeners/", &b, &res)
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
	err := callAPI("GET", "api/workspaces/"+getWorkspace()+"/event-listeners/"+url.PathEscape(listener)+"/events", nil, &res)
	if err != nil {
		log.Fatal(err)
	}
	return res.Events, res.Discarded
}

func RemoveEventListener(listener string) {
	err := callAPI("DELETE", "api/workspaces/"+getWorkspace()+"/event-listeners/"+url.PathEscape(listener), nil, nil)
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
	err := callAPI("POST", "api/workspaces/"+getWorkspace()+"/connect-warehouse", b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceDisconnectWarehouse() {
	err := callAPI("POST", "api/workspaces/"+getWorkspace()+"/disconnect-warehouse", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceInitWarehouse() {
	err := callAPI("POST", "api/workspaces/"+getWorkspace()+"/init-warehouse", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspacePingWarehouse(typ string, settings []byte) {
	req := struct {
		Type     string
		Settings json.RawMessage
	}{typ, settings}
	b := &bytes.Buffer{}
	_ = json.NewEncoder(b).Encode(req)
	err := callAPI("POST", "api/workspaces/"+getWorkspace()+"/ping-warehouse", b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceReloadSchemas() {
	err := callAPI("POST", "api/workspaces/"+getWorkspace()+"/reload-schemas", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func Reload(connection int) error {
	err := callAPI("POST", "api/workspaces/"+getWorkspace()+"/connections/"+strconv.Itoa(connection)+"/reload", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

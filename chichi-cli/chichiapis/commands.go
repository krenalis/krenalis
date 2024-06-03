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

	"github.com/open2b/chichi/types"
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

func DisableConnection(workspace, connection int) {
	path := "api/workspaces/" + strconv.Itoa(workspace) + "/connections/" + strconv.Itoa(connection) + "/status"
	err := callAPI("POST", path, strings.NewReader(`{"enabled":false}`), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func EnableConnection(workspace, connection int) {
	path := "api/workspaces/" + strconv.Itoa(workspace) + "/connections/" + strconv.Itoa(connection) + "/status"
	err := callAPI("POST", path, strings.NewReader(`{"enabled":true}`), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func ListConnections(workspace int) {
	var connections []*Connection
	err := callAPI("GET", "api/workspaces/"+strconv.Itoa(workspace)+"/connections/", nil, &connections)
	if err != nil {
		log.Fatal(err)
	}
	for _, connection := range connections {
		fmt.Printf("%-10v %s\n", connection.ID, connection.Name)
	}
}

func AddEventListener(workspace, size, source, server, stream int) string {
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
	err := callAPI("PUT", "api/workspaces/"+strconv.Itoa(workspace)+"/event-listeners/", &b, &res)
	if err != nil {
		log.Fatal(err)
	}
	return res.ID
}

func Events(workspace int, listener string) ([]ProcessedEvent, int) {
	var res struct {
		Events    []ProcessedEvent
		Discarded int
	}
	err := callAPI("GET", "api/workspaces/"+strconv.Itoa(workspace)+"/event-listeners/"+url.PathEscape(listener)+"/events", nil, &res)
	if err != nil {
		log.Fatal(err)
	}
	return res.Events, res.Discarded
}

func RemoveEventListener(workspace int, listener string) {
	err := callAPI("DELETE", "api/workspaces/"+strconv.Itoa(workspace)+"/event-listeners/"+url.PathEscape(listener), nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceChangeUserSchema(workspace int, schema types.Type, rePaths map[string]any) {
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(req)
	err := callAPI("PUT", "api/workspaces/"+strconv.Itoa(workspace)+"/user-schema", &b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceChangeUserSchemaQueries(workspace int, schema types.Type, rePaths map[string]any) []string {
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var b bytes.Buffer
	_ = json.NewEncoder(&b).Encode(req)
	var resp struct {
		Queries []string
	}
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/change-user-schema-queries", &b, &resp)
	if err != nil {
		log.Fatal(err)
	}
	return resp.Queries
}

func WorkspaceConnectWarehouse(workspace int, typ string, settings []byte) {
	req := struct {
		Type     string
		Settings json.RawMessage
	}{typ, settings}
	b := &bytes.Buffer{}
	_ = json.NewEncoder(b).Encode(req)
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse", b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceDisconnectWarehouse(workspace int) {
	err := callAPI("DELETE", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceInitWarehouse(workspace int) {
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse/initializations", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func RunIdentityResolution(workspace int) {
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/identity-resolutions", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspacePingWarehouse(workspace int, typ string, settings []byte) {
	req := struct {
		Type     string
		Settings json.RawMessage
	}{typ, settings}
	b := &bytes.Buffer{}
	_ = json.NewEncoder(b).Encode(req)
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse/pings", b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergoapis

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

type ProcessedEvent struct {
	Source int            `json:"source"`
	Server int            `json:"server"`
	Stream int            `json:"stream"`
	Header *MessageHeader `json:"header"`
	Data   []byte         `json:"data"`
	Err    string         `json:"err"`
}

type MessageHeader struct {
	ReceivedAt string      `json:"receivedAt"`
	RemoteAddr string      `json:"remoteAddr"`
	Method     string      `json:"method"`
	Proto      string      `json:"proto"`
	URL        string      `json:"url"`
	Headers    http.Header `json:"headers"`
}

func CanChangeWarehouseSettings(workspace int, settings json.Value) {
	var b json.Buffer
	b.WriteString(`{"settings":`)
	b.Write(settings)
	b.WriteString(`}`)
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse/can-change-settings", &b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func CanInitializeWarehouse(typ string, settings json.Value) {
	var b json.Buffer
	b.WriteByte('{')
	_ = b.EncodeKeyValue("type", typ)
	_ = b.EncodeKeyValue("settings", settings)
	b.WriteString(`}`)
	err := callAPI("POST", "api/can-initialize-warehouse", &b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func ChangeWarehouseSettings(workspace int, mode string, settings json.Value) {
	var b json.Buffer
	b.WriteByte('{')
	_ = b.EncodeKeyValue("mode", mode)
	_ = b.EncodeKeyValue("settings", settings)
	b.WriteString(`}`)
	err := callAPI("PUT", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse/settings", &b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

// Connection represents a connection.
type Connection struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type PrivacyRegion string

const (
	PrivacyRegionNotSpecified PrivacyRegion = ""
	PrivacyRegionEurope       PrivacyRegion = "Europe"
)

func CreateWorkspace(name string, privacyRegion PrivacyRegion, userSchema types.Type, warehouseName string, warehouseSettings json.Value) int {
	var b json.Buffer
	b.WriteByte('{')
	_ = b.EncodeKeyValue("name", name)
	_ = b.EncodeKeyValue("privacyRegion", privacyRegion)
	_ = b.EncodeKeyValue("userSchema", userSchema)
	b.WriteString(`,"warehouse":{`)
	_ = b.EncodeKeyValue("name", warehouseName)
	_ = b.EncodeKeyValue("settings", warehouseSettings)
	b.WriteString(`}}`)
	var response struct {
		ID int `json:"id"`
	}
	err := callAPI("POST", "api/workspaces", &b, &response)
	if err != nil {
		log.Fatal(err)
	}
	return response.ID
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
	var b json.Buffer
	b.WriteByte('{')
	_ = b.EncodeKeyValue("size", size)
	_ = b.EncodeKeyValue("source", source)
	_ = b.EncodeKeyValue("server", server)
	_ = b.EncodeKeyValue("stream", stream)
	b.WriteString(`}`)
	var res struct {
		ID string `json:"id"`
	}
	err := callAPI("PUT", "api/workspaces/"+strconv.Itoa(workspace)+"/event-listeners/", &b, &res)
	if err != nil {
		log.Fatal(err)
	}
	return res.ID
}

func Events(workspace int, listener string) ([]ProcessedEvent, int) {
	var res struct {
		Events    []ProcessedEvent `json:"events"`
		Discarded int              `json:"discarded"`
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
	var b json.Buffer
	b.WriteByte('{')
	_ = b.EncodeKeyValue("schema", schema)
	_ = b.EncodeKeyValue("rePaths", rePaths)
	b.WriteString(`}`)
	err := callAPI("PUT", "api/workspaces/"+strconv.Itoa(workspace)+"/user-schema", &b, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func WorkspaceChangeUserSchemaQueries(workspace int, schema types.Type, rePaths map[string]any) []string {
	var b json.Buffer
	b.WriteByte('{')
	_ = b.EncodeKeyValue("schema", schema)
	_ = b.EncodeKeyValue("rePaths", rePaths)
	b.WriteString(`}`)
	var resp struct {
		Queries []string `json:"queries"`
	}
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/change-user-schema-queries", &b, &resp)
	if err != nil {
		log.Fatal(err)
	}
	return resp.Queries
}

func ResolveIdentities(workspace int) {
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/identity-resolutions", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func RepairWarehouse(workspace int) {
	err := callAPI("POST", "api/workspaces/"+strconv.Itoa(workspace)+"/warehouse/repair", nil, nil)
	if err != nil {
		log.Fatal(err)
	}
}

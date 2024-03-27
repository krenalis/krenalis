//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/backoff"
	"github.com/open2b/chichi/types"

	"github.com/segmentio/analytics-go/v3"
)

var defaultStrategy Strategy = "AB-C"

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) Action(connection, action int) Action {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/actions/" + strconv.Itoa(action)
	var response map[string]any
	c.MustCall("GET", url, nil, &response)
	data, err := json.Marshal(response)
	if err != nil {
		c.t.Fatal(err)
	}
	var a Action
	err = json.Unmarshal(data, &a)
	if err != nil {
		c.t.Fatal(err)
	}
	return a
}

func (c *Chichi) ActionSchemas(conn int, target apis.Target, eventType string) map[string]any {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(conn) + "/action-schemas/" + target.String()
	if eventType != "" {
		url += "/" + eventType
	}
	var schemas map[string]any
	c.MustCall("GET", url, nil, &schemas)
	return schemas
}

func (c *Chichi) AddAction(connection int, target string, action ActionToSet) int {
	switch target {
	case "Events", "Users", "Groups":
	default:
		panic(fmt.Sprintf("invalid target %q", target))
	}
	data := map[string]any{
		"Target": target,
		"Action": action,
	}
	var id int
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions", data, &id)
	return id
}

// AddActionErr is like AddAction but returns an error instead of panicking.
func (c *Chichi) AddActionErr(connection int, target string, action ActionToSet) (int, error) {
	switch target {
	case "Events", "Users", "Groups":
	default:
		panic(fmt.Sprintf("invalid target %q", target))
	}
	data := map[string]any{
		"Target": target,
		"Action": action,
	}
	var id int
	err := c.Call("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions", data, &id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Chichi) AddConnection(connection ConnectionToAdd) int {
	data := map[string]any{
		"Connection": connection,
	}
	var id int
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/add-connection", data, &id)
	return id
}

func (c *Chichi) AddDestinationFilesystem(storageDir string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "Filesystem",
		Role:      Destination,
		Enabled:   true,
		Connector: FilesystemConnector,
		Settings: JSONEncodeSettings(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Chichi) AddDestinationPostgreSQL() int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "PostgreSQL (destination)",
		Role:      Destination,
		Enabled:   true,
		Connector: PostgreSQLConnector,
		Settings: JSONEncodeSettings(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
		}),
	})
}

func (c *Chichi) AddDummy(name string, role Role, businessID string) int {
	conn := ConnectionToAdd{
		Name:       name,
		Role:       role,
		Enabled:    true,
		Connector:  DummyConnector,
		BusinessID: businessID,
		Settings:   []byte("{}"),
	}
	if role == Destination {
		mode := Cloud
		conn.SendingMode = &mode
	}
	return c.AddConnection(conn)
}

func (c *Chichi) AddJavaScriptSource(name, host string, eventConnections []int) int {
	return c.AddConnection(ConnectionToAdd{
		Name:             name,
		Role:             Source,
		Enabled:          true,
		Connector:        JavaScriptConnector,
		Strategy:         &defaultStrategy,
		WebsiteHost:      host,
		EventConnections: eventConnections,
	})
}

func (c *Chichi) AddSourceFilesystem(storageDir string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "Filesystem",
		Role:      Source,
		Enabled:   true,
		Connector: FilesystemConnector,
		Settings: JSONEncodeSettings(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Chichi) AddSourcePostgreSQL() int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "PostgreSQL (destination)",
		Role:      Source,
		Enabled:   true,
		Connector: PostgreSQLConnector,
		Settings: JSONEncodeSettings(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
		}),
	})
}

func (c *Chichi) ChangeUsersSchema(schema types.Type, rePaths map[string]any) {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/change-users-schema"
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	c.MustCall("POST", url, req, nil)
}

func (c *Chichi) ChangeUsersSchemaQueries(schema types.Type, rePaths map[string]any) []string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/change-users-schema-queries"
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	c.MustCall("POST", url, req, &response)
	return response.Queries
}

func (c *Chichi) CompletePath(storage int, path string) string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(storage) + "/complete-path/" + url.PathEscape(path)
	var response struct {
		Path string `json:"path"`
	}
	c.MustCall("GET", url, nil, &response)
	return response.Path
}

func (c *Chichi) ConnectionIdentities(connection int, first, limit int) ([]UserIdentity, int) {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/identities"
	req := map[string]any{
		"First": first,
		"Limit": limit,
	}
	var response struct {
		Count      int            `json:"count"`
		Identities []UserIdentity `json:"identities"`
	}
	c.MustCall("POST", url, req, &response)
	return response.Identities, response.Count
}

func (c *Chichi) ConnectionKeys(conn int) []string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(conn) + "/keys"
	var keys []string
	c.MustCall("GET", url, nil, &keys)
	return keys
}

func (c *Chichi) DeleteConnection(connection int) {
	method := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection)
	c.MustCall("DELETE", method, nil, nil)
}

func (c *Chichi) ExecuteAction(connection, action int, reimport bool) {
	method := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/actions/" + strconv.Itoa(action) + "/execute"
	c.MustCall("POST", method, map[string]any{"Reimport": reimport}, nil)
}

func (c *Chichi) Executions(connection int) []any {
	method := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/executions"
	var executions []any
	c.MustCall("GET", method, nil, &executions)
	return executions
}

func (c *Chichi) IdentifiersSchema() types.Type {
	var schema types.Type
	c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/identifiers-schema", nil, &schema)
	return schema
}

func (c *Chichi) Records(storage int, fileConnector int, path, sheet string, compression Compression, settings json.RawMessage, limit int) ([]map[string]any, types.Type) {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(storage) + "/records"
	req := map[string]any{
		"FileConnector": fileConnector,
		"Path":          path,
		"Sheet":         sheet,
		"Compression":   compression,
		"Settings":      settings,
		"Limit":         limit,
	}
	var response struct {
		Records []map[string]any `json:"records"`
		Schema  types.Type       `json:"schema"`
	}
	c.MustCall("POST", url, req, &response)
	return response.Records, response.Schema
}

func (c *Chichi) RunWorkspaceIdentityResolution() {
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/run-identity-resolution", nil, nil)
}

func (c *Chichi) SendEvent(writeKey string, message analytics.Message) {
	endpoint := "https://" + testsSettings.ChichiHost + "/" + "api"
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client, err := analytics.NewWithConfig(
		writeKey,
		analytics.Config{
			Endpoint:  endpoint,
			Transport: tr,
		},
	)
	if err != nil {
		c.t.Fatalf("cannot create client: %s", err)
	}
	err = client.Enqueue(message)
	if err != nil {
		c.t.Fatalf("cannot enqueue event: %s", err)
	}
	err = client.Close()
	if err != nil {
		c.t.Fatalf("cannot close client when sending events: %s", err)
	}
}

func (c *Chichi) SetWorkspaceIdentifiers(identifiers []string) {
	body := map[string]any{
		"Identifiers": identifiers,
	}
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/identifiers", body, nil)
}

func (c *Chichi) Sheets(storage int, fileConnector int, path string, compression Compression, settings json.RawMessage) []string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(storage) + "/sheets"
	req := map[string]any{
		"FileConnector": fileConnector,
		"Path":          path,
		"Compression":   compression,
		"Settings":      settings,
	}
	var request struct {
		Sheets []string `json:"sheets"`
	}
	c.MustCall("POST", url, req, &request)
	return request.Sheets
}

func (c *Chichi) TableSchema(connection int, table string) types.Type {
	var schema types.Type
	c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+
		strconv.Itoa(connection)+"/tables/"+url.PathEscape(table)+"/schema", nil, &schema)
	return schema
}

func (c *Chichi) UserEvents(user int) []map[string]any {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/users/" + strconv.Itoa(user) + "/events"
	var response struct {
		Events []map[string]any `json:"events"`
	}
	c.MustCall("GET", url, nil, &response)
	return response.Events
}

func (c *Chichi) UserIdentities(user int, first, limit int) ([]UserIdentity, int) {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/users/" + strconv.Itoa(user) + "/identities"
	req := map[string]any{
		"First": first,
		"Limit": limit,
	}
	var response struct {
		Count      int            `json:"count"`
		Identities []UserIdentity `json:"identities"`
	}
	c.MustCall("POST", url, req, &response)
	return response.Identities, response.Count
}

func (c *Chichi) Users(properties []string, order string, first, limit int) (users []map[string]any, schema types.Type, count int) {
	req := map[string]any{
		"Properties": properties,
		"Order":      order,
		"First":      first,
		"Limit":      limit,
	}
	var response struct {
		Users  []map[string]any `json:"users"`
		Count  int              `json:"count"`
		Schema types.Type       `json:"schema"`
	}
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/users", req, &response)
	return response.Users, response.Schema, response.Count
}

func (c *Chichi) WaitActionsToFinish(conn int) {
	time.Sleep(500 * time.Millisecond)
	for {
		stillRunning := false
		for _, exec := range c.Executions(conn) {
			e := exec.(map[string]any)
			// If the action execution ended with an error,
			// make the test fail.
			if err := e["Error"].(string); err != "" {
				actionID := string(e["Action"].(json.Number))
				connID := string(e["ID"].(json.Number))
				c.t.Fatalf("an error occurred when running action %q on connection %q: %s", actionID, connID, err)
			}
			if e["EndTime"] == nil {
				stillRunning = true
				break
			}
		}
		if stillRunning {
			time.Sleep(1 * time.Second)
			continue
		}
		return
	}
}

func (c *Chichi) WaitEventsStoredIntoWarehouse(ctx context.Context, expected int) {
	bo := backoff.New(20)
	bo.SetAttempts(10)
	bo.SetCap(2 * time.Second)
	bo.SetNextWaitTime(200 * time.Millisecond)
	for bo.Next(ctx) {
		count := c.CountEventsInWarehouse(ctx)
		if count == expected {
			break
		}
		c.t.Logf("[attempt %d] %d event(s) stored in warehouse until now", bo.Attempt(), count)
		if bo.WaitTime() == 0 {
			c.t.Fatalf("too many failed attempts")
		}
	}
}

func (c *Chichi) Workspace() Workspace {
	var ws Workspace
	c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(c.workspace), nil, &ws)
	return ws
}

func JSONEncodeSettings(settings map[string]any) []byte {
	data, err := json.Marshal(settings)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection settings to JSON: %s", err))
	}
	return data
}

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
	"time"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/backoff"
	"github.com/open2b/chichi/types"

	"github.com/segmentio/analytics-go/v3"
)

var defaultStrategy Strategy = "AB-C"

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) Action(connection, action int) Action {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/%d", c.ws, connection, action)
	var response map[string]any
	c.MustCall("GET", method, nil, &response)
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
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/schemas/%s", c.ws, conn, target)
	if eventType != "" {
		method += "/" + eventType
	}
	var schemas map[string]any
	c.MustCall("GET", method, nil, &schemas)
	return schemas
}

func (c *Chichi) AddAction(conn int, target string, action ActionToSet) int {
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
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions", c.ws, conn)
	c.MustCall("POST", method, data, &id)
	return id
}

// AddActionErr is like AddAction but returns an error instead of panicking.
func (c *Chichi) AddActionErr(conn int, target string, action ActionToSet) (int, error) {
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
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions", c.ws, conn)
	err := c.Call("POST", method, data, &id)
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
	method := fmt.Sprintf("/api/workspaces/%d/connections", c.ws)
	c.MustCall("POST", method, data, &id)
	return id
}

func (c *Chichi) AddDestinationFilesystem(storageDir string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "Filesystem",
		Role:      Destination,
		Enabled:   true,
		Connector: "Filesystem",
		UIValues: JSONEncodeUIValues(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Chichi) AddDestinationPostgreSQL() int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "PostgreSQL (destination)",
		Role:      Destination,
		Enabled:   true,
		Connector: "PostgreSQL",
		UIValues: JSONEncodeUIValues(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
		}),
	})
}

func (c *Chichi) AddDummy(name string, role Role) int {
	conn := ConnectionToAdd{
		Name:      name,
		Role:      role,
		Enabled:   true,
		Connector: "Dummy",
		UIValues:  []byte("{}"),
	}
	if role == Destination {
		mode := Cloud
		conn.SendingMode = &mode
	}
	return c.AddConnection(conn)
}

func (c *Chichi) AddDummyWithUIValues(name string, role Role, values map[string]any) int {
	conn := ConnectionToAdd{
		Name:      name,
		Role:      role,
		Enabled:   true,
		Connector: "Dummy",
		UIValues:  JSONEncodeUIValues(values),
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
		Connector:        "JavaScript",
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
		Connector: "Filesystem",
		UIValues: JSONEncodeUIValues(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Chichi) AddSourcePostgreSQL() int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "PostgreSQL (destination)",
		Role:      Source,
		Enabled:   true,
		Connector: "PostgreSQL",
		UIValues: JSONEncodeUIValues(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
		}),
	})
}

func (c *Chichi) ChangeUsersSchema(schema types.Type, rePaths map[string]any) {
	method := fmt.Sprintf("/api/workspaces/%d/user-schema", c.ws)
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	c.MustCall("PUT", method, req, nil)
}

// ChangeUsersSchemaErr is like ChangeUsersSchema but returns an error instead
// of panicking.
func (c *Chichi) ChangeUsersSchemaErr(schema types.Type, rePaths map[string]any) error {
	method := fmt.Sprintf("/api/workspaces/%d/user-schema", c.ws)
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	return c.Call("PUT", method, req, nil)
}

func (c *Chichi) ChangeUsersSchemaQueries(schema types.Type, rePaths map[string]any) []string {
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	method := fmt.Sprintf("/api/workspaces/%d/change-users-schema-queries", c.ws)
	c.MustCall("POST", method, req, &response)
	return response.Queries
}

// ChangeUsersSchemaQueriesErr is like ChangeUsersSchemaQueries but returns an
// error instead of panicking.
func (c *Chichi) ChangeUsersSchemaQueriesErr(schema types.Type, rePaths map[string]any) ([]string, error) {
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	method := fmt.Sprintf("/api/workspaces/%d/change-users-schema-queries", c.ws)
	err := c.Call("POST", method, req, &response)
	if err != nil {
		return nil, err
	}
	return response.Queries, nil
}

func (c *Chichi) CompletePath(storage int, path string) string {
	var response struct {
		Path string `json:"path"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/complete-path/%s", c.ws, storage, url.PathEscape(path))
	c.MustCall("GET", method, nil, &response)
	return response.Path
}

func (c *Chichi) ConnectionIdentities(conn, first, limit int) ([]UserIdentity, int) {
	req := map[string]any{
		"First": first,
		"Limit": limit,
	}
	var response struct {
		Count      int            `json:"count"`
		Identities []UserIdentity `json:"identities"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/identities", c.ws, conn)
	c.MustCall("POST", method, req, &response)
	return response.Identities, response.Count
}

func (c *Chichi) ConnectionKeys(conn int) []string {
	var keys []string
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/keys", c.ws, conn)
	c.MustCall("GET", method, nil, &keys)
	return keys
}

func (c *Chichi) DeleteConnection(conn int) {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d", c.ws, conn)
	c.MustCall("DELETE", method, nil, nil)
}

func (c *Chichi) ExecuteAction(conn, action int, reimport bool) {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/%d/executions", c.ws, conn, action)
	c.MustCall("POST", method, map[string]any{"Reimport": reimport}, nil)
}

func (c *Chichi) Executions(conn int) []any {
	var executions []any
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/executions", c.ws, conn)
	c.MustCall("GET", method, nil, &executions)
	return executions
}

func (c *Chichi) IdentifiersSchema() types.Type {
	var schema types.Type
	method := fmt.Sprintf("/api/workspaces/%d/identifiers-schema", c.ws)
	c.MustCall("GET", method, nil, &schema)
	return schema
}

func (c *Chichi) Records(storage int, fileConnector string, path, sheet string, compression Compression, uiValues json.RawMessage, limit int) ([]map[string]any, types.Type) {
	req := map[string]any{
		"FileConnector": fileConnector,
		"Path":          path,
		"Sheet":         sheet,
		"Compression":   compression,
		"UIValues":      uiValues,
		"Limit":         limit,
	}
	var response struct {
		Records []map[string]any `json:"records"`
		Schema  types.Type       `json:"schema"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/records", c.ws, storage)
	c.MustCall("POST", method, req, &response)
	return response.Records, response.Schema
}

func (c *Chichi) RunWorkspaceIdentityResolution() {
	method := fmt.Sprintf("/api/workspaces/%d/identity-resolutions", c.ws)
	c.MustCall("POST", method, nil, nil)
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

func (c *Chichi) GetConnectionUI(connection int) map[string]any {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/ui", c.ws, connection)
	var ui map[string]any
	c.MustCall("GET", method, nil, &ui)
	return ui
}

func (c *Chichi) SetAction(conn int, actionID int, action ActionToSet) {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/%d", c.ws, conn, actionID)
	c.MustCall("PUT", method, action, nil)
}

func (c *Chichi) SetWorkspaceIdentifiers(identifiers []string) {
	body := map[string]any{
		"Identifiers": identifiers,
	}
	method := fmt.Sprintf("/api/workspaces/%d/identifiers", c.ws)
	c.MustCall("PUT", method, body, nil)
}

func (c *Chichi) Sheets(storage int, fileConnector string, path string, compression Compression, uiValues json.RawMessage) []string {
	req := map[string]any{
		"FileConnector": fileConnector,
		"Path":          path,
		"Compression":   compression,
		"UIValues":      uiValues,
	}
	var request struct {
		Sheets []string `json:"sheets"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/sheets", c.ws, storage)
	c.MustCall("POST", method, req, &request)
	return request.Sheets
}

func (c *Chichi) TableSchema(conn int, table string) types.Type {
	var schema types.Type
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/tables/%s/schema", c.ws, conn, url.PathEscape(table))
	c.MustCall("GET", method, nil, &schema)
	return schema
}

func (c *Chichi) UserEvents(user int) []map[string]any {
	var response struct {
		Events []map[string]any `json:"events"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/users/%d/events", c.ws, user)
	c.MustCall("GET", method, nil, &response)
	return response.Events
}

func (c *Chichi) UserIdentities(user int, first, limit int) ([]UserIdentity, int) {
	var response struct {
		Count      int            `json:"count"`
		Identities []UserIdentity `json:"identities"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/users/%d/identities?first=%d&limit=%d", c.ws, user, first, limit)
	c.MustCall("GET", method, nil, &response)
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
	method := fmt.Sprintf("/api/workspaces/%d/users", c.ws)
	c.MustCall("POST", method, req, &response)
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
	method := fmt.Sprintf("/api/workspaces/%d", c.ws)
	c.MustCall("GET", method, nil, &ws)
	return ws
}

func JSONEncodeUIValues(values map[string]any) []byte {
	data, err := json.Marshal(values)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection UI values to JSON: %s", err))
	}
	return data
}

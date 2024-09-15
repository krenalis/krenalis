//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/meergo/meergo/apis"
	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/open2b/analytics-go"
)

var defaultStrategy Strategy = "AB-C"

// This file contains support methods which reduce verbosity of tests.

func (c *Meergo) Action(connection, action int) Action {
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

func (c *Meergo) ActionSchemas(conn int, target apis.Target, eventType string) map[string]any {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/schemas/%s", c.ws, conn, target)
	if eventType != "" {
		method += "/" + eventType
	}
	var schemas map[string]any
	c.MustCall("GET", method, nil, &schemas)
	return schemas
}

func (c *Meergo) AddAction(conn int, target string, action ActionToSet) int {
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

func (c *Meergo) AddEventAction(conn int, eventType string, action ActionToSet) int {
	data := map[string]any{
		"Target":    "Events",
		"EventType": eventType,
		"Action":    action,
	}
	var id int
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions", c.ws, conn)
	c.MustCall("POST", method, data, &id)
	return id
}

// AddActionErr is like AddAction but returns an error instead of panicking.
func (c *Meergo) AddActionErr(conn int, target string, action ActionToSet) (int, error) {
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

func (c *Meergo) AddConnection(connection ConnectionToAdd) int {
	data := map[string]any{
		"Connection": connection,
	}
	var id int
	method := fmt.Sprintf("/api/workspaces/%d/connections", c.ws)
	c.MustCall("POST", method, data, &id)
	return id
}

func (c *Meergo) AddDestinationFilesystem(storageDir string) int {
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

func (c *Meergo) AddDestinationPostgreSQL() int {
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

func (c *Meergo) AddDummy(name string, role Role) int {
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

func (c *Meergo) AddDummyWithSettings(name string, role Role, settings DummySettings) int {
	conn := ConnectionToAdd{
		Name:      name,
		Role:      role,
		Enabled:   true,
		Connector: "Dummy",
		UIValues:  JSONEncodeUIValues(settings),
	}
	if role == Destination {
		mode := Cloud
		conn.SendingMode = &mode
	}
	return c.AddConnection(conn)
}

func (c *Meergo) AddJavaScriptSource(name, host string, eventConnections []int) int {
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

func (c *Meergo) AddSourceFilesystem(storageDir string) int {
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

func (c *Meergo) AddSourcePostgreSQL() int {
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

func (c *Meergo) IdentityResolutionExecution() (startTime, endTime *time.Time) {
	var response struct {
		StartTime *time.Time `json:"startTime"`
		EndTime   *time.Time `json:"endTime"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/identity-resolution/execution", c.ws)
	c.MustCall("GET", method, nil, &response)
	return response.StartTime, response.EndTime
}

func (c *Meergo) ChangeIdentityResolutionSettings(runOnBatchImport bool, identifiers []string) {
	body := map[string]any{
		"RunOnBatchImport": runOnBatchImport,
		"Identifiers":      identifiers,
	}
	method := fmt.Sprintf("/api/workspaces/%d/identity-resolution/settings", c.ws)
	c.MustCall("PUT", method, body, nil)
}

func (c *Meergo) ChangeIdentityResolutionSettingsErr(identifiers []string) error {
	body := map[string]any{
		"Identifiers": identifiers,
	}
	method := fmt.Sprintf("/api/workspaces/%d/identity-resolution/settings", c.ws)
	return c.Call("PUT", method, body, nil)
}

func (c *Meergo) ChangeUserSchema(schema types.Type, primarySources map[string]int, rePaths map[string]any) {
	method := fmt.Sprintf("/api/workspaces/%d/user-schema", c.ws)
	req := map[string]any{
		"Schema":         schema,
		"PrimarySources": primarySources,
		"RePaths":        rePaths,
	}
	c.MustCall("PUT", method, req, nil)
}

// ChangeUserSchemaErr is like ChangeUserSchema but returns an error instead of
// panicking.
func (c *Meergo) ChangeUserSchemaErr(schema types.Type, primarySources map[string]int, rePaths map[string]any) error {
	method := fmt.Sprintf("/api/workspaces/%d/user-schema", c.ws)
	req := map[string]any{
		"Schema":         schema,
		"PrimarySources": primarySources,
		"RePaths":        rePaths,
	}
	return c.Call("PUT", method, req, nil)
}

func (c *Meergo) ChangeUserSchemaQueries(schema types.Type, rePaths map[string]any) []string {
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	method := fmt.Sprintf("/api/workspaces/%d/change-user-schema-queries", c.ws)
	c.MustCall("POST", method, req, &response)
	return response.Queries
}

// ChangeUserSchemaQueriesErr is like ChangeUserSchemaQueries but returns an
// error instead of panicking.
func (c *Meergo) ChangeUserSchemaQueriesErr(schema types.Type, rePaths map[string]any) ([]string, error) {
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	method := fmt.Sprintf("/api/workspaces/%d/change-user-schema-queries", c.ws)
	err := c.Call("POST", method, req, &response)
	if err != nil {
		return nil, err
	}
	return response.Queries, nil
}

func (c *Meergo) CompletePath(storage int, path string) string {
	var response struct {
		Path string `json:"path"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/complete-path/%s", c.ws, storage, url.PathEscape(path))
	c.MustCall("GET", method, nil, &response)
	return response.Path
}

func (c *Meergo) ConnectionIdentities(conn, first, limit int) ([]UserIdentity, int) {
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

func (c *Meergo) ConnectionKeys(conn int) []string {
	var keys []string
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/keys", c.ws, conn)
	c.MustCall("GET", method, nil, &keys)
	return keys
}

// ConnectWarehouse connects the warehouse specified in the tests configuration.
func (c *Meergo) ConnectWarehouse(behavior ConnectWarehouseBehavior) {
	req := map[string]any{
		"Type":     testsSettings.WarehouseType,
		"Settings": testsSettings.Warehouse,
		"Behavior": behavior,
	}
	method := fmt.Sprintf("/api/workspaces/%d/warehouse", c.ws)
	c.MustCall("POST", method, req, nil)
}

func (c *Meergo) DeleteConnection(conn int) {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d", c.ws, conn)
	c.MustCall("DELETE", method, nil, nil)
}

func (c *Meergo) DisconnectWarehouse() {
	method := fmt.Sprintf("/api/workspaces/%d/warehouse", c.ws)
	c.MustCall("DELETE", method, nil, nil)
}

func (c *Meergo) ExecuteAction(conn, action int, reload bool) {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/%d/executions", c.ws, conn, action)
	c.MustCall("POST", method, map[string]any{"Reload": reload}, nil)
}

func (c *Meergo) Executions(conn int) []Execution {
	var executions []Execution
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/executions", c.ws, conn)
	c.MustCall("GET", method, nil, &executions)
	return executions
}

func (c *Meergo) IdentifiersSchema() types.Type {
	var schema types.Type
	method := fmt.Sprintf("/api/workspaces/%d/identifiers-schema", c.ws)
	c.MustCall("GET", method, nil, &schema)
	return schema
}

func (c *Meergo) Records(storage int, fileConnector string, path, sheet string, compression Compression, uiValues json.RawMessage, limit int) ([]map[string]any, types.Type) {
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

func (c *Meergo) ResolveIdentities() {
	method := fmt.Sprintf("/api/workspaces/%d/identity-resolutions", c.ws)
	c.MustCall("POST", method, nil, nil)
}

func (c *Meergo) SendEvent(writeKey string, message analytics.Message) {
	endpoint := "https://" + testsSettings.MeergoHost + "/" + "api"
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

func (c *Meergo) GetConnectionUI(connection int) map[string]any {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/ui", c.ws, connection)
	var ui map[string]any
	c.MustCall("GET", method, nil, &ui)
	return ui
}

func (c *Meergo) SetAction(conn int, actionID int, action ActionToSet) {
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/actions/%d", c.ws, conn, actionID)
	c.MustCall("PUT", method, action, nil)
}

func (c *Meergo) Sheets(storage int, fileConnector string, path string, compression Compression, uiValues json.RawMessage) []string {
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

func (c *Meergo) TableSchema(conn int, table string) types.Type {
	var schema types.Type
	method := fmt.Sprintf("/api/workspaces/%d/connections/%d/tables/%s/schema", c.ws, conn, url.PathEscape(table))
	c.MustCall("GET", method, nil, &schema)
	return schema
}

func (c *Meergo) UserEvents(user uuid.UUID) []map[string]any {
	var response struct {
		Events []map[string]any `json:"events"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/users/%s/events", c.ws, user)
	c.MustCall("GET", method, nil, &response)
	return response.Events
}

func (c *Meergo) UserIdentities(user uuid.UUID, first, limit int) ([]UserIdentity, int) {
	var response struct {
		Count      int            `json:"count"`
		Identities []UserIdentity `json:"identities"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/users/%s/identities?first=%d&limit=%d", c.ws, user, first, limit)
	c.MustCall("GET", method, nil, &response)
	return response.Identities, response.Count
}

func (c *Meergo) Users(properties []string, order string, orderDesc bool, first, limit int) (users []User, schema types.Type, count int) {
	req := map[string]any{
		"Properties": properties,
		"Order":      order,
		"OrderDesc":  orderDesc,
		"First":      first,
		"Limit":      limit,
	}
	var response struct {
		Users  []User     `json:"users"`
		Count  int        `json:"count"`
		Schema types.Type `json:"schema"`
	}
	method := fmt.Sprintf("/api/workspaces/%d/users", c.ws)
	c.MustCall("POST", method, req, &response)
	return response.Users, response.Schema, response.Count
}

func (c *Meergo) WaitActionsToFinish(conn int) {
	time.Sleep(500 * time.Millisecond)
	for {
		stillRunning := false
		for _, exec := range c.Executions(conn) {
			// If the action execution ended with an error,
			// make the test fail.
			if exec.Error != "" {
				c.t.Fatalf("an error occurred when running action %q on connection %q: %s", exec.Action, exec.ID, exec.Error)
			}
			if exec.EndTime == nil {
				stillRunning = true
				break
			}
			if exec.Failed > 0 {
				c.t.Fatalf("an error occurred when running action %q on connection %q: %d failed", exec.Action, exec.ID, exec.Failed)
			}
		}
		if stillRunning {
			time.Sleep(1 * time.Second)
			continue
		}
		return
	}
}

func (c *Meergo) WaitEventsStoredIntoWarehouse(ctx context.Context, expected int) {
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

func (c *Meergo) Workspace() Workspace {
	var ws Workspace
	method := fmt.Sprintf("/api/workspaces/%d", c.ws)
	c.MustCall("GET", method, nil, &ws)
	return ws
}

func UIJSONProperties(properties map[string]bool) []byte {
	var uiValues = struct {
		Properties []KV
	}{
		Properties: []KV{},
	}
	for name, required := range properties {
		kv := KV{Key: name}
		if required {
			kv.Value = "t"
		} else {
			kv.Value = "f"
		}
		uiValues.Properties = append(uiValues.Properties, kv)
	}
	return JSONEncodeUIValues(uiValues)
}

func JSONEncodeUIValues(values any) []byte {
	data, err := json.Marshal(values)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection UI values to JSON: %s", err))
	}
	return data
}

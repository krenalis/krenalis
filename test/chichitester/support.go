//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"chichi/apis"
	"chichi/backoff"
	"chichi/connector/types"

	"github.com/segmentio/analytics-go/v3"
)

var defaultStrategy Strategy = "AB-C"

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) Action(connection, action int) Action {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/actions/" + strconv.Itoa(action)
	response := c.MustCall("GET", url, nil).(map[string]any)
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
	return c.MustCall("GET", url, nil).(map[string]any)
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
	n := c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions", data)
	id, err := strconv.Atoi(string(n.(json.Number)))
	if err != nil {
		c.t.Fatalf("ID %q is not integer", n)
	}
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
	n, err := c.Call("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions", data)
	if err != nil {
		return 0, err
	}
	id, err := strconv.Atoi(string(n.(json.Number)))
	if err != nil {
		return 0, fmt.Errorf("ID %q is not integer", string(n.(json.Number)))
	}
	return id, nil
}

func (c *Chichi) AddConnection(connection ConnectionToAdd) int {
	data := map[string]any{
		"Connection": connection,
	}
	n := c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/add-connection", data)
	id, err := strconv.Atoi(string(n.(json.Number)))
	if err != nil {
		c.t.Fatalf("ID %q is not integer", n)
	}
	return id
}

func (c *Chichi) AddDestinationFilesystem(storageDir, businessID string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:       "Filesystem",
		Role:       Destination,
		Enabled:    true,
		Connector:  FilesystemConnector,
		BusinessID: businessID,
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
	return c.AddConnection(ConnectionToAdd{
		Name:       name,
		Role:       role,
		Enabled:    true,
		Connector:  DummyConnector,
		BusinessID: businessID,
		Settings:   []byte("{}"),
	})
}

func (c *Chichi) AddJavaScriptSource(name, host, businessID string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:        name,
		Role:        Source,
		Enabled:     true,
		Connector:   JavaScriptConnector,
		Strategy:    &defaultStrategy,
		WebsiteHost: host,
		BusinessID:  businessID,
	})
}

func (c *Chichi) AddSourceFilesystem(storageDir, businessID string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:       "Filesystem",
		Role:       Source,
		Enabled:    true,
		Connector:  FilesystemConnector,
		BusinessID: businessID,
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
	c.MustCall("POST", url, req)
}

func (c *Chichi) ChangeUsersSchemaQueries(schema types.Type, rePaths map[string]any) []string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/change-users-schema-queries"
	req := map[string]any{
		"Schema":  schema,
		"RePaths": rePaths,
	}
	response := c.MustCall("POST", url, req)
	var queries []string
	for _, query := range response.(map[string]any)["Queries"].([]any) {
		queries = append(queries, query.(string))
	}
	return queries
}

func (c *Chichi) CompletePath(storage int, path string) string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(storage) + "/complete-path/" + url.PathEscape(path)
	return c.MustCall("GET", url, nil).(map[string]any)["path"].(string)
}

func (c *Chichi) ConnectionIdentities(connection int, first, limit int) ([]UserIdentity, int) {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/identities"
	req := map[string]any{
		"First": first,
		"Limit": limit,
	}
	response := c.MustCall("POST", url, req).(map[string]any)
	count, err := response["count"].(json.Number).Int64()
	if err != nil {
		c.t.Fatalf("invalid 'count' for user identities: %s", err)
	}
	jsonIdentities, err := json.Marshal(response["identities"].([]any))
	if err != nil {
		c.t.Fatalf("cannot marshal identities: %s", err)
	}
	var identities []UserIdentity
	err = json.Unmarshal(jsonIdentities, &identities)
	if err != nil {
		c.t.Fatalf("cannot unmarshal identities: %s", err)
	}
	return identities, int(count)
}

func (c *Chichi) ConnectionKeys(conn int) []string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(conn) + "/keys"
	rawKeys := c.MustCall("GET", url, nil).([]any)
	keys := make([]string, len(rawKeys))
	for i := range rawKeys {
		keys[i] = rawKeys[i].(string)
	}
	return keys
}

func (c *Chichi) DeleteConnection(connection int) {
	method := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection)
	c.MustCall("DELETE", method, nil)
}

func (c *Chichi) ExecuteAction(connection, action int, reimport bool) {
	method := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/actions/" + strconv.Itoa(action) + "/execute"
	c.MustCall("POST", method, map[string]any{"Reimport": reimport})
}

func (c *Chichi) Executions(connection int) []any {
	method := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(connection) + "/executions"
	return c.MustCall("GET", method, nil).([]any)
}

func (c *Chichi) IdentifiersSchema() types.Type {
	mapSchema := c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/identifiers-schema", nil)
	jsonSchema, err := json.Marshal(mapSchema)
	if err != nil {
		c.t.Fatalf("cannot marshal schema: %s", err)
	}
	schema, err := types.Parse(string(jsonSchema))
	if err != nil {
		c.t.Fatalf("cannot parse schema: %s", err)
	}
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
	response := c.MustCall("POST", url, req).(map[string]any)
	rawRecords := response["records"].([]any)
	mapSchema := response["schema"].(map[string]any)
	jsonSchema, err := json.Marshal(mapSchema)
	if err != nil {
		c.t.Fatalf("cannot marshal schema: %s", err)
	}
	schema, err := types.Parse(string(jsonSchema))
	if err != nil {
		c.t.Fatalf("cannot parse schema: %s", err)
	}
	var records []map[string]any
	for _, b := range rawRecords {
		records = append(records, b.(map[string]any))
	}
	return records, schema
}

func (c *Chichi) RunWorkspaceIdentityResolution() {
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/run-identity-resolution", nil)
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
	c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/identifiers", body)
}

func (c *Chichi) Sheets(storage int, fileConnector int, path string, compression Compression, settings json.RawMessage) []string {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(storage) + "/sheets"
	req := map[string]any{
		"FileConnector": fileConnector,
		"Path":          path,
		"Compression":   compression,
		"Settings":      settings,
	}
	sheets := []string{}
	for _, s := range c.MustCall("POST", url, req).(map[string]any)["sheets"].([]any) {
		sheets = append(sheets, s.(string))
	}
	return sheets
}

func (c *Chichi) TableSchema(connection int, table string) types.Type {
	mapSchema := c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+
		strconv.Itoa(connection)+"/tables/"+url.PathEscape(table)+"/schema", nil)
	jsonSchema, err := json.Marshal(mapSchema)
	if err != nil {
		c.t.Fatalf("cannot marshal schema: %s", err)
	}
	schema, err := types.Parse(string(jsonSchema))
	if err != nil {
		c.t.Fatalf("cannot parse schema: %s", err)
	}
	return schema
}

func (c *Chichi) UserEvents(user int) []map[string]any {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/users/" + strconv.Itoa(user) + "/events"
	response := c.MustCall("GET", url, nil).(map[string]any)
	events := make([]map[string]any, len(response["events"].([]any)))
	for i, event := range response["events"].([]any) {
		events[i] = event.(map[string]any)
	}
	return events
}

func (c *Chichi) UserIdentities(user int, first, limit int) ([]UserIdentity, int) {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/users/" + strconv.Itoa(user) + "/identities"
	req := map[string]any{
		"First": first,
		"Limit": limit,
	}
	response := c.MustCall("POST", url, req).(map[string]any)
	count, err := response["count"].(json.Number).Int64()
	if err != nil {
		c.t.Fatalf("invalid 'count' for user identities: %s", err)
	}
	jsonIdentities, err := json.Marshal(response["identities"].([]any))
	if err != nil {
		c.t.Fatalf("cannot marshal identities: %s", err)
	}
	var identities []UserIdentity
	err = json.Unmarshal(jsonIdentities, &identities)
	if err != nil {
		c.t.Fatalf("cannot unmarshal identities: %s", err)
	}
	return identities, int(count)
}

func (c *Chichi) Users(properties []string, order string, first, limit int) (users []map[string]any, schema types.Type, count int) {
	req := map[string]any{
		"Properties": properties,
		"Order":      order,
		"First":      first,
		"Limit":      limit,
	}

	response := c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/users", req).(map[string]any)

	// Users.
	jsonUsers, err := json.Marshal(response["users"].([]any))
	if err != nil {
		c.t.Fatalf("cannot marshal identities: %s", err)
	}
	usersDecoder := json.NewDecoder(bytes.NewReader(jsonUsers))
	usersDecoder.UseNumber()
	err = usersDecoder.Decode(&users)
	if err != nil {
		c.t.Fatalf("cannot unmarshal users: %s", err)
	}

	// Schema.
	jsonSchema, err := json.Marshal(response["schema"])
	if err != nil {
		c.t.Fatalf("cannot marshal schema: %s", err)
	}
	schema, err = types.Parse(string(jsonSchema))
	if err != nil {
		c.t.Fatalf("cannot parse schema: %s", err)
	}

	// Count.
	count64, err := response["count"].(json.Number).Int64()
	if err != nil {
		c.t.Fatalf("invalid 'count' for user identities: %s", err)
	}
	count = int(count64)

	return users, schema, count
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
	response := c.MustCall("GET", "/api/workspaces/"+strconv.Itoa(c.workspace), nil)
	jsonWs, err := json.Marshal(response)
	if err != nil {
		c.t.Fatal(err)
	}
	var ws Workspace
	err = json.Unmarshal(jsonWs, &ws)
	if err != nil {
		c.t.Fatal(err)
	}
	return ws
}

func JSONEncodeSettings(settings map[string]any) []byte {
	data, err := json.Marshal(settings)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection settings to JSON: %s", err))
	}
	return data
}

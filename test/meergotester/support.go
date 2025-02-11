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
	"slices"
	"time"

	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/core"
	"github.com/meergo/meergo/test/analytics-go"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

var defaultStrategy Strategy = "Conversion"

// This file contains support methods which reduce verbosity of tests.

func (c *Meergo) AbsolutePath(storage int, path string) string {
	var response struct {
		Path string `json:"path"`
	}
	endpointPath := fmt.Sprintf("/api/connections/%d/files/%s/absolute", storage, url.PathEscape(path))
	c.MustCall("GET", endpointPath, nil, &response)
	return response.Path
}

func (c *Meergo) Action(action int) Action {
	path := fmt.Sprintf("/api/actions/%d", action)
	var response map[string]any
	c.MustCall("GET", path, nil, &response)
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

func (c *Meergo) ActionSchemas(conn int, target core.Target, eventType string) map[string]any {
	path := fmt.Sprintf("/api/connections/%d/actions/schemas/%s", conn, target)
	if eventType != "" {
		path += "/" + eventType
	}
	var schemas map[string]any
	c.MustCall("GET", path, nil, &schemas)
	return schemas
}

func (c *Meergo) ConnectionIdentities(conn, first, limit int) ([]UserIdentity, int) {
	req := map[string]any{
		"first": first,
		"limit": limit,
	}
	var response struct {
		Identities []UserIdentity `json:"identities"`
		Total      int            `json:"total"`
	}
	path := fmt.Sprintf("/api/connections/%d/identities", conn)
	c.MustCall("POST", path, req, &response)
	return response.Identities, response.Total
}

func (c *Meergo) ConnectionUI(connection int) map[string]any {
	path := fmt.Sprintf("/api/connections/%d/ui", connection)
	var ui map[string]any
	c.MustCall("GET", path, nil, &ui)
	return ui
}

func (c *Meergo) CreateAction(conn int, target string, action ActionToSet) int {
	switch target {
	case "Events", "Users", "Groups":
	default:
		panic(fmt.Sprintf("invalid target %q", target))
	}
	actionJSON, err := json.Marshal(action)
	if err != nil {
		panic(err)
	}
	var body map[string]any
	err = json.Unmarshal(actionJSON, &body)
	if err != nil {
		panic(err)
	}
	body["connection"] = conn
	body["target"] = target
	var id int
	c.MustCall("POST", "/api/actions", body, &id)
	return id
}

// CreateActionErr is like CreateAction but returns an error instead of
// panicking.
func (c *Meergo) CreateActionErr(conn int, target string, action ActionToSet) (int, error) {
	switch target {
	case "Events", "Users", "Groups":
	default:
		panic(fmt.Sprintf("invalid target %q", target))
	}
	actionJSON, err := json.Marshal(action)
	if err != nil {
		panic(err)
	}
	var body map[string]any
	err = json.Unmarshal(actionJSON, &body)
	if err != nil {
		panic(err)
	}
	body["connection"] = conn
	body["target"] = target
	var id int
	err = c.Call("POST", "/api/actions", body, &id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (c *Meergo) CreateConnection(connection ConnectionToCreate) int {
	var id int
	c.MustCall("POST", "/api/connections", connection, &id)
	return id
}

func (c *Meergo) CreateDestinationFilesystem(storageDir string) int {
	return c.CreateConnection(ConnectionToCreate{
		Name:      "Filesystem",
		Role:      Destination,
		Connector: "Filesystem",
		Settings: JSONEncodeSettings(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Meergo) CreateDestinationPostgreSQL() int {
	return c.CreateConnection(ConnectionToCreate{
		Name:      "PostgreSQL (destination)",
		Role:      Destination,
		Connector: "PostgreSQL",
		Settings: JSONEncodeSettings(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
			"Schema":   testsSettings.Database.Schema,
		}),
	})
}

func (c *Meergo) CreateDummy(name string, role Role) int {
	conn := ConnectionToCreate{
		Name:      name,
		Role:      role,
		Connector: "Dummy",
	}
	if role == Destination {
		conn.Settings = []byte("{}")
	}
	if role == Destination {
		mode := Cloud
		conn.SendingMode = &mode
	}
	return c.CreateConnection(conn)
}

func (c *Meergo) CreateDummyWithSettings(name string, role Role, settings DummySettings) int {
	conn := ConnectionToCreate{
		Name:      name,
		Role:      role,
		Connector: "Dummy",
		Settings:  JSONEncodeSettings(settings),
	}
	if role == Destination {
		mode := Cloud
		conn.SendingMode = &mode
	}
	return c.CreateConnection(conn)
}

func (c *Meergo) CreateEventAction(conn int, eventType string, action ActionToSet) int {
	actionJSON, err := json.Marshal(action)
	if err != nil {
		panic(err)
	}
	var body map[string]any
	err = json.Unmarshal(actionJSON, &body)
	if err != nil {
		panic(err)
	}
	body["connection"] = conn
	body["target"] = "Events"
	body["eventType"] = eventType
	var id int
	c.MustCall("POST", "/api/actions", body, &id)
	return id
}

func (c *Meergo) CreateJavaScriptSource(name, host string, linkedConnections []int) int {
	return c.CreateConnection(ConnectionToCreate{
		Name:              name,
		Role:              Source,
		Connector:         "JavaScript",
		Strategy:          &defaultStrategy,
		WebsiteHost:       host,
		LinkedConnections: linkedConnections,
	})
}

func (c *Meergo) CreateSourceFilesystem(storageDir string) int {
	return c.CreateConnection(ConnectionToCreate{
		Name:      "Filesystem",
		Role:      Source,
		Connector: "Filesystem",
		Settings: JSONEncodeSettings(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Meergo) CreateSourcePostgreSQL() int {
	return c.CreateConnection(ConnectionToCreate{
		Name:      "PostgreSQL (destination)",
		Role:      Source,
		Connector: "PostgreSQL",
		Settings: JSONEncodeSettings(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
			"Schema":   testsSettings.Database.Schema,
		}),
	})
}

func (c *Meergo) DeleteConnection(conn int) {
	path := fmt.Sprintf("/api/connections/%d", conn)
	c.MustCall("DELETE", path, nil, nil)
}

func (c *Meergo) ExecuteAction(action int) int {
	var response struct {
		ID int
	}
	path := fmt.Sprintf("/api/actions/%d/exec", action)
	c.MustCall("POST", path, nil, &response)
	return response.ID
}

func (c *Meergo) Execution(id int) Execution {
	var exe Execution
	path := fmt.Sprintf("/api/actions/executions/%d", id)
	c.MustCall("GET", path, nil, &exe)
	return exe
}

func (c *Meergo) Executions() []Execution {
	var response struct {
		Executions []Execution
	}
	c.MustCall("GET", "/api/actions/executions", nil, &response)
	return response.Executions
}

func (c *Meergo) File(storage int, path, format, sheet string, compression Compression, settings json.RawMessage, limit int) ([]map[string]any, types.Type) {
	req := map[string]any{
		"format":         format,
		"sheet":          sheet,
		"compression":    compression,
		"formatSettings": settings,
		"limit":          limit,
	}
	var response struct {
		Records []map[string]any `json:"records"`
		Schema  types.Type       `json:"schema"`
	}
	endpointPath := fmt.Sprintf("/api/connections/%d/files/%s", storage, url.PathEscape(path))
	c.MustCall("POST", endpointPath, req, &response)
	return response.Records, response.Schema
}

func (c *Meergo) IdentifiersSchema() types.Type {
	var schema types.Type
	c.MustCall("GET", "/api/identifiers-schema", nil, &schema)
	return schema
}

func (c *Meergo) LatestIdentityResolution() (startTime, endTime *time.Time) {
	var response struct {
		StartTime *time.Time `json:"startTime"`
		EndTime   *time.Time `json:"endTime"`
	}
	c.MustCall("GET", "/api/identity-resolution/latest", nil, &response)
	return response.StartTime, response.EndTime
}

func (c *Meergo) PreviewUserSchemaUpdate(schema types.Type, rePaths map[string]any) []string {
	req := map[string]any{
		"schema":  schema,
		"rePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	c.MustCall("PUT", "/api/users/schema/preview", req, &response)
	return response.Queries
}

// PreviewUserSchemaUpdateErr is like PreviewUserSchemaUpdate but returns an
// error instead of panicking.
func (c *Meergo) PreviewUserSchemaUpdateErr(schema types.Type, rePaths map[string]any) ([]string, error) {
	req := map[string]any{
		"schema":  schema,
		"rePaths": rePaths,
	}
	var response struct {
		Queries []string
	}
	err := c.Call("PUT", "/api/users/schema/preview", req, &response)
	if err != nil {
		return nil, err
	}
	return response.Queries, nil
}

func (c *Meergo) RepairWarehouse() {
	c.MustCall("POST", "/api/warehouse/repair", nil, nil)
}

// RunIdentityResolution starts the identity resolution and waits for it to
// complete.
func (c *Meergo) RunIdentityResolution() {
	ts := time.Now().UTC()
	c.MustCall("POST", "/api/identity-resolution/start", nil, nil)
	// Waits for the Identity Resolution that was started following the call to
	// this method to finish.
	for {
		time.Sleep(50 * time.Millisecond)
		startTime, endTime := c.LatestIdentityResolution()
		if startTime.After(ts) && endTime != nil {
			break
		}
	}
}

func (c *Meergo) SendEvent(writeKey string, message analytics.Message) {
	endpoint := "http://" + testsSettings.MeergoHost + "/" + "api"
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

func (c *Meergo) Sheets(storage int, path string, format string, compression Compression, settings json.RawMessage) []string {
	request := map[string]any{
		"format":         format,
		"compression":    compression,
		"formatSettings": settings,
	}
	var response struct {
		Sheets []string `json:"sheets"`
	}
	endpointPath := fmt.Sprintf("/api/connections/%d/files/%s/sheets", storage, url.PathEscape(path))
	c.MustCall("POST", endpointPath, request, &response)
	return response.Sheets
}

func (c *Meergo) TableSchema(conn int, table string) types.Type {
	var schema types.Type
	path := fmt.Sprintf("/api/connections/%d/tables/%s", conn, url.PathEscape(table))
	c.MustCall("GET", path, nil, &schema)
	return schema
}

func (c *Meergo) TestWarehouseUpdate(settings []byte) {
	body := map[string]any{
		"settings": json.RawMessage(settings),
	}
	c.MustCall("PUT", "/api/warehouse/test", body, nil)
}

func (c *Meergo) TestWorkspaceCreation(name string, userSchema types.Type,
	displayedProperties DisplayedProperties, whType string, whSettings []byte, mode WarehouseMode) error {
	body := map[string]any{
		"name":                name,
		"userSchema":          userSchema,
		"displayedProperties": displayedProperties,
		"warehouse": map[string]any{
			"type":     whType,
			"mode":     mode,
			"settings": json.RawMessage(whSettings),
		},
	}
	return c.Call("POST", "/api/workspaces/test", body, nil)
}

func (c *Meergo) UpdateAction(actionID int, action ActionToSet) {
	path := fmt.Sprintf("/api/actions/%d", actionID)
	c.MustCall("PUT", path, action, nil)
}

func (c *Meergo) UpdateIdentityResolution(runOnBatchImport bool, identifiers []string) {
	body := map[string]any{
		"runOnBatchImport": runOnBatchImport,
		"identifiers":      identifiers,
	}
	c.MustCall("PUT", "/api/identity-resolution/settings", body, nil)
}

func (c *Meergo) UpdateIdentityResolutionErr(identifiers []string) error {
	body := map[string]any{
		"identifiers": identifiers,
	}
	return c.Call("PUT", "/api/identity-resolution/settings", body, nil)
}

func (c *Meergo) UpdateUserSchema(schema types.Type, primarySources map[string]int, rePaths map[string]any) {
	req := map[string]any{
		"schema":         schema,
		"primarySources": primarySources,
		"rePaths":        rePaths,
	}
	c.MustCall("PUT", "/api/users/schema", req, nil)
}

// UpdateUserSchemaErr is like UpdateUserSchema but returns an error instead of
// panicking.
func (c *Meergo) UpdateUserSchemaErr(schema types.Type, primarySources map[string]int, rePaths map[string]any) error {
	req := map[string]any{
		"schema":         schema,
		"primarySources": primarySources,
		"rePaths":        rePaths,
	}
	return c.Call("PUT", "/api/users/schema", req, nil)
}

func (c *Meergo) UpdateWarehouse(mode string, settings []byte) {
	body := map[string]any{
		"mode":     mode,
		"settings": json.RawMessage(settings),
	}
	c.MustCall("PUT", "/api/warehouse", body, nil)
}

func (c *Meergo) UserEvents(user uuid.UUID, properties []string) []map[string]any {
	request := map[string]any{
		"properties": properties,
		"filter": Filter{
			Logical: OpAnd,
			Conditions: []FilterCondition{
				{Property: "user",
					Operator: OpIs,
					Values:   []string{user.String()}},
			},
		},
		"order":     "timestamp",
		"orderDesc": true,
		"first":     0,
		"limit":     10,
	}
	var response struct {
		Events []map[string]any `json:"events"`
	}
	c.MustCall("POST", "/api/events", request, &response)
	return response.Events
}

func (c *Meergo) UserIdentities(user uuid.UUID, first, limit int) ([]UserIdentity, int) {
	var response struct {
		Identities []UserIdentity `json:"identities"`
		Total      int            `json:"total"`
	}
	path := fmt.Sprintf("/api/users/%s/identities?first=%d&limit=%d", user, first, limit)
	c.MustCall("GET", path, nil, &response)
	return response.Identities, response.Total
}

func (c *Meergo) Users(properties []string, order string, orderDesc bool, first, limit int) (users []User, schema types.Type, total int) {
	req := map[string]any{
		"properties": properties,
		"order":      order,
		"orderDesc":  orderDesc,
		"first":      first,
		"limit":      limit,
	}
	var response struct {
		Users  []User     `json:"users"`
		Schema types.Type `json:"schema"`
		Total  int        `json:"total"`
	}
	c.MustCall("POST", "/api/users", req, &response)
	return response.Users, response.Schema, response.Total
}

func (c *Meergo) WaitEventsStoredIntoWarehouse(ctx context.Context, expected int) {
	bo := backoff.New(200)
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

// WaitForExecutionsCompletion waits for the executions with the specified IDs,
// belonging to the connection, to be completed. In the event that an execution
// ends with an error, or there is at least one "Failed", this method will
// result in an error.
//
// If you intend to proceed even in the case of one or more "Failed" (but not an
// error for the entire execution), see the method
// WaitForExecutionsCompletionAllowFailed.
func (c *Meergo) WaitForExecutionsCompletion(conn int, executions ...int) {
	c.waitForExecutionsCompletion(false, executions...)
}

// WaitForExecutionsCompletionAllowFailed waits for the executions with the
// specified IDs, belonging to the connection, to be completed. In the event
// that an execution ends with an error, this method will result in an error. If
// there is one or more Failed, they are ignored.
//
// If you want the method to result in an error even in the case of one or more
// "Failed", see the method WaitForExecutionsCompletion.
func (c *Meergo) WaitForExecutionsCompletionAllowFailed(conn int, executions ...int) {
	c.waitForExecutionsCompletion(true, executions...)
}

func (c *Meergo) EventWriteKeys(conn int) []string {
	var keys []string
	path := fmt.Sprintf("/api/connections/%d/event-write-keys", conn)
	c.MustCall("GET", path, nil, &keys)
	return keys
}

func (c *Meergo) Workspace() Workspace {
	var ws Workspace
	c.MustCall("GET", "/api/workspaces/current", nil, &ws)
	return ws
}

func (c *Meergo) waitForExecutionsCompletion(allowFailed bool, ids ...int) {
	time.Sleep(500 * time.Millisecond)
	for {
		if len(ids) == 1 {
			exe := c.Execution(ids[0])
			if exe.EndTime != nil {
				// If the action execution ended with an error, make the test fail.
				if exe.Error != "" {
					c.t.Fatalf("an error occurred when running action %d on connection %d: %s", exe.Action, exe.ID, exe.Error)
				}
				if !allowFailed && exe.Failed != [6]int{} {
					c.t.Fatalf("an error occurred when running action %d on connection %d: %d failed", exe.Action, exe.ID, exe.Failed)
				}
				return
			}
			time.Sleep(1 * time.Second)
			continue
		}
		completed := true
		for _, exe := range c.Executions() {
			if !slices.Contains(ids, exe.ID) {
				continue
			}
			if exe.EndTime == nil {
				completed = false
				continue
			}
			// If the action execution ended with an error, make the test fail.
			if exe.Error != "" {
				c.t.Fatalf("an error occurred when running action %d on connection %d: %s", exe.Action, exe.ID, exe.Error)
			}
			if !allowFailed && exe.Failed != [6]int{} {
				c.t.Fatalf("an error occurred when running action %d on connection %d: %d failed", exe.Action, exe.ID, exe.Failed)
			}
		}
		if completed {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func JSONEncodeSettings(values any) []byte {
	data, err := json.Marshal(values)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection settings to JSON: %s", err))
	}
	return data
}

func SettingsProperties(properties map[string]bool) []byte {
	var settings = struct {
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
		settings.Properties = append(settings.Properties, kv)
	}
	return JSONEncodeSettings(settings)
}

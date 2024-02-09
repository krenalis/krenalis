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

	"chichi/apis"
	"chichi/backoff"
	"chichi/connector/types"

	"github.com/segmentio/analytics-go/v3"
)

// This file contains support methods which reduce verbosity of tests.

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

func (c *Chichi) AddDestinationCSV(filesystem int) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "CSV",
		Role:      Destination,
		Enabled:   true,
		Connector: 5, // CSV.
		Storage:   filesystem,
		Settings: JSONEncodeSettings(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})
}

func (c *Chichi) AddDestinationFilesystem(storageDir string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "Filesystem",
		Role:      Destination,
		Enabled:   true,
		Connector: 19, // Filesystem.
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
		Connector: 10, // PostgreSQL.
		Settings: JSONEncodeSettings(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
		}),
	})
}

func (c *Chichi) AddDummy(name string, role Role) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      name,
		Role:      role,
		Enabled:   true,
		Connector: 3, // Dummy.
		Settings:  []byte("{}"),
	})
}

func (c *Chichi) AddSourceCSV(filesystem int) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "CSV",
		Role:      Source,
		Enabled:   true,
		Connector: 5, // CSV.
		Storage:   filesystem,
		Settings: JSONEncodeSettings(map[string]any{
			"Comma":          ",",
			"HasColumnNames": true,
		}),
	})
}

func (c *Chichi) AddSourceFilesystem(storageDir string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "Filesystem",
		Role:      Source,
		Enabled:   true,
		Connector: 19, // Filesystem.
		Settings: JSONEncodeSettings(map[string]any{
			"Root": storageDir,
		}),
	})
}

func (c *Chichi) AddSourceJSON(filesystem int) int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "JSON",
		Role:      Source,
		Enabled:   true,
		Storage:   filesystem,
		Connector: 21, // JSON.
		Settings:  []byte("{}"),
	})
}

func (c *Chichi) AddSourcePostgreSQL() int {
	return c.AddConnection(ConnectionToAdd{
		Name:      "PostgreSQL (destination)",
		Role:      Source,
		Enabled:   true,
		Connector: 10, // PostgreSQL.
		Settings: JSONEncodeSettings(map[string]any{
			"Host":     testsSettings.Database.Host,
			"Port":     testsSettings.Database.Port,
			"Username": testsSettings.Database.Username,
			"Password": testsSettings.Database.Password,
			"Database": testsSettings.Database.Database,
		}),
	})
}

func (c *Chichi) AddWebsiteSource(name, host string) int {
	return c.AddConnection(ConnectionToAdd{
		Name:        name,
		Role:        Source,
		Enabled:     true,
		Connector:   12, // Website.
		WebsiteHost: host,
	})
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

func (c *Chichi) Users(properties []string, order string, first, limit int) map[string]any {
	req := map[string]any{
		"Properties": properties,
		"Order":      order,
		"First":      first,
		"Limit":      limit,
	}
	return c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/users", req).(map[string]any)
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

func JSONEncodeSettings(settings map[string]any) []byte {
	data, err := json.Marshal(settings)
	if err != nil {
		panic(fmt.Sprintf("cannot encode connection settings to JSON: %s", err))
	}
	return data
}

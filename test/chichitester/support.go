//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"chichi/apis"
	"chichi/connector"
	"chichi/connector/types"

	"github.com/segmentio/analytics-go"
)

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) ActionSchemas(conn int, target apis.Target, eventType string) map[string]any {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/connections/" + strconv.Itoa(conn) + "/action-schemas/" + target.String()
	if eventType != "" {
		url += "/" + eventType
	}
	return c.MustCall("GET", url, nil).(map[string]any)
}

func (c *Chichi) AddAction(connection int, data map[string]any) int {
	id := c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions", data).(float64)
	return int(id)
}

// AddActionErr is like AddAction but returns an error instead of panicking.
func (c *Chichi) AddActionErr(connection int, data map[string]any) (int, error) {
	id, err := c.Call("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions", data)
	if err != nil {
		return 0, err
	}
	return int(id.(float64)), nil
}

func (c *Chichi) AddConnection(data map[string]any) int {
	id := c.MustCall("POST", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/add-connection", data).(float64)
	return int(id)
}

func (c *Chichi) AddDestinationPostgreSQL() int {
	return c.AddConnection(map[string]any{
		"Connection": map[string]any{
			"Name":      "PostgreSQL (destination)",
			"Role":      apis.Destination,
			"Enabled":   true,
			"Connector": 10, // PostgreSQL.
			"Settings": map[string]any{
				"Host":     testsSettings.Database.Host,
				"Port":     testsSettings.Database.Port,
				"Username": testsSettings.Database.Username,
				"Password": testsSettings.Database.Password,
				"Database": testsSettings.Database.Database,
			},
		},
	})
}

func (c *Chichi) AddDummy(name string, role connector.Role) int {
	return c.AddConnection(map[string]any{
		"Connection": map[string]any{
			"Name":      name,
			"Role":      role.String(),
			"Enabled":   true,
			"Connector": 3,                // Dummy.
			"Settings":  map[string]any{}, // TODO(Gianluca): see https://github.com/open2b/chichi/issues/355.
		},
	})
}

func (c *Chichi) AddSourceCSV(filesystem int) int {
	return c.AddConnection(map[string]any{
		"Connection": map[string]any{
			"Name":      "CSV",
			"Role":      "Source",
			"Enabled":   true,
			"Connector": 5, // CSV.
			"Storage":   filesystem,
			"Settings": map[string]any{
				"Comma":          ",",
				"HasColumnNames": true,
			},
		},
	})
}

func (c *Chichi) AddSourceFilesystem(storageDir string) int {
	return c.AddConnection(map[string]any{
		"Connection": map[string]any{
			"Name":      "Filesystem",
			"Role":      "Source",
			"Enabled":   true,
			"Connector": 19, // Filesystem.
			"Settings": map[string]any{
				"Root": storageDir,
			},
		},
	})
}

func (c *Chichi) AddSourceJSON(filesystem int) int {
	return c.AddConnection(map[string]any{
		"Connection": map[string]any{
			"Name":      "JSON",
			"Role":      "Source",
			"Enabled":   true,
			"Storage":   filesystem,
			"Connector": 21, // JSON.
			"Settings":  map[string]any{},
		},
	})
}

func (c *Chichi) AddWebsiteSource(name, host string) int {
	return c.AddConnection(map[string]any{
		"Connection": map[string]any{
			"Name":        name,
			"Role":        connector.Source.String(),
			"Enabled":     true,
			"Connector":   12, // Website.
			"WebsiteHost": host,
		},
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

func (c *Chichi) SetAction(connection, action int, data map[string]any) {
	c.MustCall("PUT", "/api/workspaces/"+strconv.Itoa(c.workspace)+"/connections/"+strconv.Itoa(connection)+"/actions/"+strconv.Itoa(action), data)
}

func (c *Chichi) SetWorkspaceIdentifiers(identifiers []string, anonymousIdentifiers apis.AnonymousIdentifiers) {
	body := map[string]any{
		"Identifiers":          identifiers,
		"AnonymousIdentifiers": anonymousIdentifiers,
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

func (c *Chichi) UserEvents(user int) []apis.Event {
	url := "/api/workspaces/" + strconv.Itoa(c.workspace) + "/users/" + strconv.Itoa(user) + "/events"
	response := c.MustCall("GET", url, nil).(map[string]any)
	rawEvents := response["events"]
	data, err := json.Marshal(rawEvents)
	if err != nil {
		c.t.Fatal(err)
	}
	var events []apis.Event
	err = json.Unmarshal(data, &events)
	if err != nil {
		c.t.Fatal(err)
	}
	return events
}

func (c *Chichi) Users(properties []string, start, end int) map[string]any {
	req := map[string]any{
		"Properties": properties,
		"Start":      start,
		"End":        end,
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
				actionID := int(e["Action"].(float64))
				connID := int(e["ID"].(float64))
				c.t.Fatalf("an error occurred when running action %d on connection %d: %s", actionID, connID, err)
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

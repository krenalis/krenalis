//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"strconv"
	"time"

	"chichi/apis"
	"chichi/connector"
)

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) AddAction(connection int, data map[string]any) int {
	id := c.MustCall("POST", "/api/connections/"+strconv.Itoa(connection)+"/actions", data).(float64)
	return int(id)
}

func (c *Chichi) AddConnection(data map[string]any) int {
	id := c.MustCall("POST", "/api/workspace/add-connection", data).(float64)
	return int(id)
}

func (c *Chichi) AddDummy(name string, role connector.Role) int {
	return c.AddConnection(map[string]any{
		"Connector": 3, // Dummy.
		"Role":      role.String(),
		"Options": map[string]any{
			"Name":    name,
			"Enabled": true,
		},
	})
}

func (c *Chichi) AddSourceCSV(filesystem int) int {
	return c.AddConnection(map[string]any{
		"Connector": 5, // CSV.
		"Role":      "Source",
		"Options": map[string]any{
			"Name":    "CSV",
			"Enabled": true,
			"Storage": filesystem,
		},
		"Settings": map[string]any{
			"Comma": ",",
		},
	})
}

func (c *Chichi) AddSourceFilesystem(storageDir string) int {
	return c.AddConnection(map[string]any{
		"Connector": 19, // Filesystem.
		"Role":      "Source",
		"Options": map[string]any{
			"Name":    "Filesystem",
			"Enabled": true,
		},
		"Settings": map[string]any{
			"Root": storageDir,
		},
	})
}

func (c *Chichi) ActionSchemas(conn int, target apis.ActionTarget, eventType string) map[string]any {
	url := "/api/connections/" + strconv.Itoa(conn) + "/action-schemas/" + target.String()
	if eventType != "" {
		url += "/" + eventType
	}
	return c.MustCall("GET", url, nil).(map[string]any)
}

func (c *Chichi) ExecuteAction(connection, action int, reimport bool) {
	method := "/api/connections/" + strconv.Itoa(connection) + "/actions/" + strconv.Itoa(action) + "/execute"
	c.MustCall("POST", method, map[string]any{"Reimport": reimport})
}

func (c *Chichi) Imports(connection int) []any {
	method := "/api/connections/" + strconv.Itoa(connection) + "/imports"
	return c.MustCall("GET", method, nil).([]any)
}

func (c *Chichi) Users(properties []string, start, end int) map[string]any {
	req := map[string]any{
		"Properties": properties,
		"Start":      start,
		"End":        end,
	}
	return c.MustCall("POST", "/api/users", req).(map[string]any)
}

func (c *Chichi) SetAction(connection, action int, data map[string]any) {
	c.MustCall("PUT", "/api/connections/"+strconv.Itoa(connection)+"/actions/"+strconv.Itoa(action), data)
}

func (c *Chichi) WaitActionsToFinish(conn int) {
	time.Sleep(500 * time.Millisecond)
	for {
		// TODO(Gianluca): here 'Imports' is called because it also returns the
		// executions of exports. This call will be changed when we will revise
		// the implementation of such method.
		stillRunning := false
		for _, exec := range c.Imports(conn) {
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

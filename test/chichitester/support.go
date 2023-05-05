//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import "strconv"

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) AddAction(connection int, data map[string]any) int {
	id := c.MustCall("POST", "/api/connections/"+strconv.Itoa(connection)+"/actions", data).(float64)
	return int(id)
}

func (c *Chichi) AddConnection(data map[string]any) int {
	id := c.MustCall("POST", "/api/workspace/add-connection", data).(float64)
	return int(id)
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

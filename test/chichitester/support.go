//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

// This file contains support methods which reduce verbosity of tests.

func (c *Chichi) AddConnection(data map[string]any) int {
	id := c.MustCall("POST", "/api/workspace/add-connection", data).(float64)
	return int(id)
}

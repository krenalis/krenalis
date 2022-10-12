//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package chichiapis

import (
	"fmt"
	"log"
	"sort"
)

func ListEnabledConnectors() {
	resp, err := call("admin/connectors/findInstalledConnectors", nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, connector := range resp.([]any) {
		c := connector.(map[string]any)
		fmt.Printf("%-10v %s\n", c["ID"], c["Name"])
	}
}

func ListConnectorProperties(connector int) {
	resp, err := call("admin/connectors-properties", map[string]any{"Connector": connector})
	if err != nil {
		log.Fatal(err)
	}
	for _, property := range resp.([]any) {
		p := property.(map[string]any)
		fmt.Printf("%-50s %-40s %s\n", p["Label"], p["Name"], p["Type"])
	}
}

func ImportUsersFromConnector(connector int, resetCursor bool) {
	body := map[string]any{
		"Connector":   connector,
		"ResetCursor": resetCursor,
	}
	resp, err := call("admin/import-raw-user-data-from-connector", body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.(map[string]any)["status"])
}

func GetTransformation(connector int) {
	body := map[string]any{
		"Connector": connector,
	}
	resp, err := call("admin/transformations/get", body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp)
}

func UpdateTransformation(connector int, transformation []byte) {
	body := map[string]any{
		"Connector":      connector,
		"Transformation": string(transformation),
	}
	resp, err := call("admin/transformations/update", body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp)
}

func ListUsers() {
	resp, err := call("admin/list-users", nil)
	if err != nil {
		log.Fatal(err)
	}
	users := resp.([]any)
	if len(users) == 0 {
		return
	}
	columns := []string{}
	for k := range users[0].(map[string]any) {
		columns = append(columns, k)
	}
	sort.Strings(columns)
	for _, column := range columns {
		fmt.Printf("%-30s", column)
	}
	fmt.Printf("\n")
	for range columns {
		fmt.Printf("%-30s", "------")
	}
	fmt.Printf("\n")
	for _, user := range users {
		for _, column := range columns {
			fmt.Printf("%-30s", user.(map[string]any)[column])
		}
		fmt.Printf("\n")
	}
}

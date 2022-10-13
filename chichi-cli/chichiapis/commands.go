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
	"strconv"
)

func ListEnabledConnectors() {
	resp, err := callAdmin("admin/connectors/findInstalledConnectors", nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, connector := range resp.([]any) {
		c := connector.(map[string]any)
		fmt.Printf("%-10v %s\n", c["ID"], c["Name"])
	}
}

// Property represents a connector property.
type Property struct {
	Name  string
	Type  string
	Label string
}

func ListConnectorProperties(connector int) {
	var properties []*Property
	err := callAPI("GET", "apis/connectors/"+strconv.Itoa(connector)+"/properties", nil, &properties)
	if err != nil {
		log.Fatal(err)
	}
	for _, property := range properties {
		fmt.Printf("%-50s %-40s %s\n", property.Label, property.Name, property.Type)
	}
}

func ImportUsersFromConnector(connector int, reimport bool) {
	path := "apis/connectors/" + strconv.Itoa(connector)
	if reimport {
		path += "/reimport"
	} else {
		path += "/import"
	}
	err := callAPI("POST", path, nil, nil)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func GetTransformation(connector int) {
	body := map[string]any{
		"Connector": connector,
	}
	resp, err := callAdmin("admin/transformations/get", body)
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
	resp, err := callAdmin("admin/transformations/update", body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp)
}

func ListUsers() {
	resp, err := callAdmin("admin/list-users", nil)
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

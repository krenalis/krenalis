//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package chichiapis

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
)

// DataSource represents a data source.
type DataSource struct {
	ID   int
	Name string
}

func ListDataSources() {
	var sources []*DataSource
	err := callAPI("GET", "apis/data-sources/", nil, &sources)
	if err != nil {
		log.Fatal(err)
	}
	for _, source := range sources {
		fmt.Printf("%-10v %s\n", source.ID, source.Name)
	}
}

// Property represents a data source property.
type Property struct {
	Name  string
	Type  string
	Label string
}

func ListDataSourcesProperties(connector int) {
	var properties []*Property
	err := callAPI("GET", "apis/data-sources/"+strconv.Itoa(connector)+"/properties", nil, &properties)
	if err != nil {
		log.Fatal(err)
	}
	for _, property := range properties {
		fmt.Printf("%-50s %-40s %s\n", property.Label, property.Name, property.Type)
	}
}

func ImportUsersFromDataSource(connector int, reimport bool) {
	path := "apis/data-sources/" + strconv.Itoa(connector)
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
	var transformation []byte
	err := callAPI("GET", "apis/data-sources/"+strconv.Itoa(connector)+"/transformation", nil, &transformation)
	if err != nil {
		log.Fatal(err)
	}
	_, _ = os.Stdout.Write(transformation)
}

func UpdateTransformation(connector int, transformation []byte) {
	body := bytes.NewReader(transformation)
	err := callAPI("POST", "apis/data-sources/"+strconv.Itoa(connector)+"/transformation", body, nil)
	if err != nil {
		log.Fatal(err)
	}
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

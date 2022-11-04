//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import "fmt"

type Users struct {
	*APIs
}

func (users *Users) Find() ([]map[string]any, error) {
	rows, err := users.myDB.Query("SELECT * FROM `warehouse_users`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	allUsers := []map[string]any{}
	for rows.Next() {
		slice := make([]any, len(columnTypes))
		for i, typ := range columnTypes {
			switch t := typ.DatabaseTypeName(); t {
			case "VARCHAR", "TEXT":
				var s string
				slice[i] = &s
			case "INT", "BIGINT":
				var i int
				slice[i] = &i
			default:
				panic(fmt.Sprintf("%q not supported", t))
			}
		}
		err := rows.Scan(slice...)
		if err != nil {
			return nil, err
		}
		user := map[string]any{}
		for i, typ := range columnTypes {
			user[typ.Name()] = slice[i]
		}
		allUsers = append(allUsers, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return allUsers, nil
}

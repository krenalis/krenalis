//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"chichi/connectors"
	"chichi/pkg/open2b/sql"

	"github.com/open2b/scriggo"
	"github.com/open2b/scriggo/native"
)

// Firehose is the Firehose API used by the connectors.
type Firehose struct {
	connector int
	account   int
	APIs      *APIs
}

// NewFirehose returns a new Firehose for the given connector and account.
func (apis *APIs) NewFirehose(connector, account int) *Firehose {
	return &Firehose{
		connector: connector,
		account:   account,
		APIs:      apis,
	}
}

func (fh *Firehose) SetCursor(cursor string) {
	_, err := fh.APIs.myDB.Table("AccountConnectors").Add(
		map[string]any{
			"account":     fh.account,
			"connector":   fh.connector,
			"user_cursor": cursor,
		},
		sql.Set{
			"user_cursor": cursor,
		},
	)
	if err != nil {
		panic(err)
	}
}

func (fh *Firehose) ApplyConfig(conf map[string]any) {
	return
}

func (fh *Firehose) UpdateGroup(ident connectors.Identity, updateTime int64, properties map[string]any, users []string) {
	return
}

func (fh *Firehose) UpdateUser(ident connectors.Identity, updateTime int64, properties map[string]any, groups []string) {
	data, err := json.Marshal(properties)
	if err != nil {
		panic(err)
	}
	_, err = fh.APIs.myDB.Table("ConnectorsRawUserData").Add(
		map[string]any{
			"account":   fh.account,
			"connector": fh.connector,
			"data":      string(data),
		},
		sql.Set{"data": string(data)},
	)
	if err != nil {
		panic(err)
	}
	goldenRecordData, err := fh.transformProperties(properties)
	if err != nil {
		panic(fmt.Sprintf("cannot transform input properties to output properties: %s", err))
	}
	// Serialize the data to the Golden Record.
	{
		columns := make([]string, len(goldenRecordData))
		i := 0
		for column := range goldenRecordData {
			columns[i] = column
			i++
		}
		sort.Strings(columns)
		query := "INSERT INTO `warehouse_users` (" + strings.Join(columns, ", ") + ") VALUES ("
		for i := range columns {
			if i > 0 {
				query += ","
			}
			query += "?"
		}
		query += ") ON DUPLICATE KEY UPDATE "
		for i := range columns {
			if i > 0 {
				query += ", "
			}
			query += columns[i] + " = ?"
		}
		values := []any{}
		for _, column := range columns {
			values = append(values, goldenRecordData[column])
		}
		_, err := fh.APIs.myDB.Exec(query, append(values, values...)...)
		if err != nil {
			panic(fmt.Sprintf("cannot write data to database: %s", err))
		}
	}
}

func (fh *Firehose) CreateGroup(ident connectors.Identity, creationTime int64, properties map[string]any) {
	return
}

func (fh *Firehose) CreateUser(ident connectors.Identity, creationTime int64, properties map[string]any) {
	return
}

func (fh *Firehose) DeleteGroup(ident connectors.Identity) {
	return
}

func (fh *Firehose) DeleteUser(ident connectors.Identity) {
	return
}

// transformProperties transforms the incoming properties using the
// transformation function specified for the current connector.
func (fh *Firehose) transformProperties(incoming map[string]any) (map[string]any, error) {
	transformationSource, err := fh.APIs.Transformations.Get(fh.account, fh.connector)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve transformation from DB: %s", err)
	}
	fullSourceCode := strings.Replace(
		`{% Transform = {{ transformationFunction }} %}`,
		"{{ transformationFunction }}",
		string(transformationSource),
		1,
	)
	opts := &scriggo.BuildOptions{
		Globals: native.Declarations{
			"Transform": (*func(map[string]any) (map[string]any, error))(nil),
		},
	}
	fs := scriggo.Files{"main.txt": []byte(fullSourceCode)}
	template, err := scriggo.BuildTemplate(fs, "main.txt", opts)
	if err != nil {
		return nil, err
	}
	transform := (func(map[string]any) (map[string]any, error))(nil)
	vars := map[string]interface{}{
		"Transform": &transform,
	}
	err = template.Run(io.Discard, vars, nil)
	if err != nil {
		return nil, err
	}
	data, err := transform(incoming)
	if err != nil {
		return nil, err
	}
	return data, nil
}

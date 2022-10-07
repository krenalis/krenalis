//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"encoding/json"

	"chichi/connectors"
	"chichi/pkg/open2b/sql"
)

// Firehose is the Firehose API used by the connectors.
type Firehose struct {
	connector int
	account   int
	myDB      *sql.DB
}

// NewFirehose returns a new Firehose for the given connector and account.
func (apis *APIs) NewFirehose(connector, account int) *Firehose {
	return &Firehose{
		connector: connector,
		account:   account,
		myDB:      apis.myDB,
	}
}

func (fh *Firehose) SetCursor(cursor string) {
	_, err := fh.myDB.Table("AccountConnectors").Add(
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

func (fh *Firehose) UpdateGroup(ident connectors.Identity, updateTime int64, properties map[string]string, users []string) {
	return
}

func (fh *Firehose) UpdateUser(ident connectors.Identity, updateTime int64, properties map[string]string, groups []string) {
	data, err := json.Marshal(properties)
	if err != nil {
		panic(err)
	}
	_, err = fh.myDB.Table("ConnectorsRawUserData").Add(
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
}

func (fh *Firehose) CreateGroup(ident connectors.Identity, creationTime int64, properties map[string]string) {
	return
}

func (fh *Firehose) CreateUser(ident connectors.Identity, creationTime int64, properties map[string]string) {
	return
}

func (fh *Firehose) DeleteGroup(ident connectors.Identity) {
	return
}

func (fh *Firehose) DeleteUser(ident connectors.Identity) {
	return
}

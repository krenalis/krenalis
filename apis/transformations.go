//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import (
	"chichi/pkg/open2b/sql"
)

type Transformations struct {
	*APIs
}

// A Transformation is a transformation which is applied on the connector's
// properties to obtain the golden record field values. In fact, this is a
// source code of a Go function with the given sign:
//
// func Transform(in map[string]any) map[string]any
type Transformation string

// Get gets the transformation relative to the given installed connector, which
// is identified by the tuple account/connector.
func (transformation *Transformations) Get(account, connector int) (Transformation, error) {
	// TODO(Gianluca): revise table name and column names after the merging of
	// the PR of @retini on OAuth.
	row, err := transformation.APIs.myDB.Table("AccountConnectors").Get(
		sql.Where{"account": account, "connector": connector},
		[]any{"transformation"},
	)
	if err != nil {
		return "", err
	}
	return Transformation(row["transformation"].(string)), nil
}

// Update updates the transformation relative to the given installed connector,
// which is identified by the tuple account/connector.
func (transformation *Transformations) Update(account, connector int, transform Transformation) error {
	// TODO(Gianluca): revise table name and column names after the merging of
	// the PR of @retini on OAuth.
	_, err := transformation.APIs.myDB.Table("AccountConnectors").Update(
		sql.Set{"transformation": transform},
		sql.Where{"account": account, "connector": connector},
	)
	if err != nil {
		return err
	}
	return nil
}

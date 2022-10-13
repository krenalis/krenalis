//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package apis

import "chichi/pkg/open2b/sql"

type Cursors struct {
	*APIs
}

// UserCursor returns the user cursor for the given account and connector.
// If the cursor is not defined, returns the empty string and nil.
func (cursors *Cursors) UserCursor(account, connector int) (string, error) {
	row, err := cursors.myDB.Table("DataSources").Get(
		sql.Where{"account": account, "connector": connector},
		[]any{"userCursor"},
	)
	if err != nil {
		return "", err
	}
	cursor, _ := row["userCursor"].(string)
	return cursor, nil
}

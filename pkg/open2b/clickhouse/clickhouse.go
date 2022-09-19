//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package clickhouse

import "regexp"

// https://clickhouse.com/docs/en/sql-reference/syntax/#identifiers
var identifierRegexp = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]*$`)

// QuoteColumn quotes the given column name.
func QuoteColumn(name string) string {
	// TODO(Gianluca): replace the regular expression with a 'for' loop to
	// increase efficiency.
	if !identifierRegexp.MatchString(name) {
		panic("invalid identifier")
	}
	return "`" + name + "`"
}

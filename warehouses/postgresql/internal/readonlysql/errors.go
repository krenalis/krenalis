// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import (
	"fmt"
	"strings"

	"github.com/meergo/meergo/warehouses"
)

func reject(reason string) error {
	return &warehouses.RejectedReadOnlyQueryError{Msg: fmt.Sprintf("rejected: %s", reason)}
}

func rejectSemicolon() error {
	return reject("semicolon found outside opaque region")
}

func rejectForbiddenToken(token string) error {
	return reject(fmt.Sprintf("forbidden token %s found outside opaque region", strings.ToUpper(token)))
}

func rejectNoVisibleSelect() error {
	return reject("no visible SELECT token found")
}

func rejectQuotedIdentifierFunctionCall() error {
	return reject("function call with quoted identifier is not supported")
}

func rejectQualifiedFunctionCall(name string) error {
	return reject(fmt.Sprintf("qualified function call %s is not allowed", name))
}

func rejectUnterminatedSingleQuotedString() error {
	return reject("unterminated single-quoted string")
}

func rejectUnterminatedDoubleQuotedIdentifier() error {
	return reject("unterminated double-quoted identifier")
}

func rejectUnterminatedBlockComment() error {
	return reject("unterminated block comment")
}

func rejectUnterminatedDollarQuotedString() error {
	return reject("unterminated dollar-quoted string")
}

func rejectSpecialFormNotAllowed(name string) error {
	return reject(fmt.Sprintf("special form %s is not allowed", name))
}

func rejectSpecialFormDoesNotAllowParentheses(name string) error {
	return reject(fmt.Sprintf("special form %s does not allow parentheses", name))
}

func rejectMalformedSpecialFormPrecision(name string) error {
	return reject(fmt.Sprintf("malformed precision for special form %s", name))
}

func rejectLockingClause(clause string) error {
	return reject(fmt.Sprintf("locking clause %s is not allowed", clause))
}

func rejectTypeCast() error {
	return reject("type cast operator :: is not allowed")
}

func newFunctionNotAllowedError(name string) error {
	return &warehouses.RejectedReadOnlyQueryError{
		Msg:      fmt.Sprintf("rejected: function or built-in %s is not allowed in read-only queries", name),
		Function: name,
	}
}

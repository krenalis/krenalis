// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package readonlysql

import (
	"fmt"
	"strings"

	"github.com/krenalis/krenalis/warehouses"
)

func reject(reason string) error {
	return &warehouses.RejectedReadOnlyQueryError{Msg: fmt.Sprintf("query rejected: %s", reason)}
}

func rejectSemicolon() error {
	return reject("multiple statements are not allowed in read-only queries")
}

func rejectForbiddenToken(token string) error {
	return reject(fmt.Sprintf("%s is not allowed in read-only queries", strings.ToUpper(token)))
}

func rejectNoVisibleSelect() error {
	return reject("a read-only SELECT query is required")
}

func rejectQuotedIdentifierFunctionCall() error {
	return reject("function calls with quoted identifiers are not allowed in read-only queries")
}

func rejectIdentifierTooLong() error {
	return reject("identifiers longer than 63 bytes are not allowed in read-only queries")
}

func rejectUnicodeQuotedIdentifier() error {
	return reject(`Unicode quoted identifiers are not allowed in read-only queries`)
}

func rejectUnicodeEscapeStringConstant() error {
	return reject(`Unicode escape strings are not allowed in read-only queries`)
}

func rejectEscapeStringConstant() error {
	return reject(`escape strings are not allowed in read-only queries`)
}

func rejectBitStringConstant() error {
	return reject(`bit strings are not allowed in read-only queries`)
}

func rejectHexStringConstant() error {
	return reject(`hex strings are not allowed in read-only queries`)
}

func rejectQualifiedFunctionCall(name string) error {
	return reject(fmt.Sprintf("schema-qualified function call %s is not allowed in read-only queries", strings.ToUpper(name)))
}

func rejectUnterminatedSingleQuotedString() error {
	return reject("unterminated single-quoted string")
}

func rejectUnterminatedDoubleQuotedIdentifier() error {
	return reject("unterminated double-quoted identifier")
}

func rejectNULInQuotedIdentifier() error {
	return reject("double-quoted identifier contains NUL byte")
}

func rejectUnterminatedBlockComment() error {
	return reject("unterminated block comment")
}

func rejectSpecialFormNotAllowed(name string) error {
	return reject(fmt.Sprintf("%s is not allowed in read-only queries", name))
}

func rejectSpecialFormDoesNotAllowParentheses(name string) error {
	return reject(fmt.Sprintf("%s with parentheses is not allowed in read-only queries", name))
}

func rejectMalformedSpecialFormPrecision(name string) error {
	return reject(fmt.Sprintf("invalid precision for %s in read-only queries", name))
}

func rejectLockingClause(clause string) error {
	return reject(fmt.Sprintf("locking clause %s is not allowed in read-only queries", clause))
}

func rejectTypeCast() error {
	return reject("the :: type cast syntax is not allowed in read-only queries")
}

func rejectDollarSign() error {
	return reject("dollar-quoted strings are not allowed in read-only queries")
}

func newFunctionNotAllowedError(name string) error {
	return &warehouses.RejectedReadOnlyQueryError{
		Msg:      fmt.Sprintf("query rejected: function or built-in %s is not allowed in read-only queries", strings.ToUpper(name)),
		Function: name,
	}
}

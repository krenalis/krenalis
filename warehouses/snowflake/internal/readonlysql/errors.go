// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package readonlysql

import (
	"fmt"
	"strings"

	"github.com/krenalis/krenalis/warehouses"
)

// reject returns a read-only query rejection with reason.
func reject(reason string) error {
	return &warehouses.RejectedReadOnlyQueryError{Msg: fmt.Sprintf("query rejected: %s", reason)}
}

// rejectSemicolon reports a multi-statement rejection.
func rejectSemicolon() error {
	return reject("multiple statements are not allowed in read-only queries")
}

// rejectForbiddenToken reports a rejection for a forbidden SQL token.
func rejectForbiddenToken(token string) error {
	return reject(fmt.Sprintf("%s is not allowed in read-only queries", strings.ToUpper(token)))
}

// rejectNoVisibleSelect reports that no SELECT statement was found.
func rejectNoVisibleSelect() error {
	return reject("a read-only SELECT query is required")
}

// rejectUnsupportedTopLevelStatement reports a non-SELECT top-level statement.
func rejectUnsupportedTopLevelStatement(token string) error {
	if token == "" {
		return rejectNoVisibleSelect()
	}
	return reject(fmt.Sprintf("top-level %s statements are not allowed in read-only queries", strings.ToUpper(token)))
}

// rejectQuotedIdentifierFunctionCall reports a quoted function-name rejection.
func rejectQuotedIdentifierFunctionCall() error {
	return reject("function calls with quoted identifiers are not allowed in read-only queries")
}

// rejectQualifiedFunctionCall reports a qualified function-call rejection.
func rejectQualifiedFunctionCall(name string) error {
	return reject(fmt.Sprintf("qualified function call %s is not allowed in read-only queries", strings.ToUpper(name)))
}

// rejectUnterminatedSingleQuotedString reports an unterminated string literal.
func rejectUnterminatedSingleQuotedString() error {
	return reject("unterminated single-quoted string")
}

// rejectUnterminatedDoubleQuotedIdentifier reports an unterminated identifier.
func rejectUnterminatedDoubleQuotedIdentifier() error {
	return reject("unterminated double-quoted identifier")
}

// rejectNULInQuotedIdentifier reports a NUL byte in a quoted identifier.
func rejectNULInQuotedIdentifier() error {
	return reject("double-quoted identifier contains NUL byte")
}

// rejectNonASCIICharacter reports an unsupported visible non-ASCII byte.
func rejectNonASCIICharacter() error {
	return reject("non-ASCII characters outside strings, comments, and quoted identifiers are not allowed in read-only queries")
}

// rejectUnterminatedBlockComment reports an unterminated block comment.
func rejectUnterminatedBlockComment() error {
	return reject("unterminated block comment")
}

// rejectDollarQuotedString reports a dollar-quoted string rejection.
func rejectDollarQuotedString() error {
	return reject("dollar-quoted string constants are not allowed in read-only queries")
}

// rejectTypeCast reports a :: cast rejection.
func rejectTypeCast() error {
	return reject("the :: type cast syntax is not allowed in read-only queries")
}

// rejectStageReference reports a Snowflake stage-reference rejection.
func rejectStageReference() error {
	return reject("stage references are not allowed in read-only queries")
}

// rejectSessionVariable reports a session-variable reference rejection.
func rejectSessionVariable() error {
	return reject("session variable references are not allowed in read-only queries")
}

// rejectTableFunction reports a TABLE(...) rejection.
func rejectTableFunction() error {
	return reject("TABLE(...) is not allowed in read-only queries")
}

// rejectPipeOperator reports a command-result pipe rejection.
func rejectPipeOperator() error {
	return reject("pipe command-result processing is not allowed in read-only queries")
}

// newFunctionNotAllowedError returns a rejection for name.
func newFunctionNotAllowedError(name string) error {
	return &warehouses.RejectedReadOnlyQueryError{
		Msg:      fmt.Sprintf("query rejected: function or built-in %s is not allowed in read-only queries", strings.ToUpper(name)),
		Function: name,
	}
}

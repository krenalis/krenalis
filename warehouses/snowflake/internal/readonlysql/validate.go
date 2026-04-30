// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package readonlysql validates whether a SQL query can be accepted as
// read-only by Snowflake QueryReadOnly implementations.
//
// Validation is conservative: queries that cannot be classified safely as
// read-only are rejected, even if Snowflake would accept them.
package readonlysql

// ValidateReadOnly reports whether query is accepted as read-only.
//
// It returns nil if the query is accepted. Otherwise, it returns a
// warehouses.RejectedReadOnlyQueryError describing the reason for the
// rejection.
func ValidateReadOnly(query string) error {
	var seenMainSelect bool
	var seenTopLevel bool
	var expectCTEName bool
	var inWithPrelude bool
	var parenDepth int
	var lastVisibleChar byte
	var previousWordWasWithinAfterCloseParen bool

	for i := 0; i < len(query); {
		switch c := query[i]; {
		case isSpace(c):
			i++
		case c == '-' && i+1 < len(query) && query[i+1] == '-':
			i = skipLineComment(query, i)
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			next, err := skipBlockComment(query, i)
			if err != nil {
				return err
			}
			i = next
		case c == '\'':
			next, err := skipSingleQuotedString(query, i)
			if err != nil {
				return err
			}
			lastVisibleChar = query[next-1]
			previousWordWasWithinAfterCloseParen = false
			i = next
		case c == '"':
			next, err := scanDoubleQuotedIdentifier(query, i)
			if err != nil {
				return err
			}
			nextChar, err := nextVisibleChar(query, next)
			if err != nil {
				return err
			}
			if nextChar == '(' {
				return rejectQuotedIdentifierFunctionCall()
			}
			lastVisibleChar = query[next-1]
			previousWordWasWithinAfterCloseParen = false
			i = next
		case c == ';':
			next, err := skipIgnored(query, i+1)
			if err != nil {
				return err
			}
			if next < len(query) {
				return rejectSemicolon()
			}
			i = len(query)
		case c == ':' && i+1 < len(query) && query[i+1] == ':':
			return rejectTypeCast()
		case c == '-' && i+2 < len(query) && query[i+1] == '>' && query[i+2] == '>':
			return rejectPipeOperator()
		case c == '@':
			return rejectStageReference()
		case c == '$':
			if i+1 < len(query) && query[i+1] == '$' {
				return rejectDollarQuotedString()
			}
			return rejectSessionVariable()
		case isWordStart(c):
			name, err := scanIdentifierChain(query, i)
			if err != nil {
				return err
			}
			isWithinGroup := previousWordWasWithinAfterCloseParen && name.isWord("group") && !name.isQualified && lastVisibleChar != '.'
			previousWordWasWithinAfterCloseParen = name.isWord("within") && !name.isQualified && lastVisibleChar == ')'
			if !seenTopLevel {
				seenTopLevel = true
				if !name.isSelect() && !name.isWord("with") {
					if isForbiddenToken(name.token) {
						return rejectForbiddenToken(name.token)
					}
					if name.isWord("desc") {
						return rejectForbiddenToken(name.token)
					}
					return rejectUnsupportedTopLevelStatement(name.token)
				}
				if name.isWord("with") {
					expectCTEName = true
					inWithPrelude = true
					lastVisibleChar = query[name.next-1]
					i = name.next
					continue
				}
			}
			if expectCTEName && !name.isQualified {
				if name.isWord("recursive") {
					lastVisibleChar = query[name.next-1]
					i = name.next
					continue
				}
				if isForbiddenToken(name.token) || isDisallowedSpecialForm(name.token) {
					return rejectForbiddenToken(name.token)
				}
				expectCTEName = false
				lastVisibleChar = query[name.next-1]
				i = name.next
				continue
			}
			if inWithPrelude && parenDepth == 0 && name.isWord("values") {
				return rejectUnsupportedTopLevelStatement(name.token)
			}
			if name.followedByParen {
				if name.isTable() && !name.isQualified {
					return rejectTableFunction()
				}
				if isWithinGroup {
					lastVisibleChar = query[name.next-1]
					i = name.next
					continue
				}
				if isNonFunctionCallKeyword(name.token) && !name.isQualified {
					lastVisibleChar = query[name.next-1]
					i = name.next
					continue
				}
				if lastVisibleChar == '.' || name.isQualified {
					return rejectQualifiedFunctionCall(name.normalizedChain(query))
				}
				if !isAllowedFunction(name.token) {
					return newFunctionNotAllowedError(name.normalizedToken())
				}
				lastVisibleChar = query[name.next-1]
				i = name.next
				continue
			}
			if name.isSelect() && parenDepth == 0 {
				seenMainSelect = true
				inWithPrelude = false
			}
			if isDisallowedSpecialForm(name.token) {
				return newFunctionNotAllowedError(name.normalizedToken())
			}
			if isForbiddenToken(name.token) {
				return rejectForbiddenToken(name.token)
			}
			lastVisibleChar = query[name.next-1]
			i = name.next
		default:
			if c >= 0x80 {
				return rejectNonASCIICharacter()
			}
			switch c {
			case '(':
				parenDepth++
			case ')':
				if parenDepth > 0 {
					parenDepth--
				}
			case ',':
				if inWithPrelude && parenDepth == 0 {
					expectCTEName = true
				}
			}
			lastVisibleChar = c
			previousWordWasWithinAfterCloseParen = false
			i++
		}
	}

	if !seenMainSelect {
		return rejectNoVisibleSelect()
	}

	return nil
}

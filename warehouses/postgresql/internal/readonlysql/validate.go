// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package readonlysql validates whether a SQL query can be accepted as
// read-only by PostgreSQL QueryReadOnly implementations.
//
// Validation is conservative: queries that cannot be classified safely as
// read-only are rejected, even if PostgreSQL would accept them.
//
// This package is intended for PostgreSQL 14 through 18.
package readonlysql

import (
	"strings"
)

// lockingClauseState tracks partial recognition of a FOR ... locking clause.
type lockingClauseState uint8

const (
	lockingClauseNone lockingClauseState = iota
	lockingClauseFor
	lockingClauseForKey
)

// ValidateReadOnly reports whether query is accepted as read-only.
//
// It returns nil if the query is accepted. Otherwise, it returns a
// *warehouses.RejectedReadOnlyQueryError describing the reason for the
// rejection.
//
// Validation is conservative: queries that cannot be classified safely as
// read-only are rejected, even if PostgreSQL would accept them.
func ValidateReadOnly(query string) error {
	// This checker is intentionally conservative and not a full SQL parser.
	// It recognizes only a small set of opaque regions with certainty, scans
	// visible SQL words using the approximation [A-Za-z_][A-Za-z0-9_]*, and
	// applies a blacklist of forbidden non-SELECT statement tokens.
	//
	// Function calls are accepted only for non-qualified names on an explicit
	// allowlist. A separate allowlist handles a few PostgreSQL special forms
	// without parentheses. Rejections for non-allowlisted non-qualified
	// function calls use RejectedReadOnlyQueryError with the normalized
	// function name in its Function field. All qualified function calls are
	// rejected. This policy relies on the execution assumption that PostgreSQL
	// runs with search_path fixed to public.
	//
	// The lexical and special-form behavior used here is intentionally limited
	// to rules that are stable across PostgreSQL 14 through 18. MERGE remains
	// blacklisted for the whole range: it is a real command from 15 onward,
	// while on 14 it remains a syntax error, which is still conservative for
	// this classifier. Future PostgreSQL versions require review.

	// As a special case, accept a single trailing statement terminator
	// only when it is the final byte of the query.
	query = strings.TrimSuffix(query, ";")

	var seenSelect bool
	var lastVisibleChar byte
	var lockingClauseState lockingClauseState

	for i := 0; i < len(query); {
		switch c := query[i]; {
		case isSpace(c):
			i++
		case hasUnicodeQuotedIdentifierPrefix(query, i):
			return rejectUnicodeQuotedIdentifier()
		case hasUnicodeEscapeStringConstantPrefix(query, i):
			return rejectUnicodeEscapeStringConstant()
		case hasEscapeStringConstantPrefix(query, i):
			return rejectEscapeStringConstant()
		case hasBitStringConstantPrefix(query, i):
			return rejectBitStringConstant()
		case hasHexStringConstantPrefix(query, i):
			return rejectHexStringConstant()
		case c == ':' && i+1 < len(query) && query[i+1] == ':':
			return rejectTypeCast()
		case c == ';':
			return rejectSemicolon()
		case c == '\'':
			next, err := skipSingleQuotedString(query, i)
			if err != nil {
				return err
			}
			lastVisibleChar = query[next-1]
			lockingClauseState = lockingClauseNone
			i = next
		case c == '"':
			next, byteLen, err := scanDoubleQuotedIdentifier(query, i)
			if err != nil {
				return err
			}
			if byteLen > maxIdentifierBytes {
				return rejectIdentifierTooLong()
			}
			nextChar, err := nextVisibleChar(query, next)
			if err != nil {
				return err
			}
			if nextChar == '(' {
				return rejectQuotedIdentifierFunctionCall()
			}
			lastVisibleChar = query[next-1]
			lockingClauseState = lockingClauseNone
			i = next
		case c == '-' && i+1 < len(query) && query[i+1] == '-':
			i = skipLineComment(query, i)
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			next, err := skipBlockComment(query, i)
			if err != nil {
				return err
			}
			i = next
		case c == '$':
			return rejectDollarSign()
		case isWordStart(c):
			name, err := scanIdentifierChain(query, i)
			if err != nil {
				return err
			}
			switch lockingClauseState {
			case lockingClauseFor:
				switch {
				case !name.isQualified && name.isWord("KEY"):
					lockingClauseState = lockingClauseForKey
				case !name.isQualified && name.isWord("SHARE"):
					return rejectLockingClause("FOR SHARE")
				case !name.isQualified && name.isWord("FOR"):
					lockingClauseState = lockingClauseFor
				default:
					lockingClauseState = lockingClauseNone
				}
			case lockingClauseForKey:
				switch {
				case !name.isQualified && name.isWord("SHARE"):
					return rejectLockingClause("FOR KEY SHARE")
				case !name.isQualified && name.isWord("FOR"):
					lockingClauseState = lockingClauseFor
				default:
					lockingClauseState = lockingClauseNone
				}
			case lockingClauseNone:
				if !name.isQualified && name.isWord("FOR") {
					lockingClauseState = lockingClauseFor
				}
			}
			if handled, next, err := handleSpecialForm(query, name); handled {
				if err != nil {
					return err
				}
				lastVisibleChar = query[next-1]
				i = next
				continue
			}
			if name.followedByParen {
				if isNonFunctionCallKeyword(name.token) && !name.isQualified {
					lastVisibleChar = query[name.next-1]
					i = name.next
					continue
				}
				if lastVisibleChar == '.' {
					return rejectQualifiedFunctionCall(name.normalizedChain(query))
				}
				if name.isQualified {
					return rejectQualifiedFunctionCall(name.normalizedChain(query))
				}
				if !isAllowedFunction(name.token) {
					return newFunctionNotAllowedError(name.normalizedToken())
				}
				if name.isSelect() {
					seenSelect = true
				}
				lastVisibleChar = query[name.next-1]
				i = name.next
				continue
			}
			if name.isSelect() {
				seenSelect = true
			}
			lastVisibleChar = query[name.next-1]
			if isForbiddenToken(name.token) {
				return rejectForbiddenToken(name.token)
			}
			i = name.next
		default:
			lastVisibleChar = c
			lockingClauseState = lockingClauseNone
			i++
		}
	}

	if !seenSelect {
		return rejectNoVisibleSelect()
	}

	return nil
}

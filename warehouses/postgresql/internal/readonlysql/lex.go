// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import "strings"

// maxIdentifierBytes is the default PostgreSQL identifier limit in bytes.
// PostgreSQL truncates longer identifiers by default; this checker rejects them
// conservatively instead of modeling truncation semantics.
const maxIdentifierBytes = 63

type scannedName struct {
	token          string
	start          int
	isQualified    bool
	isFunctionCall bool
	next           int
}

// isSelect reports whether n is the SELECT keyword.
func (n scannedName) isSelect() bool {
	return asciiEqualFold(n.token, "select")
}

// isWord reports whether n matches word case-insensitively.
func (n scannedName) isWord(word string) bool {
	return asciiEqualFold(n.token, word)
}

// normalizedToken returns the lowercase ASCII form of the first identifier
// segment. It is intended for error paths.
func (n scannedName) normalizedToken() string {
	return asciiLowerString(n.token)
}

// normalizedChain returns the lowercase ASCII dotted identifier chain. It is
// intended for error paths.
func (n scannedName) normalizedChain(sql string) string {
	if !n.isQualified {
		return n.normalizedToken()
	}

	var b strings.Builder
	b.Grow(n.next - n.start)

	for i := n.start; i < n.next; {
		partEnd := scanIdentifierEnd(sql, i)
		for j := i; j < partEnd; j++ {
			b.WriteByte(asciiLower(sql[j]))
		}

		next, err := skipIgnored(sql, partEnd)
		if err != nil || next >= n.next || sql[next] != '.' {
			break
		}

		b.WriteByte('.')
		i, err = skipIgnored(sql, next+1)
		if err != nil || i >= n.next {
			break
		}
	}

	return b.String()
}

// scanIdentifierChain scans an identifier, optionally followed by dotted parts.
func scanIdentifierChain(sql string, start int) (scannedName, error) {
	tokenEnd := scanIdentifierEnd(sql, start)
	if tokenEnd-start > maxIdentifierBytes {
		return scannedName{}, rejectIdentifierTooLong()
	}
	lastEnd := tokenEnd
	isQualified := false

	for {
		i, err := skipIgnored(sql, lastEnd)
		if err != nil {
			return scannedName{}, err
		}
		if i >= len(sql) || sql[i] != '.' {
			break
		}

		i++
		i, err = skipIgnored(sql, i)
		if err != nil {
			return scannedName{}, err
		}
		if i >= len(sql) || !isWordStart(sql[i]) {
			break
		}

		partEnd := scanIdentifierEnd(sql, i)
		if partEnd-i > maxIdentifierBytes {
			return scannedName{}, rejectIdentifierTooLong()
		}
		isQualified = true
		lastEnd = partEnd
	}

	callStart, err := skipIgnored(sql, lastEnd)
	if err != nil {
		return scannedName{}, err
	}

	return scannedName{
		token:          sql[start:tokenEnd],
		start:          start,
		isQualified:    isQualified,
		isFunctionCall: callStart < len(sql) && sql[callStart] == '(',
		next:           lastEnd,
	}, nil
}

// scanIdentifierEnd returns the first index after the identifier at start.
func scanIdentifierEnd(sql string, start int) int {
	end := start + 1
	for end < len(sql) && isWordChar(sql[end]) {
		end++
	}
	return end
}

// nextVisibleChar returns the next non-ignored byte starting at start.
func nextVisibleChar(sql string, start int) (byte, error) {
	start, err := skipIgnored(sql, start)
	if err != nil {
		return 0, err
	}
	if start >= len(sql) {
		return 0, nil
	}
	return sql[start], nil
}

// scanDoubleQuotedIdentifier scans a double-quoted identifier and returns the
// first index after it together with the identifier length in bytes after
// unescaping doubled double quotes.
func scanDoubleQuotedIdentifier(sql string, start int) (int, int, error) {
	byteLen := 0
	for i := start + 1; i < len(sql); i++ {
		if sql[i] != '"' {
			if sql[i] == 0 {
				return 0, 0, rejectNULInQuotedIdentifier()
			}
			byteLen++
			continue
		}
		if i+1 < len(sql) && sql[i+1] == '"' {
			byteLen++
			i++
			continue
		}
		return i + 1, byteLen, nil
	}
	return 0, 0, rejectUnterminatedDoubleQuotedIdentifier()
}

// hasUnicodeQuotedIdentifierPrefix reports whether a Unicode quoted identifier
// starts at start using PostgreSQL's U&"..." introducer.
func hasUnicodeQuotedIdentifierPrefix(sql string, start int) bool {
	if start+2 >= len(sql) {
		return false
	}
	if sql[start] != 'U' && sql[start] != 'u' {
		return false
	}
	return sql[start+1] == '&' && sql[start+2] == '"'
}

// hasUnicodeEscapeStringConstantPrefix reports whether a PostgreSQL Unicode
// escape string constant starts at start using the U&'...' introducer.
func hasUnicodeEscapeStringConstantPrefix(sql string, start int) bool {
	if start+2 >= len(sql) {
		return false
	}
	if sql[start] != 'U' && sql[start] != 'u' {
		return false
	}
	return sql[start+1] == '&' && sql[start+2] == '\''
}

// hasPrefixedSingleQuotedForm reports whether a single-letter PostgreSQL form
// starts at start using an introducer such as E'...'.
func hasPrefixedSingleQuotedForm(sql string, start int, upper byte) bool {
	if start+1 >= len(sql) {
		return false
	}
	return (sql[start] == upper || sql[start] == upper+'a'-'A') && sql[start+1] == '\''
}

// hasEscapeStringConstantPrefix reports whether a PostgreSQL escape string
// constant starts at start using the E'...' introducer.
func hasEscapeStringConstantPrefix(sql string, start int) bool {
	return hasPrefixedSingleQuotedForm(sql, start, 'E')
}

// hasBitStringConstantPrefix reports whether a PostgreSQL bit string constant
// starts at start using the B'...' introducer.
func hasBitStringConstantPrefix(sql string, start int) bool {
	return hasPrefixedSingleQuotedForm(sql, start, 'B')
}

// hasHexStringConstantPrefix reports whether a PostgreSQL hex string constant
// starts at start using the X'...' introducer.
func hasHexStringConstantPrefix(sql string, start int) bool {
	return hasPrefixedSingleQuotedForm(sql, start, 'X')
}

// skipIgnored skips spaces and comments.
func skipIgnored(sql string, start int) (int, error) {
	for {
		start = skipSpaces(sql, start)
		switch {
		case start+1 < len(sql) && sql[start] == '-' && sql[start+1] == '-':
			start = skipLineComment(sql, start)
		case start+1 < len(sql) && sql[start] == '/' && sql[start+1] == '*':
			next, err := skipBlockComment(sql, start)
			if err != nil {
				return 0, err
			}
			start = next
		default:
			return start, nil
		}
	}
}

// skipSpaces skips ASCII whitespace.
func skipSpaces(sql string, start int) int {
	for start < len(sql) && isSpace(sql[start]) {
		start++
	}
	return start
}

// skipSingleQuotedString skips a single-quoted string literal.
func skipSingleQuotedString(sql string, start int) (int, error) {
	for i := start + 1; i < len(sql); i++ {
		if sql[i] != '\'' {
			continue
		}
		if i+1 < len(sql) && sql[i+1] == '\'' {
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, rejectUnterminatedSingleQuotedString()
}

// skipDoubleQuotedIdentifier skips a double-quoted identifier.
func skipDoubleQuotedIdentifier(sql string, start int) (int, error) {
	next, _, err := scanDoubleQuotedIdentifier(sql, start)
	return next, err
}

// skipLineComment skips a line comment.
func skipLineComment(sql string, start int) int {
	i := start + 2
	for i < len(sql) && sql[i] != '\n' && sql[i] != '\r' {
		i++
	}
	return i
}

// skipBlockComment skips a possibly nested block comment.
func skipBlockComment(sql string, start int) (int, error) {
	depth := 1
	for i := start + 2; i < len(sql)-1; i++ {
		switch {
		case sql[i] == '/' && sql[i+1] == '*':
			depth++
			i++
		case sql[i] == '*' && sql[i+1] == '/':
			depth--
			i++
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, rejectUnterminatedBlockComment()
}

// isSpace reports whether b is ASCII whitespace.
func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

// isWordStart reports whether b can start an SQL word.
func isWordStart(b byte) bool {
	return b == '_' || ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z')
}

// isWordChar reports whether b can continue an SQL word.
func isWordChar(b byte) bool {
	return isWordStart(b) || ('0' <= b && b <= '9')
}

// isDigit reports whether b is an ASCII decimal digit.
func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

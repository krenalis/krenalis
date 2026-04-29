// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import "strings"

type scannedName struct {
	token           string
	start           int
	isQualified     bool
	followedByParen bool
	next            int
}

// isSelect reports whether n is the SELECT keyword.
func (n scannedName) isSelect() bool {
	return asciiEqualFold(n.token, "select")
}

// isTable reports whether n is the TABLE keyword.
func (n scannedName) isTable() bool {
	return asciiEqualFold(n.token, "table")
}

// isWord reports whether n matches word case-insensitively.
func (n scannedName) isWord(word string) bool {
	return asciiEqualFold(n.token, word)
}

// normalizedToken returns the lowercase ASCII form of n's first segment.
func (n scannedName) normalizedToken() string {
	return asciiLowerString(n.token)
}

// normalizedChain returns the lowercase ASCII dotted chain.
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
		if err != nil || i >= n.next || !isWordStart(sql[i]) {
			break
		}
	}

	return b.String()
}

// scanIdentifierChain scans an unquoted identifier and following dotted parts.
func scanIdentifierChain(sql string, start int) (scannedName, error) {
	tokenEnd := scanIdentifierEnd(sql, start)
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

		lastEnd = scanIdentifierEnd(sql, i)
		isQualified = true
	}

	callStart, err := skipIgnored(sql, lastEnd)
	if err != nil {
		return scannedName{}, err
	}

	return scannedName{
		token:           sql[start:tokenEnd],
		start:           start,
		isQualified:     isQualified,
		followedByParen: callStart < len(sql) && sql[callStart] == '(',
		next:            lastEnd,
	}, nil
}

// scanIdentifierEnd returns the first index after the unquoted identifier.
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
		if sql[i] == '\\' {
			i++
			continue
		}
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

// scanDoubleQuotedIdentifier scans a double-quoted identifier.
func scanDoubleQuotedIdentifier(sql string, start int) (int, error) {
	for i := start + 1; i < len(sql); i++ {
		if sql[i] != '"' {
			if sql[i] == 0 {
				return 0, rejectNULInQuotedIdentifier()
			}
			continue
		}
		if i+1 < len(sql) && sql[i+1] == '"' {
			i++
			continue
		}
		return i + 1, nil
	}
	return 0, rejectUnterminatedDoubleQuotedIdentifier()
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

// isWordStart reports whether b can start an unquoted identifier.
func isWordStart(b byte) bool {
	return b == '_' || ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z')
}

// isWordChar reports whether b can continue an unquoted identifier.
func isWordChar(b byte) bool {
	return isWordStart(b) || ('0' <= b && b <= '9') || b == '$'
}

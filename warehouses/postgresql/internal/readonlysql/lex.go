// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package readonlysql

import "strings"

type scannedName struct {
	token          string
	normalized     string
	isQualified    bool
	isFunctionCall bool
	next           int
}

// isSelect reports whether n is the SELECT keyword.
func (n scannedName) isSelect() bool {
	return strings.EqualFold(n.token, "SELECT")
}

// isWord reports whether n matches word case-insensitively.
func (n scannedName) isWord(word string) bool {
	return strings.EqualFold(n.token, word)
}

// scanIdentifierChain scans an identifier, optionally followed by dotted parts.
func scanIdentifierChain(sql string, start int) (scannedName, error) {
	tokenEnd := scanIdentifierEnd(sql, start)
	parts := []string{strings.ToLower(sql[start:tokenEnd])}
	lastEnd := tokenEnd

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
		parts = append(parts, strings.ToLower(sql[i:partEnd]))
		lastEnd = partEnd
	}

	callStart, err := skipIgnored(sql, lastEnd)
	if err != nil {
		return scannedName{}, err
	}

	return scannedName{
		token:          sql[start:tokenEnd],
		normalized:     strings.Join(parts, "."),
		isQualified:    len(parts) > 1,
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
	for i := start + 1; i < len(sql); i++ {
		if sql[i] != '"' {
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

// skipDollarQuotedString skips a dollar-quoted string if one starts at start.
func skipDollarQuotedString(sql string, start int) (int, bool, error) {
	tag, ok := parseDollarQuoteTag(sql, start)
	if !ok {
		return 0, false, nil
	}

	delimiter := "$" + tag + "$"
	bodyStart := start + len(delimiter)
	end := strings.Index(sql[bodyStart:], delimiter)
	if end < 0 {
		return 0, true, rejectUnterminatedDollarQuotedString()
	}
	return bodyStart + end + len(delimiter), true, nil
}

// parseDollarQuoteTag parses the tag of a dollar-quoted string delimiter.
func parseDollarQuoteTag(sql string, start int) (string, bool) {
	if start+1 >= len(sql) || sql[start] != '$' {
		return "", false
	}
	if sql[start+1] == '$' {
		return "", true
	}
	if !isTagStart(sql[start+1]) {
		return "", false
	}

	i := start + 2
	for i < len(sql) && isTagChar(sql[i]) {
		i++
	}
	if i >= len(sql) || sql[i] != '$' {
		return "", false
	}
	return sql[start+1 : i], true
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
	return isWordStart(b) || ('0' <= b && b <= '9') || b == '$'
}

// isTagStart reports whether b can start a dollar-quote tag.
func isTagStart(b byte) bool {
	return isWordStart(b)
}

// isTagChar reports whether b can continue a dollar-quote tag.
func isTagChar(b byte) bool {
	return isWordStart(b) || ('0' <= b && b <= '9')
}

// isDigit reports whether b is an ASCII decimal digit.
func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package dotenv

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// maxEnvFileSize is the 10 MiB limit for .env files.
const maxEnvFileSize = 10 * 1024 * 1024

// Load reads an ".env" file and sets its variables in the process environment.
// Each line must follow the format NAME=VALUE. Lines that are empty, missing
// the "=" separator, or whose keys contain spaces or a NUL character are
// ignored.
//
// Values that contain a NUL character are also ignored. Leading and trailing
// spaces around NAME are removed. VALUE can be:
// - unquoted: trailing spaces are preserved
// - double-quoted: supports escapes for \n, \r, \t, \, and "
// - single-quoted: supports escaping only for '
//
// A line may optionally start with "export" followed by one or more spaces.
// A line starting with "#" (after optional spaces) or containing " #" marks a
// comment.
//
// Variables defined in the file always replace existing environment values.
func Load(name string) error {

	// Read the file if exists.
	name, err := filepath.Abs(name)
	if err != nil {
		return fmt.Errorf("cannot resolve path for %q: %w", ".env", err)
	}
	fi, err := os.Open(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cannot open %q: %w", name, err)
	}
	defer fi.Close()
	st, err := fi.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat %q: %w", name, err)
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("cannot parse %q: file is not a regular file", name)
	}
	r := io.LimitReader(fi, maxEnvFileSize+1)
	rest, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("cannot read %q: %w", name, err)
	}
	if len(rest) > maxEnvFileSize {
		return fmt.Errorf("cannot parse %q: file size exceeds 10 MiB limit", name)
	}

	const bom = "\uFEFF"
	if bytes.HasPrefix(rest, []byte(bom)) {
		rest = rest[len(bom):]
	}

	const singleQuote = '\''
	const doubleQuote = '"'

	var found bool
	var k, v []byte

	parseErr := func(line int, description string) error {
		return fmt.Errorf("cannot parse %q: line %d %s", name, line, description)
	}

	// Parse the rows.
	for line := 1; len(rest) > 0; line++ {

		v, rest, _ = bytes.Cut(rest, []byte("\n"))
		k, v, found = bytes.Cut(v, []byte("="))
		if !found {
			continue
		}
		// Key: remove leading and trailing white spaces and validate.
		k = bytes.Trim(k, " \t")
		if i := bytes.IndexAny(k, " \t\x00"); i != -1 {
			// Allow only "export KEY=value".
			if k[i] == 0 || !bytes.Equal(k[:i], []byte("export")) {
				continue
			}
			k = bytes.TrimLeft(k[i+1:], " \t")
			if bytes.IndexAny(k, " \t\x00") != -1 {
				continue
			}
		}
		if len(k) == 0 {
			continue
		}
		// Value: remove trailing '\r' and leading white spaces.
		if n := len(v); n > 0 && v[n-1] == '\r' {
			v = v[:n-1]
		}
		v = bytes.TrimLeft(v, " \t\r\n")
		if len(v) > 0 {
			switch c := v[0]; c {
			case singleQuote, doubleQuote:
				var i int
				for i = 1; i < len(v); i++ {
					if v[i] == c {
						break
					}
					if v[i] != '\\' {
						continue
					}
					if i == len(v)-1 {
						continue
					}
					if c == singleQuote {
						if v[i+1] == singleQuote {
							v = append(v[:i], v[i+1:]...)
						}
						continue
					}
					v = append(v[:i], v[i+1:]...)
					switch v[i] {
					case 'n':
						v[i] = '\n'
					case 'r':
						v[i] = '\r'
					case 't':
						v[i] = '\t'
					case '\\', '"':
						// leave them as they are
					default:
						return parseErr(line, "contains an invalid escape sequence")
					}
				}
				if i == len(v) {
					return parseErr(line, "has an unterminated quoted value")
				}
				var s []byte
				v, s = v[1:i], v[i+1:]
				// Verify that v is followed only by spaces or by a comment starting with a space.
				if n := len(s); n > 0 {
					if s = bytes.TrimLeft(s, " \t\r\n"); len(s) > 0 && (s[0] != '#' || n == len(s)) {
						return parseErr(line, "contains characters after the closing quote")
					}
				}
			default:
				// Remove inline comments introduced by " #".
				end := len(v)
				for i := 1; i < len(v); i++ {
					if v[i] == '#' && v[i-1] == ' ' {
						end = i - 1
						break
					}
				}
				if end != len(v) {
					if end < 0 {
						end = 0
					}
					v = bytes.TrimRight(v[:end], " \t\r\n")
				}
			}
		}
		// Validate the value.
		if bytes.IndexByte(v, 0) != -1 {
			continue
		}
		// Set the variabile.
		_ = os.Setenv(string(k), string(v))
	}

	return nil
}

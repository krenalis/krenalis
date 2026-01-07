// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package validation

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

type URLValidationFlag int

const (
	NoPath URLValidationFlag = 1 << iota
	NoQuery
)

func hasURLValidationFlag(f, flag URLValidationFlag) bool {
	return f&flag != 0
}

// ParseURL parses s as an HTTP(S) URL and returns it in normalized form.
//
// The URL must use scheme "http" or "https", include a host, and must not
// include user info or a fragment.
//
// Flags enable additional constraints:
//   - NoPath: the URL must have no path or path "/", and the result uses path "/"
//   - NoQuery: the URL must have no query (including a lone "?"), and the result
//     omits the query entirely
//
// If s is empty, ParseURL returns an empty string and a nil error.
func ParseURL(s string, flags URLValidationFlag) (string, error) {
	if s == "" {
		return "", nil
	}
	if s[0] == ' ' {
		return "", errors.New(`it starts with a space`)
	}
	if s[len(s)-1] == ' ' {
		return "", errors.New(`it ends with a space`)
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New(`scheme must be "http" or "https"`)
	}
	if u.User != nil {
		return "", errors.New("user and password cannot be specified")
	}
	if u.Host == "" {
		return "", errors.New("host must be specified")
	}
	if err := ValidateHost(u.Hostname()); err != nil {
		return "", err
	}
	port := u.Port()
	if port != "" {
		if _, err := ValidatePortString(port); err != nil {
			return "", err
		}
	}
	if hasURLValidationFlag(flags, NoPath) {
		if u.Path != "" && u.Path != "/" {
			return "", errors.New(`path must be "/"`)
		}
	}
	if hasURLValidationFlag(flags, NoQuery) {
		if u.RawQuery != "" || u.ForceQuery {
			return "", errors.New("query cannot be specified")
		}
	}
	if strings.IndexByte(s, '#') != -1 {
		return "", errors.New("fragment cannot be specified")
	}
	var normalized bool
	if port != "" && port[0] == '0' {
		port = strings.TrimLeft(port, "0")
		u.Host = net.JoinHostPort(u.Hostname(), port)
		normalized = true
	}
	if u.Scheme == "http" && port == "80" || u.Scheme == "https" && port == "443" {
		i := strings.LastIndex(u.Host, ":")
		u.Host = u.Host[:i]
		normalized = true
	}
	if u.Path == "" {
		u.Path = "/"
		normalized = true
	}
	if normalized {
		s = u.String()
	}
	return s, nil
}

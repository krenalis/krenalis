// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package validation

import (
	"strings"
)

// ParseBearer extracts a Bearer token from an Authorization header. It reports
// whether the header uses the Bearer scheme and contains a non-empty token.
func ParseBearer(header string) (string, bool) {
	const scheme = "Bearer"
	if len(header) < len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return "", false
	}
	token := header[len(scheme):]
	if token == "" || (token[0] != ' ' && token[0] != '\t') {
		return "", false
	}
	token = strings.TrimLeft(token, " \t")
	if token == "" {
		return "", false
	}
	return token, true
}

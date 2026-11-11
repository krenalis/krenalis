// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// SessionConnector checks each new connection in a campaign overlay. The
// underlying connector must set the expected parameters at login.
type SessionConnector struct {
	driver.Connector
	Timezone string
	// Profile selects standard (the zero value), timeout or cancellation.
	Profile string
}

// Connect opens a connection and checks its effective session parameters.
// Connections that fail verification are closed before returning the error.
func (c SessionConnector) Connect(ctx context.Context) (connection driver.Conn, err error) {

	if c.Connector == nil || c.Timezone != "UTC" && c.Timezone != "America/Los_Angeles" {
		return nil, fmt.Errorf("test/snowflaketester: invalid campaign session connector")
	}
	profile, timeout := c.Profile, "60"
	switch profile {
	case "", "standard":
		profile = "standard"
	case "timeout":
		timeout = "10"
	case "cancellation":
		// Cancellation retains the safety timeout while disabling result reuse.
	default:
		return nil, fmt.Errorf("test/snowflaketester: invalid campaign session profile")
	}
	connected, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	if connected == nil {
		return nil, fmt.Errorf("test/snowflaketester: missing session connection")
	}
	defer func() {
		if err != nil {
			connected.Close()
			connection = nil
		}
	}()
	queryer, ok := connected.(driver.QueryerContext)
	if !ok {
		return nil, fmt.Errorf("test/snowflaketester: session connection does not support QueryContext")
	}
	rows, err := queryer.QueryContext(ctx, "SHOW PARAMETERS IN SESSION", nil)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return nil, fmt.Errorf("test/snowflaketester: missing session parameter rows")
	}
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil {
			if err == nil {
				err = closeErr
			}
		}
	}()

	columns := rows.Columns()
	if len(columns) < 2 || len(columns) > 16 {
		return nil, fmt.Errorf("test/snowflaketester: invalid session parameter columns")
	}
	keyColumn, valueColumn := -1, -1
	for i, name := range columns {
		if strings.EqualFold(name, "key") {
			if keyColumn >= 0 {
				return nil, fmt.Errorf("test/snowflaketester: duplicate session parameter key column")
			}
			keyColumn = i
		}
		if strings.EqualFold(name, "value") {
			if valueColumn >= 0 {
				return nil, fmt.Errorf("test/snowflaketester: duplicate session parameter value column")
			}
			valueColumn = i
		}
	}
	if keyColumn < 0 || valueColumn < 0 {
		return nil, fmt.Errorf("test/snowflaketester: missing session parameter columns")
	}

	expected := map[string]string{
		"TIMEZONE":                     c.Timezone,
		"TIMESTAMP_TYPE_MAPPING":       "TIMESTAMP_NTZ",
		"TIMESTAMP_INPUT_FORMAT":       "AUTO",
		"STATEMENT_TIMEOUT_IN_SECONDS": timeout,
	}
	if profile != "standard" {
		expected["USE_CACHED_RESULT"] = "false"
	}
	seen := map[string]bool{}
	values := make([]driver.Value, len(columns))
	for count := 0; ; count++ {
		err = rows.Next(values)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if count >= 1024 {
			return nil, fmt.Errorf("test/snowflaketester: too many session parameter rows")
		}
		key, ok := values[keyColumn].(string)
		if !ok {
			return nil, fmt.Errorf("test/snowflaketester: invalid session parameter key")
		}
		want, relevant := expected[key]
		if !relevant {
			continue
		}
		value, ok := values[valueColumn].(string)
		if !ok || value != want || seen[key] {
			return nil, fmt.Errorf("test/snowflaketester: unexpected session parameter %s", key)
		}
		seen[key] = true
	}
	if len(seen) != len(expected) {
		return nil, fmt.Errorf("test/snowflaketester: missing required session parameters")
	}
	attributes := []any{"profile", profile, "timezone", c.Timezone,
		"timestamp_type_mapping", "TIMESTAMP_NTZ", "timestamp_input_format", "AUTO",
		"statement_timeout_in_seconds", timeout}
	if profile != "standard" {
		attributes = append(attributes, "use_cached_result", false)
	}
	slog.Info("Snowflake test session parameters", attributes...)

	return connected, nil
}

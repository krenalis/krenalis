// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestSetWarehouse checks copied settings without changing the global default.
func TestSetWarehouse(t *testing.T) {
	settings := json.RawMessage(`{"schema":"fixture"}`)
	platform := testsSettings.WarehousePlatform
	k := &Krenalis{}
	k.SetWarehouse("Snowflake", settings)
	clear(settings)
	if k.warehousePlatform != "Snowflake" || string(k.warehouseSettings) != `{"schema":"fixture"}` ||
		testsSettings.WarehousePlatform != platform {
		t.Fatalf("expected isolated copied warehouse settings, got %s %s", k.warehousePlatform, k.warehouseSettings)
	}
}

// TestSetWarehouseInvalid rejects non-object settings before server startup.
func TestSetWarehouseInvalid(t *testing.T) {
	for _, test := range []struct{ platform, settings string }{
		{"", `{}`}, {"Snowflake", ``}, {"Snowflake", `null`}, {"Snowflake", `[]`},
		{"Snowflake", `"text"`}, {"Snowflake", `{`},
	} {
		t.Run(test.platform+"/"+test.settings, func(t *testing.T) {
			defer func() {
				value := recover()
				message, ok := value.(string)
				if !ok || !strings.HasPrefix(message, "test/krenalistester:") {
					t.Fatalf("expected invalid warehouse panic, got %v", value)
				}
			}()
			k := &Krenalis{}
			k.SetWarehouse(test.platform, json.RawMessage(test.settings))
		})
	}
}

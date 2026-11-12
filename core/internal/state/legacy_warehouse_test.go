// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"testing"
)

// TestDecodeLegacyCreateWorkspace preserves fields beside retired credentials.
func TestDecodeLegacyCreateWorkspace(t *testing.T) {
	payload := `{"ID":"workspace111","Organization":"organization","Name":"legacy",` +
		`"Warehouse":{"Platform":"PostgreSQL","Settings":"Aw==",` +
		`"SettingsKey":"BA==","MCPSettingsKey":"AQI="}}`
	var event CreateWorkspace
	decodeNotification(notification{Name: "CreateWorkspace", Version: 1, Payload: payload}, &event)
	if event.ID != "workspace111" || event.Organization != "organization" || event.Name != "legacy" ||
		event.Warehouse.Platform != "PostgreSQL" || !bytes.Equal(event.Warehouse.Settings, []byte{3}) ||
		!bytes.Equal(event.Warehouse.SettingsKey, []byte{4}) {
		t.Fatalf("expected retained legacy workspace fields, got %+v", event)
	}
}

// TestDecodeLegacyUpdateWarehouse accepts notifications with retired settings.
func TestDecodeLegacyUpdateWarehouse(t *testing.T) {
	payload := `{"Workspace":"workspace111","Settings":"Aw==","MCPSettings":"AQI=",` +
		`"CancelIncompatibleOperations":true}`
	var event UpdateWarehouse
	decodeNotification(notification{Name: "UpdateWarehouse", Version: 1, Payload: payload}, &event)
	if event.Workspace != "workspace111" || !bytes.Equal(event.Settings, []byte{3}) ||
		!event.CancelIncompatibleOperations {
		t.Fatalf("expected retained legacy warehouse fields, got %+v", event)
	}
}

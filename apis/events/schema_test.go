//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package events

import (
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_SchemaWithoutGID(t *testing.T) {

	props := types.Properties(Schema)
	if len(props) != 15 {
		t.Fatalf("expecting 15 properties, got %d", len(props))
	}

}

func Test_SchemaWithUserGID(t *testing.T) {

	props := types.Properties(SchemaWithUserGID)
	if len(props) != 16 {
		t.Fatalf("expecting 16 properties, got %d", len(props))
	}

	gid := props[0]
	if gid.Name != "user" {
		t.Fatalf("name of first property should be \"user\", got %q", gid.Name)
	}
	if !gid.Type.EqualTo(types.UUID()) {
		t.Fatalf("type of first property should be %s, got %s", types.UUID(), gid.Type)
	}

}

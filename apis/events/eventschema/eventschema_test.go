//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package eventschema

import (
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_SchemaWithoutGID(t *testing.T) {

	props := SchemaWithoutGID.Properties()
	if len(props) != 15 {
		t.Fatalf("expecting 15 properties, got %d", len(props))
	}

}

func Test_SchemaWithGID(t *testing.T) {

	props := SchemaWithGID.Properties()
	if len(props) != 16 {
		t.Fatalf("expecting 16 properties, got %d", len(props))
	}

	gid := props[0]
	if gid.Name != "gid" {
		t.Fatalf("name of first property should be \"gid\", got %q", gid.Name)
	}
	if !gid.Type.EqualTo(types.Int(32)) {
		t.Fatalf("type of first property should be %s, got %s", types.Int(32), gid.Type)
	}

}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestVersion tests that the version set at build time with the "-X" linker
// flag is the one reported by Krenalis. It fails when Krenalis is not launched
// externally, as it is then not built with that flag.
func TestVersion(t *testing.T) {

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	got := k.Version()
	if got != krenalistester.KrenalisVersion {
		t.Fatalf("expected version %q, got %q", krenalistester.KrenalisVersion, got)
	}

}

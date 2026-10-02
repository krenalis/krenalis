// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"runtime/debug"
	"sync"
)

// version is the Krenalis version set at build time with:
//
//	-ldflags "-X github.com/krenalis/krenalis/cmd.version=v1.2.3"
//
// When empty, the version is read from the build information.
var version string

// krenalisVersion returns the Krenalis version: the one set at build time, if
// any, otherwise the main module version recorded by the Go toolchain, or
// "(devel)" when it is not available.
var krenalisVersion = sync.OnceValue(func() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
})

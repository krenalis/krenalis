// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"runtime/debug"
	"sync"
)

// buildVersion is the Krenalis version set at build time with:
//
//	-ldflags "-X github.com/krenalis/krenalis/cmd.buildVersion=v1.2.3"
//
// When empty, the version is read from the build information.
var buildVersion string

// krenalisVersion returns the Krenalis version: the one set at build time, if
// any, otherwise the version of the Krenalis module recorded by the Go
// toolchain, whether it is the main module or a dependency, or "(devel)" when
// it is not available or the module is replaced.
var krenalisVersion = sync.OnceValue(func() string {
	if buildVersion != "" {
		return buildVersion
	}
	const modulePath = "github.com/krenalis/krenalis"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	if info.Main.Path == modulePath && info.Main.Version != "" {
		return info.Main.Version
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath && dep.Replace == nil && dep.Version != "" {
			return dep.Version
		}
	}
	return "(devel)"
})

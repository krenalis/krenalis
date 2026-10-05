// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"runtime/debug"
	"sync"
)

// buildVersion is the Krenalis version optionally set at build time with:
//
//	-ldflags "-X github.com/krenalis/krenalis/cmd.buildVersion=v1.2.3"
//
// When empty, krenalisVersion reads the version from the build information.
var buildVersion string

// krenalisVersion returns the Krenalis version, determined as follows:
//
//  1. the version set at build time, if any;
//  2. otherwise, the version recorded by the Go toolchain for the main module,
//     if it is the Krenalis module;
//  3. otherwise, the version recorded by the Go toolchain for the Krenalis
//     module as a dependency, if it is not replaced;
//  4. otherwise, "(devel)".
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

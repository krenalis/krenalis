// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

// UsesInternalServer reports whether the server runs in the test process.
func UsesInternalServer() bool {
	return !launchKrenalisExternally
}

// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package embed

import _ "embed"

//go:embed normalize.js
var JavaScriptNormalizeFunc string

//go:embed normalize.py
var PythonNormalizeFunc string

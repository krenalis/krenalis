//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package embed

import _ "embed"

//go:embed normalize.js
var JavaScriptNormalizeFunc string

//go:embed normalize.py
var PythonNormalizeFunc string

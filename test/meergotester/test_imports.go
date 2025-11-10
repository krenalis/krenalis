// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package meergotester

// This file has two purposes:
//
// (1) to import connectors and warehouses into the Meergo executable of the
// tests, when the tests are run by compiling Meergo directly inside the test
// executable. In this first case, this file acts as any other source file in
// Go.
//
// (2) to define the connectors and warehouses needed by the tests, so that this
// file is then copied to the temporary directory where the Meergo executable
// used in the tests will be compiled, in those cases where the tests are run by
// running Meergo in a separate process (which is the default case). For this
// reason, it is IMPORTANT that this file is not moved or renamed without
// changing the test execution procedure.

import (
	_ "github.com/meergo/meergo/connectors/csv"
	_ "github.com/meergo/meergo/connectors/dummy"
	_ "github.com/meergo/meergo/connectors/filesystem"
	_ "github.com/meergo/meergo/connectors/json"
	_ "github.com/meergo/meergo/connectors/kafka"
	_ "github.com/meergo/meergo/connectors/parquet"
	_ "github.com/meergo/meergo/connectors/sdk"
	_ "github.com/meergo/meergo/connectors/webhook"

	_ "github.com/meergo/meergo/warehouses/postgresql"
)

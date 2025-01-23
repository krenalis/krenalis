//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"os"
	"sync"
)

type TestsSettings struct {
	MeergoHost       string
	Database         *DBSettings
	PythonExecutable string
	WarehouseType    string
	Warehouse        *DBSettings
}

type DBSettings struct {
	Host     string
	Port     int
	Username string
	Password string
	Database string
	Schema   string
}

// PostgresWarehouseSettings returns the settings of the PostgreSQL warehouse.
// Should be called only after that the test has been initialized.
func PostgresWarehouseSettings() []byte {
	return JSONEncodeSettings(map[string]any{
		"Host":     testsSettings.Warehouse.Host,
		"Port":     testsSettings.Warehouse.Port,
		"Username": testsSettings.Warehouse.Username,
		"Password": testsSettings.Warehouse.Password,
		"Database": testsSettings.Warehouse.Database,
		"Schema":   testsSettings.Warehouse.Schema,
	})
}

var testsSettings *TestsSettings
var testsSettingsMu sync.Mutex

func init() {
	testsSettings = &TestsSettings{
		MeergoHost: "127.0.0.1:9091",
		Database: &DBSettings{
			// Host and Port will be set when PostgreSQL container starts.
			Database: "test_postgres",
			Username: "test_postgres",
			Password: "test_postgres",
			Schema:   "public",
		},
		PythonExecutable: "python3",
		WarehouseType:    "PostgreSQL",
		Warehouse: &DBSettings{
			// Host and Port will be set when warehouse container starts.
			Database: "test_warehouse",
			Username: "test_warehouse",
			Password: "test_warehouse",
			Schema:   "public",
		},
	}
	if host := os.Getenv("MEERGO_TESTS_HOST"); host != "" {
		testsSettings.MeergoHost = host
	}
	if pythonPath := os.Getenv("MEERGO_TESTS_PYTHON_PATH"); pythonPath != "" {
		testsSettings.PythonExecutable = pythonPath
	}
}

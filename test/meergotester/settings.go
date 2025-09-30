//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type TestsSettings struct {
	HTTP             *HTTPSettings
	Database         *DBSettings
	PythonExecutable string
	WarehouseType    string
	Warehouse        *DBSettings
}

type HTTPSettings struct {
	Host string
	Port int
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
		HTTP: &HTTPSettings{
			Host: "127.0.0.1",
			Port: 2023,
		},
		Database: &DBSettings{
			// Host and Port will be set when PostgreSQL container starts.
			Database: "test_postgres",
			Username: "test_postgres",
			Password: "test_postgres",
			Schema:   "public",
		},
		WarehouseType: "PostgreSQL",
		Warehouse: &DBSettings{
			// Host and Port will be set when warehouse container starts.
			Database: "test_warehouse",
			Username: "test_warehouse",
			Password: "test_warehouse",
			Schema:   "public",
		},
	}
	pyExecutable, err := lookupPythonExecPath()
	if err != nil {
		panic(err)
	}
	testsSettings.PythonExecutable = pyExecutable
	if addr := os.Getenv("MEERGO_TESTS_ADDR"); addr != "" {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			panic("MEERGO_TESTS_ADDR must be in the format host:port")
		}
		testsSettings.HTTP.Host = host
		testsSettings.HTTP.Port, err = strconv.Atoi(port)
		if err != nil {
			panic("MEERGO_TESTS_ADDR must be in the format host:port")
		}
	}
	if pythonPath := os.Getenv("MEERGO_TESTS_PYTHON_PATH"); pythonPath != "" {
		testsSettings.PythonExecutable = pythonPath
	}
}

// lookupPythonExecPath returns the path of the Python executable available on
// this system, or an error if it cannot be found.
func lookupPythonExecPath() (string, error) {
	// TODO: Keep in sync with other copies of this function, scattered
	// throughout the code, that have the same name.
	pythonNames := []string{"python", "python3", "python3.13"}
	for _, name := range pythonNames {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("the Python executable cannot be found "+
		"with any of these names: %s", strings.Join(pythonNames, ", "))
}

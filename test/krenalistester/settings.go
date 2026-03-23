// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/krenalis/krenalis/tools/json"
)

type TestsSettings struct {
	HTTP              *HTTPSettings
	Database          *DBSettings
	NATS              *NATSSettings
	PythonExecutable  string
	WarehousePlatform string
	Warehouse         *DBSettings
}

type HTTPSettings struct {
	Host string
	Port int
}

type NATSSettings struct {
	URL      string
	Port     int
	User     string
	Password string
}

type DBSettings struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
}

// PostgresWarehouseSettings returns the settings of the PostgreSQL warehouse.
// Should be called only after that the test has been initialized.
func PostgresWarehouseSettings() json.Value {
	return JSONEncodeSettings(map[string]any{
		"host":     testsSettings.Warehouse.Host,
		"port":     testsSettings.Warehouse.Port,
		"username": testsSettings.Warehouse.Username,
		"password": testsSettings.Warehouse.Password,
		"database": testsSettings.Warehouse.Database,
		"schema":   testsSettings.Warehouse.Schema,
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
		WarehousePlatform: "PostgreSQL",
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
	if addr := os.Getenv("KRENALIS_TESTS_ADDR"); addr != "" {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			panic("KRENALIS_TESTS_ADDR must be in the format host:port")
		}
		testsSettings.HTTP.Host = host
		testsSettings.HTTP.Port, err = strconv.Atoi(port)
		if err != nil {
			panic("KRENALIS_TESTS_ADDR must be in the format host:port")
		}
	}
	if pythonPath := os.Getenv("KRENALIS_TESTS_PYTHON_PATH"); pythonPath != "" {
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

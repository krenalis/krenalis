//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergotester

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

var testsSettings *TestsSettings

// loadTestConfig loads the tests configuration from the file
// "meergo-tests.json", if not already loaded.
func loadTestConfig() error {

	if testsSettings != nil {
		return nil
	}

	absFilename, _ := filepath.Abs("meergo-tests.json")
	f, err := os.Open("meergo-tests.json")
	if err != nil {
		return fmt.Errorf("cannot load JSON file %q: %s", absFilename, err)
	}
	defer f.Close()
	var setts *TestsSettings
	err = json.NewDecoder(f).Decode(&setts)
	if err != nil {
		return fmt.Errorf("cannot decode JSON from %q: %s", absFilename, err)
	}

	if setts.MeergoHost == "" {
		return errors.New("missing value for 'MeergoHost'")
	}
	if setts.PythonExecutable == "" {
		return errors.New("missing value for 'PythonExecutable'")
	}
	if setts.WarehouseType == "" {
		return errors.New("missing value for 'WarehouseType'")
	}
	if setts.WarehouseType != "PostgreSQL" {
		return errors.New("currently only the PostgreSQL warehouse is supported by the tests")
	}

	setts.MeergoHost = strings.TrimRight(setts.MeergoHost, "/")

	testsSettings = setts

	return nil
}

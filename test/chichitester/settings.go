//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package chichitester

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TestsSettings struct {
	ChichiHost       string
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
// "chichi-tests.json", if not already loaded.
func loadTestConfig() error {

	if testsSettings != nil {
		return nil
	}

	absFilename, _ := filepath.Abs("chichi-tests.json")
	f, err := os.Open("chichi-tests.json")
	if err != nil {
		return fmt.Errorf("cannot load JSON file %q: %s", absFilename, err)
	}
	defer f.Close()
	var setts *TestsSettings
	err = json.NewDecoder(f).Decode(&setts)
	if err != nil {
		return fmt.Errorf("cannot decode JSON from %q: %s", absFilename, err)
	}

	if setts.ChichiHost == "" {
		return errors.New("missing value for 'ChichiHost'")
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

	setts.ChichiHost = strings.TrimRight(setts.ChichiHost, "/")

	testsSettings = setts

	return nil
}

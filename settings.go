//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import "gopkg.in/gcfg.v1"

type Settings struct {
	DB struct {
		Address  string
		Username string
		Password string
		Database string
	}
}

// parseINIFile parses the 'app.ini' file, returning its content as a *Settings
// value.
func parseINIFile() (*Settings, error) {
	// Parse the configuration file.
	var settings Settings
	err := gcfg.ReadFileInto(&settings, "app.ini")
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

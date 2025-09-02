//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/joho/godotenv"
)

var (
	envVars    *EnvVars
	envVarsMu  sync.Mutex
	envVarsErr error
)

func init() {
	// This is only useful if no one has read the env file at startup, so as to
	// force the reading of the '.env' file immediately upon Meergo
	// initialization. Otherwise, theoretically, it could happen later, perhaps
	// after 1 hour of Meergo running, and this would conflict with the fact
	// that the '.env' file must be read at startup.
	_, _ = GetEnvVars()
}

// GetEnvVars returns an EnvVars instance that can be used to retrieve the
// environment variables passed to Meergo. In case of error, this method returns
// nil and the error.
func GetEnvVars() (*EnvVars, error) {

	envVarsMu.Lock()
	defer envVarsMu.Unlock()
	if envVars != nil {
		return envVars, nil
	}
	if envVarsErr != nil {
		return nil, envVarsErr
	}

	// Read environment variables from the '.env' file, if exists. It is
	// important to call Overload instead of Load because we want any
	// environment variables already set to be overwritten.
	err := godotenv.Overload()
	if err != nil && !os.IsNotExist(err) {
		p, err2 := filepath.Abs(".env")
		if err2 != nil {
			p = ".env"
		}
		envVarsErr = fmt.Errorf("failed to read %q file: %s", p, err)
		return nil, envVarsErr
	}

	envVars = &EnvVars{}

	return envVars, nil
}

// EnvVars provides the environment variables passed to Meergo.
type EnvVars struct{}

// Get returns the value of the environment variable key. If the variable is not
// present, it returns the empty string.
func (env *EnvVars) Get(key string) string {
	return os.Getenv(key)
}

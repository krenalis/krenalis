// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/krenalis/krenalis/tools/dotenv"
)

var (
	envVars    *EnvVars
	envVarsMu  sync.Mutex
	envVarsErr error
)

func init() {
	// This is only useful if no one has read the env file at startup, so as to
	// force the reading of the '.env' file immediately upon Krenalis
	// initialization. Otherwise, theoretically, it could happen later, perhaps
	// after 1 hour of Krenalis running, and this would conflict with the fact
	// that the '.env' file must be read at startup.
	_, _ = GetEnvVars()
}

// GetEnvVars returns an EnvVars instance that can be used to retrieve the
// environment variables passed to Krenalis. In case of error, this method
// returns nil and the error.
func GetEnvVars() (*EnvVars, error) {

	envVarsMu.Lock()
	defer envVarsMu.Unlock()
	if envVars != nil {
		return envVars, nil
	}
	if envVarsErr != nil {
		return nil, envVarsErr
	}

	// Load environment variables from the '.env' file, if exists.
	err := dotenv.Load(".env")
	if err != nil && !os.IsNotExist(err) {
		p, err2 := filepath.Abs(".env")
		if err2 != nil {
			p = ".env"
		}
		envVarsErr = fmt.Errorf("failed to read %q file: %s", p, err)
		return nil, envVarsErr
	}

	// Ensure that all the environment variables whose name starts with
	// "KRENALIS_" have values which contain only valid UTF-8 characters.
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "KRENALIS_") {
			key, value, _ := strings.Cut(v, "=")
			if !utf8.ValidString(value) {
				return nil, fmt.Errorf("the environment variable %q contains a value which is not UTF-8 valid", key)
			}
		}
	}

	envVars = &EnvVars{}

	return envVars, nil
}

// EnvVars provides the environment variables passed to Krenalis.
type EnvVars struct{}

// Get returns the value of the Krenalis environment variable with the given
// key. If the variable is not present, this method returns the empty string. It
// is guaranteed that the returned value contains only UTF-8 valid characters.
// If key does not start with "KRENALIS_", this method panics.
func (env *EnvVars) Get(key string) string {
	if !strings.HasPrefix(key, "KRENALIS_") {
		panic("EnvVars.Get: key must start with KRENALIS_")
	}
	return os.Getenv(key)
}

// Lookup returns the value of the Krenalis environment variable with the given
// key. If the variable is present, it returns its value and true; otherwise, it
// returns an empty string and false. It is guaranteed that the returned value
// contains only valid UTF-8 characters.
// If key does not start with "KRENALIS_", this method panics.
func (env *EnvVars) Lookup(key string) (string, bool) {
	if !strings.HasPrefix(key, "KRENALIS_") {
		panic("EnvVars.Lookup: key must start with KRENALIS_")
	}
	return os.LookupEnv(key)
}

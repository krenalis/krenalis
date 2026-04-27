// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/krenalis/krenalis/cmd/internal/config"
	"github.com/krenalis/krenalis/tools/dotenv"
)

type Store struct{}

func New() *Store {
	return &Store{}
}

func (s *Store) Load(_ context.Context) (config.Config, error) {
	err := dotenv.Load(".env")
	if err != nil && !os.IsNotExist(err) {
		p, err2 := filepath.Abs(".env")
		if err2 != nil {
			p = ".env"
		}
		return nil, fmt.Errorf("config/env: failed to read %q file: %s", p, err)
	}
	return &Config{}, nil
}

type Config struct{}

func (c *Config) Get(name string) string {
	return os.Getenv(name)
}

func (c *Config) Lookup(name string) (string, bool) {
	return os.LookupEnv(name)
}

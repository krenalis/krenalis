// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package config

import (
	"context"
)

type Store interface {
	Load(ctx context.Context) (Config, error)
}

type Config interface {
	Get(key string) string
	Lookup(key string) (string, bool)
}

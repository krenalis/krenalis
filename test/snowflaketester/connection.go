// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/snowflakedb/gosnowflake/v2"
)

// OpenDatabase opens a lazy pool for the configured test database, without
// creating a schema. The caller owns the pool and must close it.
func OpenDatabase(ctx context.Context) (*sql.DB, error) {
	if ctx == nil {
		return nil, fmt.Errorf("test/snowflaketester: nil database context")
	}
	db, _, err := openDatabase(ctx)
	return db, err
}

func openDatabase(ctx context.Context) (*sql.DB, Settings, error) {

	// Return a clear error if env vars are not passed.
	found := false
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, "KRENALIS_SNOWFLAKE_TESTER_") {
			found = true
			break
		}
	}
	if !found {
		return nil, Settings{}, fmt.Errorf("test/snowflaketester: missing Snowflake credential environment")
	}

	// Read the Snowflake settings from the environment.
	settings := Settings{
		Account:   os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ACCOUNT"),
		Database:  os.Getenv("KRENALIS_SNOWFLAKE_TESTER_DATABASE"),
		Password:  os.Getenv("KRENALIS_SNOWFLAKE_TESTER_PASSWORD"),
		Role:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_ROLE"),
		Schema:    "", // will be set later.
		User:      os.Getenv("KRENALIS_SNOWFLAKE_TESTER_USER"),
		Warehouse: os.Getenv("KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE"),
	}

	// When running inside GitHub Actions, mint a fresh OIDC token on every
	// invocation. A single pre-fetched token would expire (~15 minute lifetime)
	// before the long 'go run ./test/commit' run completes. The
	// ACTIONS_ID_TOKEN_REQUEST_* env vars are exposed for the whole job when
	// 'permissions: id-token: write' is set, and can be used to mint a new
	// token on demand.
	reqURL, reqToken := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if reqURL != "" && reqToken != "" {
		token, err := fetchGitHubOIDCToken(ctx, reqURL, reqToken, "snowflakecomputing.com")
		if err != nil {
			return nil, Settings{}, fmt.Errorf("cannot fetch GitHub OIDC token: %s", err)
		}
		settings.OIDCToken = token
		settings.Password = ""
	}

	// Instantiate a Snowflake connector.
	cfg := gosnowflake.Config{
		Account:   settings.Account,
		Database:  settings.Database,
		Role:      settings.Role,
		User:      settings.User,
		Warehouse: settings.Warehouse,
		Params: map[string]*string{
			"CLIENT_TELEMETRY_ENABLED": falseStrPtr,
		},
	}
	if settings.OIDCToken != "" {
		cfg.Authenticator = gosnowflake.AuthTypeWorkloadIdentityFederation
		cfg.WorkloadIdentityProvider = "OIDC"
		cfg.Token = settings.OIDCToken
	} else {
		cfg.Password = settings.Password
	}
	connector := gosnowflake.NewConnector(gosnowflake.SnowflakeDriver{}, cfg)

	return sql.OpenDB(connector), settings, nil
}

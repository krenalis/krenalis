// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestCampaignInventory checks only the schemas named in campaign manifests.
// It never creates or drops objects. Set KRENALIS_SNOWFLAKE_CAMPAIGN_DIR to the
// directory containing the manifests; absent configuration skips this test.
func TestCampaignInventory(t *testing.T) {

	directory := os.Getenv("KRENALIS_SNOWFLAKE_CAMPAIGN_DIR")
	if testing.Short() || directory == "" || os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("requires an authorized campaign inventory")
	}
	names, err := campaignSchemaNames(directory)
	if err != nil {
		t.Fatalf("expected bounded campaign schema names, got %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	db, err := OpenDatabase(ctx)
	if err != nil {
		t.Fatalf("expected inventory connection, got %v", err)
	}
	defer db.Close()
	var database string
	row := db.QueryRowContext(ctx, "SELECT CURRENT_DATABASE()")
	err = row.Scan(&database)
	if err != nil {
		t.Fatalf("expected current database, got %v", err)
	}
	if database == "" || len(database) > 255 || strings.ContainsRune(database, 0) {
		t.Fatal("expected bounded database name, got invalid name")
	}
	query := `SHOW SCHEMAS IN DATABASE "` + strings.ReplaceAll(database, `"`, `""`) +
		`" STARTS WITH 'KRENALIS_TEST_SCHEMA_' LIMIT 1000`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("expected read-only schema inventory, got %v", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatalf("expected schema metadata, got %v", err)
	}
	if len(columns) < 2 || len(columns) > 32 {
		t.Fatal("expected bounded schema metadata, got invalid columns")
	}
	index := -1
	for i, name := range columns {
		if strings.EqualFold(name, "name") {
			if index != -1 {
				t.Fatal("expected one schema name column, got duplicates")
			}
			index = i
		}
	}
	if index == -1 {
		t.Fatal("expected schema name column, got none")
	}
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	visible, remaining := 0, 0
	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			t.Fatalf("expected inventory row, got %v", err)
		}
		visible++
		name, ok := values[index].(string)
		if !ok || len(name) > 255 || visible >= 1000 {
			t.Fatal("expected complete bounded inventory, got invalid or truncated result")
		}
		if names[strings.ToUpper(name)] {
			remaining++
		}
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		t.Fatalf("expected complete inventory, got %v", err)
	}
	if closeErr != nil {
		t.Fatalf("expected inventory cleanup, got %v", closeErr)
	}
	t.Logf("campaign inventory campaign_schemas=%d visible_test_schemas=%d remaining_campaign_schemas=%d",
		len(names), visible, remaining)
	if remaining != 0 {
		t.Fatalf("expected no campaign schemas remaining, got %d", remaining)
	}

}

// TestCampaignSchemaNames checks the manifest input before any remote access.
func TestCampaignSchemaNames(t *testing.T) {

	name := "KRENALIS_TEST_SCHEMA_123_" + strings.Repeat("ab", 16)
	for _, test := range []struct {
		name, contents string
		valid          bool
	}{
		{"valid", `{"schemas":{"created":{"` + name + `":1},"dropped":{"` + name + `":1}}}`, true},
		{"unbalanced", `{"schemas":{"created":{"` + name + `":1},"dropped":{}}}`, true},
		{"invalid name", `{"schemas":{"created":{"other":1}}}`, false},
		{"invalid count", `{"schemas":{"created":{"` + name + `":0}}}`, false},
		{"empty", `{"schemas":{"created":{},"dropped":{}}}`, false},
		{"invalid JSON", `{`, false},
		{"too large", strings.Repeat(" ", 4*1024*1024+1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			err := os.WriteFile(filepath.Join(directory, "run.manifest.json"), []byte(test.contents), 0o600)
			if err != nil {
				t.Fatalf("expected manifest fixture, got %v", err)
			}
			names, err := campaignSchemaNames(directory)
			if err != nil {
				if test.valid {
					t.Fatalf("expected campaign names, got %v", err)
				}
				return
			}
			if !test.valid || len(names) != 1 || !names[strings.ToUpper(name)] {
				t.Fatalf("expected valid=%t with one name, got %v", test.valid, names)
			}
		})
	}
	_, err := campaignSchemaNames(t.TempDir())
	if err != nil {
		return
	}
	t.Fatal("expected empty directory error, got success")

}

// campaignSchemaNames reads only schema ledgers, including incomplete runs so
// that an inventory can detect leaked schemas after a failed campaign.
func campaignSchemaNames(directory string) (map[string]bool, error) {

	files, err := filepath.Glob(filepath.Join(directory, "*.manifest.json"))
	if err != nil {
		return nil, err
	}
	if len(files) == 0 || len(files) > 100 {
		return nil, fmt.Errorf("test/snowflaketester: invalid manifest count")
	}
	pattern := regexp.MustCompile(`^KRENALIS_TEST_SCHEMA_[0-9]{1,20}_[0-9A-F]{32}$`)
	names := map[string]bool{}
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(file, 4*1024*1024+1))
		closeErr := file.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(data) > 4*1024*1024 {
			return nil, fmt.Errorf("test/snowflaketester: manifest exceeds size limit")
		}
		var manifest struct {
			Schemas struct {
				Created, Dropped map[string]int
			}
		}
		err = json.Unmarshal(data, &manifest)
		if err != nil {
			return nil, fmt.Errorf("test/snowflaketester: invalid manifest JSON")
		}
		for _, ledger := range []map[string]int{manifest.Schemas.Created, manifest.Schemas.Dropped} {
			for name, count := range ledger {
				name = strings.ToUpper(name)
				if !pattern.MatchString(name) || count < 1 || count > 1000 {
					return nil, fmt.Errorf("test/snowflaketester: invalid schema ledger entry")
				}
				names[name] = true
				if len(names) > 1000 {
					return nil, fmt.Errorf("test/snowflaketester: too many campaign schemas")
				}
			}
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("test/snowflaketester: no campaign schemas")
	}

	return names, nil
}

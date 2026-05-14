// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

func TestParquetImport(t *testing.T) {

	// Retrieve the storage directory that contains the Parquet file to import.
	storageDir, err := filepath.Abs("./testdata/parquet_import_test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(storageDir); err != nil {
		t.Fatal(err)
	}

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := krenalistester.NewKrenalisInstance(t)
	c.PopulateProfileSchema(false)
	c.SetFileSystemRoot(storageDir)
	c.Start()
	defer c.Stop()

	// Change the profile schema, leaving only the properties used by this test.
	profileSchemaProperties := []types.Property{}
	profileSchemaProperties = append(profileSchemaProperties, types.Property{
		Name:         "parquet_id",
		Type:         types.Int(64),
		ReadOptional: true,
	})
	profileSchemaProperties = append(profileSchemaProperties, types.Property{
		Name:         "parquet_imported",
		Type:         types.JSON(),
		ReadOptional: true,
	})
	c.AlterProfileSchema(types.Object(profileSchemaProperties), nil, nil)

	fs := c.CreateSourceFileSystem()

	testCases := []struct {
		name        string
		path        string
		compression krenalistester.Compression
	}{
		{name: "NoCompression", path: "test.parquet"},
		{name: "Zip", path: "test.parquet.zip", compression: krenalistester.ZipCompression},
		{name: "Gzip", path: "test.parquet.gz", compression: krenalistester.GzipCompression},
		{name: "Snappy", path: "test.parquet.sz", compression: krenalistester.SnappyCompression},
	}

	for _, tc := range testCases {

		f := func(t *testing.T) {

			pipeline := c.CreatePipeline(fs, "User", parquetImportPipeline(tc.path, tc.compression))
			var identityResolutionStart time.Time
			var runStarted bool
			defer purgeParquetImportProfiles(t, c)
			defer c.DeletePipeline(pipeline)
			defer func() {
				if runStarted {
					waitParquetImportIdentityResolution(t, c, identityResolutionStart)
				}
			}()

			identityResolutionStart = time.Now().UTC()
			run := c.RunPipeline(pipeline)
			runStarted = true
			c.WaitRunsCompletion(fs, run)

			waitParquetImportProfiles(t, c, len(expectedProfiles))
			profiles, _, _ := c.Profiles([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
			checkParquetImportProfiles(t, profiles)

		}
		if !t.Run(tc.name, f) {
			return
		}

	}
}

// parquetImportPipeline returns a pipeline definition for importing a Parquet
// file with the given compression.
func parquetImportPipeline(path string, compression krenalistester.Compression) krenalistester.PipelineToSet {
	return krenalistester.PipelineToSet{
		Name:        "Parquet",
		Enabled:     true,
		Path:        path,
		Compression: compression,
		InSchema: types.Object([]types.Property{
			{Name: "parquet_id", Type: types.Int(64), Nullable: true},
			{Name: "first_name", Type: types.String(), Nullable: true},
			{Name: "last_name", Type: types.String(), Nullable: true},
			{Name: "date_of_birth", Type: types.Date(), Nullable: true},
			{Name: "updated_at", Type: types.DateTime(), Nullable: true},
			{Name: "lunch_time", Type: types.Time(), Nullable: true},
			{Name: "score", Type: types.Decimal(9, 5), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "parquet_id", Type: types.Int(64), ReadOptional: true},
			{Name: "parquet_imported", Type: types.JSON(), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Function: &krenalistester.TransformationFunction{
				Language: "Python",
				Source: strings.Join([]string{
					`def transform(user: dict) -> dict:`,
					`    return {`,
					`        "parquet_id": user["parquet_id"],`,
					`        "parquet_imported": {k: v for k, v in user.items() if v is not None}`,
					`    }`,
				}, "\n"),
				InPaths:  []string{"parquet_id", "first_name", "last_name", "date_of_birth", "updated_at", "lunch_time", "score"},
				OutPaths: []string{"parquet_id", "parquet_imported"},
			},
		},
		UserIDColumn: "parquet_id",
		Format:       "parquet",
	}
}

// checkParquetImportProfiles verifies that profiles match the expected Parquet
// import result.
func checkParquetImportProfiles(t *testing.T, profiles []krenalistester.Profile) {
	t.Helper()
	if len(profiles) != len(expectedProfiles) {
		t.Fatalf("expected %d profile(s), got %d", len(expectedProfiles), len(profiles))
	}
	var fail bool
	for i := range profiles {
		gotAttributes, ok := profiles[i].Attributes["parquet_imported"].(map[string]any)
		if !ok {
			t.Errorf("profiles[%d]: expected parquet_imported to be map[string]any, got %T", i, profiles[i].Attributes["parquet_imported"])
			fail = true
			continue
		}
		expectedAttributes := expectedProfiles[i]
		if !reflect.DeepEqual(gotAttributes, expectedAttributes) {
			t.Errorf("profiles[%d]: expected attributes %#v, got %#v", i, expectedAttributes, gotAttributes)
			fail = true
		}
	}
	if fail {
		t.Fatal("profiles do not match")
	}
}

// waitParquetImportProfiles waits until the expected number of Parquet import
// profiles is visible.
func waitParquetImportProfiles(t *testing.T, c *krenalistester.Krenalis, expected int) {
	t.Helper()
	for attempt := range 20 {
		profiles, _, _ := c.Profiles([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
		if len(profiles) == expected {
			return
		}
		t.Logf("[attempt %d] expected %d profile(s), got %d", attempt+1, expected, len(profiles))
		time.Sleep(200 * time.Millisecond)
	}
	profiles, _, _ := c.Profiles([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
	t.Fatalf("expected %d profile(s), got %d", expected, len(profiles))
}

// waitParquetImportIdentityResolution waits for the identity resolution started
// at or after since to complete.
func waitParquetImportIdentityResolution(t *testing.T, c *krenalistester.Krenalis, since time.Time) {
	t.Helper()
	for attempt := range 20 {
		startTime, endTime := c.LatestIdentityResolution()
		if startTime != nil && (startTime.Equal(since) || startTime.After(since)) && endTime != nil {
			return
		}
		t.Logf("[attempt %d] waiting for identity resolution to complete", attempt+1)
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("expected identity resolution to complete")
}

// purgeParquetImportProfiles runs identity resolution until no Parquet import
// profiles are visible.
func purgeParquetImportProfiles(t *testing.T, c *krenalistester.Krenalis) {
	t.Helper()
	for attempt := range 20 {
		c.RunIdentityResolution()
		profiles, _, _ := c.Profiles([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
		if len(profiles) == 0 {
			return
		}
		t.Logf("[attempt %d] expected 0 profile(s), got %d", attempt+1, len(profiles))
		time.Sleep(200 * time.Millisecond)
	}
	profiles, _, _ := c.Profiles([]string{"parquet_imported"}, "parquet_id", false, 0, 1000)
	t.Fatalf("expected 0 profile(s), got %d", len(profiles))
}

var expectedProfiles = []map[string]any{
	{"parquet_id": json.Number("100"), "first_name": "John", "last_name": "Lemon"},
	{"parquet_id": json.Number("101"), "first_name": "Ringo", "last_name": "Planett"},
	{"parquet_id": json.Number("102")},
	{"parquet_id": json.Number("103")},
	{"parquet_id": json.Number("104"), "date_of_birth": "1980-01-02"},
	{"parquet_id": json.Number("105"), "date_of_birth": "1935-01-02"},
	{"parquet_id": json.Number("106"), "updated_at": "2012-01-20 07:20:01"},
	{"parquet_id": json.Number("107"), "lunch_time": "13:30:00"},
	{"parquet_id": json.Number("108"), "score": "-1234.56789"},
}

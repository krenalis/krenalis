// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"
)

// TestPhoneImportIdentityResolution verifies that phone numbers are
// canonicalized during file ingestion and matched correctly by identity
// resolution.
func TestPhoneImportIdentityResolution(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	storage := t.TempDir()
	for name, data := range map[string]string{
		"formatted.csv": "id,raw_phone\nit,+39 02-36618 300\nus,+1 (200) 123-0101\n",
		"canonical.csv": "id,raw_phone\nit,+390236618300\nus,+12001230101\n",
	} {
		err := os.WriteFile(filepath.Join(storage, name), []byte(data), 0600)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.PopulateProfileSchema(false)
	k.SetFileSystemRoot(storage)
	k.Start()
	defer k.Stop()

	phone := types.String().AsPhone()
	schema := types.Object([]types.Property{{Name: "phone", Type: phone, ReadOptional: true}})
	k.AlterProfileSchemaAndWait(schema, nil, nil)
	k.UpdateIdentityResolutionSettings(true, nil)
	connections := map[string]string{}
	for _, source := range []string{"formatted.csv", "canonical.csv"} {
		connection := k.CreateSourceFileSystem()
		connections[source] = connection
		pipeline := k.CreatePipeline(connection, "User", krenalistester.PipelineToSet{
			Name: "Import phones", Enabled: true, Path: source,
			InSchema: types.Object([]types.Property{
				{Name: "id", Type: types.String()},
				{Name: "raw_phone", Type: types.String()},
			}),
			OutSchema: schema,
			Transformation: &krenalistester.Transformation{
				Mapping: map[string]string{"phone": "raw_phone"},
			},
			UserIDColumn: "id", Format: "csv",
			FormatSettings: krenalistester.JSONEncodeSettings(map[string]any{
				"separator": ",", "hasColumnNames": true,
			}),
		})
		run := k.StartPipelineRun(pipeline)
		k.WaitForRunsCompletion(run)
		identities, total := k.ConnectionIdentities(connection, 0, 100)
		if total != 2 || len(identities) != 2 {
			t.Fatalf("expected two persisted identities from %s, got total %d and length %d", source, total, len(identities))
		}
	}

	// Without phone-based identity resolution, the two connections must produce
	// four separate profiles.
	profiles, total := k.Profiles([]string{"phone"}, "", false, 0, 100)
	if total != 4 || len(profiles) != 4 {
		t.Fatalf("expected four profiles before resolution, got total %d and length %d", total, len(profiles))
	}
	counts := map[string]int{}
	for _, profile := range profiles {
		value, ok := profile.Attributes["phone"].(string)
		if !ok {
			t.Fatalf("expected a persisted phone string, got %#v", profile.Attributes["phone"])
		}
		counts[value]++
	}
	wantCounts := map[string]int{"+390236618300": 2, "+12001230101": 2}
	if !reflect.DeepEqual(counts, wantCounts) {
		t.Fatalf("expected canonical phone counts %#v, got %#v", wantCounts, counts)
	}

	// Identity resolution must merge profiles with the same canonical phone value,
	// including a structurally possible but unallocated number.
	k.UpdateIdentityResolutionSettings(true, []string{"phone"})
	k.RunIdentityResolutionAndWait()
	profiles, total = k.Profiles([]string{"phone"}, "", false, 0, 100)
	if total != 2 || len(profiles) != 2 {
		t.Fatalf("expected two resolved profiles, got total %d and length %d", total, len(profiles))
	}
	resolved := map[string]map[string]string{}
	for _, profile := range profiles {
		phoneValue, ok := profile.Attributes["phone"].(string)
		if !ok {
			t.Fatalf("expected a persisted phone string, got %#v", profile.Attributes["phone"])
		}
		identities, total := k.Identities(profile.KPID, 0, 100)
		if total != 2 || len(identities) != 2 {
			t.Fatalf(
				"expected profile %s to have two identities, got total %d and length %d",
				profile.KPID, total, len(identities),
			)
		}
		resolved[phoneValue] = map[string]string{}
		for _, identity := range identities {
			resolved[phoneValue][identity.Connection] = identity.UserID
		}
	}
	wantResolved := map[string]map[string]string{
		"+390236618300": {
			connections["formatted.csv"]: "it",
			connections["canonical.csv"]: "it",
		},
		"+12001230101": {
			connections["formatted.csv"]: "us",
			connections["canonical.csv"]: "us",
		},
	}
	if !reflect.DeepEqual(resolved, wantResolved) {
		t.Fatalf("expected resolved identities %#v, got %#v", wantResolved, resolved)
	}

}

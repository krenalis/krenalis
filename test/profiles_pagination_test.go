// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"bytes"
	"net/url"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestProfilesPagination verifies that profile pages form a stable, complete sequence.
func TestProfilesPagination(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	k.UpdateIdentityResolutionSettings(false, nil)
	dummy := k.CreateDummy("Dummy", krenalistester.Source)
	pipeline := k.CreatePipeline(dummy, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{"email": "email"},
		},
	})
	run := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(run)
	k.RunIdentityResolutionAndWait()

	const pageSize = 200
	firstPage, _, expectedTotal, datasetVersion, hasNext := k.ProfilesVersioned(
		[]string{"email"}, "", true, 0, pageSize, "")
	if expectedTotal <= pageSize {
		t.Fatalf("expected more than %d profiles, got %d", pageSize, expectedTotal)
	}
	if len(firstPage) != pageSize {
		t.Fatalf("expected %d profiles in the first page, got %d", pageSize, len(firstPage))
	}
	if datasetVersion == "" {
		t.Fatal("expected a published profile dataset version")
	}
	if !hasNext {
		t.Fatal("expected the first profile page to have a continuation")
	}
	var summaryResponse map[string]any
	k.Call("GET", "/v1/profiles?"+url.Values{
		"properties":             []string{"email"},
		"first":                  []string{"0"},
		"limit":                  []string{"1"},
		"includeSchema":          []string{"false"},
		"expectedDatasetVersion": []string{datasetVersion},
	}.Encode(), nil, nil, &summaryResponse)
	if _, ok := summaryResponse["schema"]; ok {
		t.Fatal("expected the profile summary response to omit schema metadata")
	}
	if summaryResponse["datasetVersion"] != datasetVersion {
		t.Fatalf("expected summary dataset version %q, got %v", datasetVersion, summaryResponse["datasetVersion"])
	}
	var countResponse struct {
		Total          int    `json:"total"`
		DatasetVersion string `json:"datasetVersion"`
	}
	k.Call("GET", "/v1/profiles/count", nil, nil, &countResponse)
	if countResponse.Total != expectedTotal || countResponse.DatasetVersion != datasetVersion {
		t.Fatalf("unexpected count response: total=%d version=%q", countResponse.Total, countResponse.DatasetVersion)
	}
	email, ok := firstPage[0].Attributes["email"].(string)
	if !ok {
		t.Fatalf("expected a string email, got %T", firstPage[0].Attributes["email"])
	}
	filter, err := json.Marshal(map[string]any{
		"operator": "and",
		"rules": []any{map[string]any{
			"property": "email",
			"operator": "is",
			"values":   []string{email},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	k.Call("GET", "/v1/profiles/count?"+url.Values{
		"filter":                 []string{string(filter)},
		"expectedDatasetVersion": []string{datasetVersion},
	}.Encode(), nil, nil, &countResponse)
	if countResponse.Total != 1 || countResponse.DatasetVersion != datasetVersion {
		t.Fatalf("unexpected filtered count response: total=%d version=%q", countResponse.Total, countResponse.DatasetVersion)
	}
	profiles := firstPage
	for first := pageSize; first < expectedTotal; first += pageSize {
		page, _, total, version, pageHasNext := k.ProfilesVersioned(
			[]string{"email"}, "", true, first, pageSize, datasetVersion)
		if total != expectedTotal {
			t.Fatalf("expected a total of %d profiles, got %d", expectedTotal, total)
		}
		expectedPageSize := min(pageSize, expectedTotal-first)
		if len(page) != expectedPageSize {
			t.Fatalf("expected %d profiles at offset %d, got %d", expectedPageSize, first, len(page))
		}
		if version != datasetVersion {
			t.Fatalf("expected dataset version %q, got %q", datasetVersion, version)
		}
		if pageHasNext != (first+len(page) < expectedTotal) {
			t.Fatalf("unexpected hasNext value %t at offset %d", pageHasNext, first)
		}
		profiles = append(profiles, page...)
	}

	seen := make(map[string]struct{}, len(profiles))
	hasEqualUpdateTimes := false
	for i, profile := range profiles {

		kpid := profile.KPID.String()
		if _, ok := seen[kpid]; ok {
			t.Fatalf("profile %s occurs in more than one page", kpid)
		}
		seen[kpid] = struct{}{}
		if i == 0 {
			continue
		}

		previous := profiles[i-1]
		if previous.UpdatedAt.Before(profile.UpdatedAt) {
			t.Fatalf("profile %s is newer than the preceding profile %s", profile.KPID, previous.KPID)
		}
		if previous.UpdatedAt.Equal(profile.UpdatedAt) {
			hasEqualUpdateTimes = true
			if bytes.Compare(previous.KPID[:], profile.KPID[:]) <= 0 {
				t.Fatalf("profiles %s and %s with the same update time are not ordered by descending KPID",
					previous.KPID, profile.KPID)
			}
		}

	}
	if !hasEqualUpdateTimes {
		t.Fatal("expected profiles with the same update time")
	}

	attributesPath := "/v1/profiles/" + firstPage[0].KPID.String() + "/attributes?" + url.Values{
		"expectedDatasetVersion": []string{datasetVersion},
	}.Encode()
	var attributesResponse struct {
		DatasetVersion string `json:"datasetVersion"`
	}
	k.Call("GET", attributesPath, nil, nil, &attributesResponse)
	if attributesResponse.DatasetVersion != datasetVersion {
		t.Fatalf("expected attributes dataset version %q, got %q", datasetVersion, attributesResponse.DatasetVersion)
	}

	// Empty ranges still identify the published dataset unambiguously.
	empty, _, _, emptyVersion, emptyHasNext := k.ProfilesVersioned(
		[]string{"email"}, "", true, expectedTotal, pageSize, datasetVersion)
	if len(empty) != 0 || emptyVersion != datasetVersion || emptyHasNext {
		t.Fatalf("unexpected empty range: profiles=%d version=%q hasNext=%t", len(empty), emptyVersion, emptyHasNext)
	}

	// Publishing another Identity Resolution makes the former version
	// distinguishable from a genuine profile-not-found result.
	k.RunIdentityResolutionAndWait()
	query := url.Values{
		"properties":             []string{"email"},
		"first":                  []string{"0"},
		"limit":                  []string{"1"},
		"expectedDatasetVersion": []string{datasetVersion},
	}
	err = k.TryCall("GET", "/v1/profiles?"+query.Encode(), nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `"code":"ProfileDatasetVersionMismatch"`) {
		t.Fatalf("expected a profile dataset version mismatch, got %v", err)
	}
	err = k.TryCall("GET", "/v1/profiles/count?"+url.Values{
		"expectedDatasetVersion": []string{datasetVersion},
	}.Encode(), nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `"code":"ProfileDatasetVersionMismatch"`) {
		t.Fatalf("expected a profile count dataset version mismatch, got %v", err)
	}
	err = k.TryCall("GET", attributesPath, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), `"code":"ProfileDatasetVersionMismatch"`) {
		t.Fatalf("expected an attributes dataset version mismatch, got %v", err)
	}
	for _, path := range []string{
		"/v1/profiles/" + firstPage[0].KPID.String() + "/events?limit=1&",
		"/v1/profiles/" + firstPage[0].KPID.String() + "/identities?first=0&limit=1&",
	} {

		err = k.TryCall("GET", path+url.Values{
			"expectedDatasetVersion": []string{datasetVersion},
		}.Encode(), nil, nil, nil)
		if err == nil || !strings.Contains(err.Error(), `"code":"ProfileDatasetVersionMismatch"`) {
			t.Fatalf("expected a profile dataset version mismatch from %s, got %v", path, err)
		}

	}

	_, _, _, currentDatasetVersion, _ := k.ProfilesVersioned([]string{"email"}, "", true, 0, 1, "")
	missingProfilePath := "/v1/profiles/00000000-0000-0000-0000-000000000000/attributes?" + url.Values{
		"expectedDatasetVersion": []string{currentDatasetVersion},
	}.Encode()
	err = k.TryCall("GET", missingProfilePath, nil, nil, nil)
	if err == nil || strings.Contains(err.Error(), `"code":"ProfileDatasetVersionMismatch"`) {
		t.Fatalf("expected a genuine missing profile error, got %v", err)
	}

}

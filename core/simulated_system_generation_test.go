// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

func TestSimulatedSystemPlan(t *testing.T) {
	catalog := simulatedSystemTestCatalog(t)
	for _, test := range []struct {
		name    string
		count   int
		percent string
		people  int
	}{
		{"one", 1, "0", 1},
		{"odd", 7, "33.33", 5},
		{"fractional", 101, "12.34", 89},
		{"no duplicates", 513, "0", 513},
		{"fifty percent", 10, "50", 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			account := simulatedSystemPlanFixture(t, catalog, "account-a", test.count, test.percent)
			plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog)
			if err != nil {
				t.Fatalf("expected generation plan, got %v", err)
			}
			if plan.people != test.people {
				t.Fatalf("expected %d people, got %d", test.people, plan.people)
			}
			oneByOne, oracle := simulatedSystemPlanRecords(t, plan, checkpoint, 1)
			batched, batchedOracle := simulatedSystemPlanRecords(t, plan, checkpoint, simulatedSystemBatchSize)
			if !reflect.DeepEqual(oneByOne, batched) || !reflect.DeepEqual(oracle, batchedOracle) {
				t.Fatal("expected identical records and Oracle across batch sizes, got different results")
			}
			if len(batched) != test.count {
				t.Fatalf("expected %d records, got %d", test.count, len(batched))
			}
			ids := map[string]struct{}{}
			people := map[string]int{}
			for i, record := range batched {
				if _, exists := ids[record.ID]; exists {
					t.Fatalf("expected unique record IDs, got duplicate %s", record.ID)
				}
				ids[record.ID] = struct{}{}
				people[oracle[i].PersonID]++
				if oracle[i].SourceRecordID != record.ID ||
					strings.Contains(string(record.Data), oracle[i].PersonID) ||
					strings.Contains(string(record.Data), "oracle") || strings.Contains(string(record.Data), "role") {
					t.Fatalf("expected ordinary record without Oracle or role, got %s", record.Data)
				}
			}
			if len(people) != test.people {
				t.Fatalf("expected %d distinct people, got %d", test.people, len(people))
			}
			double := 0
			for _, count := range people {
				if count == 2 {
					double++
				} else if count != 1 {
					t.Fatalf("expected one or two representations, got %d", count)
				}
			}
			if double != test.count-test.people {
				t.Fatalf("expected %d duplicate people, got %d", test.count-test.people, double)
			}
		})
	}
}

func TestSimulatedSystemPlanMaximum(t *testing.T) {

	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "account-a", 10_000_000, "50")
	plan, _, err := newSimulatedSystemPlan("workspace-a", account, catalog)
	if err != nil {
		t.Fatalf("expected generation plan, got %v", err)
	}
	if plan.people != 5_000_000 || plan.duplicate != 5_000_000 {
		t.Fatalf("expected five million people and duplicates, got %d and %d", plan.people, plan.duplicate)
	}

	account.UserCount++
	_, _, err = newSimulatedSystemPlan("workspace-a", account, catalog)
	if err == nil {
		t.Fatal("expected invalid generation configuration, got no error")
	}

}

func TestSimulatedSystemPlanResumeAndSelection(t *testing.T) {
	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "account-a", 513, "50")
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog)
	if err != nil {
		t.Fatalf("expected generation plan, got %v", err)
	}
	baseline, oracle := simulatedSystemPlanRecords(t, plan, checkpoint, 256)
	for range 257 {
		checkpoint = plan.advance(checkpoint)
	}
	account.GeneratedRecordCount = 257
	account.GenerationCheckpoint, err = json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("expected checkpoint JSON, got %v", err)
	}
	resumed, resumedCheckpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog)
	if err != nil {
		t.Fatalf("expected resumed generation plan, got %v", err)
	}
	remaining, remainingOracle := simulatedSystemPlanRecords(t, resumed, resumedCheckpoint, 17)
	if !reflect.DeepEqual(remaining, baseline[257:]) || !reflect.DeepEqual(remainingOracle, oracle[257:]) {
		t.Fatal("expected exact resumed suffix, got different records or Oracle")
	}

	first := simulatedSystemPlanFixture(t, catalog, "account-a", 30_000, "0")
	second := simulatedSystemPlanFixture(t, catalog, "account-b", 30_000, "0")
	a, _, err := newSimulatedSystemPlan("workspace-a", first, catalog)
	if err != nil {
		t.Fatalf("expected first selection plan, got %v", err)
	}
	b, _, err := newSimulatedSystemPlan("workspace-a", second, catalog)
	if err != nil {
		t.Fatalf("expected second selection plan, got %v", err)
	}
	selected := map[fakedata.PersonIndex]struct{}{}
	for i := range 30_000 {
		selected[a.personIndex(i)] = struct{}{}
	}
	overlap := 0
	var shared fakedata.PersonIndex
	for i := range 30_000 {
		if _, ok := selected[b.personIndex(i)]; ok {
			overlap++
			shared = b.personIndex(i)
		}
	}
	if len(selected) != 30_000 || overlap == 0 || a.personIndex(0) == b.personIndex(0) {
		t.Fatalf("expected distinct account selections with overlap, got unique=%d overlap=%d", len(selected), overlap)
	}
	canonicalA, err := a.world.Person(shared)
	if err != nil {
		t.Fatalf("expected first account canonical person, got %v", err)
	}
	canonicalB, err := b.world.Person(shared)
	if err != nil {
		t.Fatalf("expected second account canonical person, got %v", err)
	}
	if !reflect.DeepEqual(canonicalA, canonicalB) {
		t.Fatalf("expected shared workspace truth, got %#v and %#v", canonicalA, canonicalB)
	}

	var changed simulatedSystemCheckpoint
	err = json.Unmarshal(first.GenerationCheckpoint, &changed)
	if err != nil {
		t.Fatalf("expected initial checkpoint JSON, got %v", err)
	}
	changed.CatalogSHA256 = "different"
	first.GenerationCheckpoint, err = json.Marshal(changed)
	if err != nil {
		t.Fatalf("expected changed checkpoint JSON, got %v", err)
	}
	_, _, err = newSimulatedSystemPlan("workspace-a", first, catalog)
	if err == nil {
		t.Fatal("expected changed catalog fingerprint to fail, got success")
	}
	changed.CatalogSHA256 = checkpoint.CatalogSHA256
	changed.WorldModelVersion = "missing"
	first.GenerationCheckpoint, err = json.Marshal(changed)
	if err != nil {
		t.Fatalf("expected changed world version JSON, got %v", err)
	}
	_, _, err = newSimulatedSystemPlan("workspace-a", first, catalog)
	if err == nil {
		t.Fatal("expected changed world version to fail, got success")
	}
	changed.WorldModelVersion = checkpoint.WorldModelVersion
	changed.PersonSpecVersion = "missing"
	first.GenerationCheckpoint, err = json.Marshal(changed)
	if err != nil {
		t.Fatalf("expected changed person specification JSON, got %v", err)
	}
	_, _, err = newSimulatedSystemPlan("workspace-a", first, catalog)
	if err == nil {
		t.Fatal("expected changed person specification to fail, got success")
	}
	changed.PersonSpecVersion = checkpoint.PersonSpecVersion
	changed.SourceVersion = "missing"
	first.GenerationCheckpoint, err = json.Marshal(changed)
	if err != nil {
		t.Fatalf("expected changed source version JSON, got %v", err)
	}
	_, _, err = newSimulatedSystemPlan("workspace-a", first, catalog)
	if err == nil {
		t.Fatal("expected changed source version to fail, got success")
	}
}

func TestSimulatedSystemDocument(t *testing.T) {
	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "account-a", 40, "50")
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog)
	if err != nil {
		t.Fatalf("expected generation plan, got %v", err)
	}

	var complete datastore.SimulatedSystemRecord
	var completeSource fakedata.SourceRecord
	var completePerson fakedata.CountryPerson
	var missingEmail datastore.SimulatedSystemRecord
	var missingEmailSource fakedata.SourceRecord
	for person := 0; person < plan.people; person++ {
		records, err := plan.source.Records(plan.personIndex(person))
		if err != nil {
			t.Fatalf("expected source records, got %v", err)
		}
		if len(records) == 0 {
			t.Fatalf("expected source records for a covered person, got none")
		}
		if complete.Data == nil && records[0].Email != nil && records[0].PhotoID != nil {
			complete, err = plan.record(person, 0)
			if err != nil {
				t.Fatalf("expected complete simulated document, got %v", err)
			}
			completeSource = records[0]
			completePerson, err = plan.world.Person(plan.personIndex(person))
			if err != nil {
				t.Fatalf("expected canonical person, got %v", err)
			}
		}
		if missingEmail.Data == nil && records[0].Email == nil {
			missingEmail, err = plan.record(person, 0)
			if err != nil {
				t.Fatalf("expected missing-email simulated document, got %v", err)
			}
			missingEmailSource = records[0]
		}
		if complete.Data != nil && missingEmail.Data != nil {
			break
		}
	}
	if complete.Data == nil || missingEmail.Data == nil {
		t.Fatalf("expected complete and missing-email documents, got complete=%t missingEmail=%t",
			complete.Data != nil, missingEmail.Data != nil)
	}

	var document struct {
		CustomerID string                  `json:"customer_id"`
		FirstName  *string                 `json:"first_name"`
		LastName   *string                 `json:"last_name"`
		Email      *string                 `json:"email"`
		Gender     fakedata.Gender         `json:"gender"`
		BirthDate  string                  `json:"birth_date"`
		CurrentAge int                     `json:"current_age"`
		Phone      *string                 `json:"phone"`
		SiteURL    string                  `json:"site_url"`
		Address    *simulatedSystemAddress `json:"address"`
		Photo      *string                 `json:"photo"`
	}
	err = json.Unmarshal(complete.Data, &document)
	if err != nil {
		t.Fatalf("expected complete document JSON, got %v", err)
	}
	if document.CustomerID != complete.ID || document.CustomerID != completeSource.ID {
		t.Fatalf("expected customer_id to equal external_id %q, got %q", complete.ID, document.CustomerID)
	}
	if document.FirstName == nil || *document.FirstName != *completeSource.FirstName ||
		document.LastName == nil || *document.LastName != *completeSource.LastName ||
		document.Email == nil || *document.Email != *completeSource.Email ||
		document.Phone == nil || *document.Phone != *completeSource.Phone ||
		document.Photo == nil || !strings.HasPrefix(*document.Photo, "https://krenalis.test/photos/") ||
		!strings.HasSuffix(*document.Photo, "/256/"+*completeSource.PhotoID+".webp") {
		t.Fatalf("expected observation-specific fields, got %+v", document)
	}
	if document.Gender != completePerson.Person.Gender || document.BirthDate != completePerson.Person.BirthDate ||
		document.CurrentAge != completePerson.Person.CurrentAge || document.SiteURL != completePerson.Person.SiteURL {
		t.Fatalf("expected canonical Layer-1 fields, got %+v", document)
	}
	if document.Address == nil || document.Address.Street != completePerson.Person.Address.Street ||
		document.Address.Municipality != completePerson.Person.Address.Municipality ||
		document.Address.StateProv != completePerson.Person.Address.StateProv ||
		document.Address.PostalCode != completePerson.Person.Address.PostalCode ||
		document.Address.Country == nil || *document.Address.Country != *completeSource.Country {
		t.Fatalf("expected complete address, got %+v", document.Address)
	}
	var fields map[string]json.Value
	err = json.Unmarshal(complete.Data, &fields)
	if err != nil {
		t.Fatalf("expected document fields, got %v", err)
	}
	if len(fields) != 11 {
		t.Fatalf("expected exactly 11 document fields, got %v", fields)
	}
	for _, field := range []string{"customer_id", "first_name", "last_name", "email", "gender", "birth_date",
		"current_age", "phone", "site_url", "address", "photo"} {
		if _, ok := fields[field]; !ok {
			t.Fatalf("expected document field %q, got %v", field, fields)
		}
	}
	for _, field := range []string{"source_record_id", "photo_id", "photo_url", "country", "person_id", "oracle", "role"} {
		if _, ok := fields[field]; ok || strings.Contains(string(complete.Data), completePerson.Person.ID) {
			t.Fatalf("expected obsolete or internal field %q to be absent, got %s", field, complete.Data)
		}
	}

	fields = map[string]json.Value{}
	err = json.Unmarshal(missingEmail.Data, &fields)
	if err != nil {
		t.Fatalf("expected missing-email document fields, got %v", err)
	}
	if _, ok := fields["email"]; ok || missingEmailSource.Email != nil {
		t.Fatalf("expected missing observed email to remain absent, got %v", fields["email"])
	}

	canonical, err := plan.world.Person(plan.personIndex(0))
	if err != nil {
		t.Fatalf("expected canonical person for partial address, got %v", err)
	}
	partial := newSimulatedSystemDocument(canonical, fakedata.SourceRecord{ID: "partial"}, nil)
	if partial.Address == nil || partial.Address.Street != canonical.Person.Address.Street || partial.Address.Country != nil {
		t.Fatalf("expected partial address observation, got %+v", partial.Address)
	}

	first, err := plan.record(checkpoint.Person, checkpoint.Representation)
	if err != nil {
		t.Fatalf("expected first generated record, got %v", err)
	}
	checkpoint = plan.advance(checkpoint)
	second, err := plan.record(checkpoint.Person, checkpoint.Representation)
	if err != nil {
		t.Fatalf("expected second generated record, got %v", err)
	}
	if first.ID == second.ID || string(first.Data) == string(second.Data) {
		t.Fatalf("expected duplicate observations to retain distinct IDs and data, got %s and %s", first.Data, second.Data)
	}
}

func simulatedSystemPlanRecords(t *testing.T, plan *simulatedSystemPlan, checkpoint simulatedSystemCheckpoint, batchSize int) ([]datastore.SimulatedSystemRecord, []fakedata.OracleRecord) {
	t.Helper()
	remaining := plan.account.UserCount - plan.account.GeneratedRecordCount
	records := make([]datastore.SimulatedSystemRecord, 0, remaining)
	oracle := make([]fakedata.OracleRecord, 0, remaining)
	for len(records) < remaining {
		for i := 0; i < batchSize && len(records) < remaining; i++ {
			record, err := plan.record(checkpoint.Person, checkpoint.Representation)
			if err != nil {
				t.Fatalf("expected source record, got %v", err)
			}
			truth, err := plan.oracle(checkpoint.Person, checkpoint.Representation)
			if err != nil {
				t.Fatalf("expected Oracle record, got %v", err)
			}
			records = append(records, record)
			oracle = append(oracle, truth)
			checkpoint = plan.advance(checkpoint)
		}
	}
	return records, oracle
}

func simulatedSystemPlanFixture(t *testing.T, catalog *fakedata.FaceCatalog, id string, count int, percent string) *state.SimulatedSystem {
	t.Helper()
	checkpoint, err := initialSimulatedSystemCheckpoint(catalog, "https://krenalis.test/")
	if err != nil {
		t.Fatalf("expected initial checkpoint, got %v", err)
	}
	value, err := decimal.Parse(percent, 5, 2)
	if err != nil {
		t.Fatalf("expected duplicate percentage, got %v", err)
	}
	return &state.SimulatedSystem{ID: id, UserCount: count, DuplicateRecordPercent: value,
		GenerationPolicyVersion: simulatedSystemPolicyVersion, GenerationCheckpoint: checkpoint}
}

func simulatedSystemTestCatalog(t *testing.T) *fakedata.FaceCatalog {
	t.Helper()
	root := filepath.Join("..", "tools", "fakedata", "testdata", "fakefacegen-v1-selected")
	directory := t.TempDir()
	var catalog, manifest map[string]any
	var specs []map[string]any
	for _, item := range []struct {
		name string
		out  any
	}{{"catalog.json", &catalog}, {"specs.json", &specs}, {"manifest.json", &manifest}} {
		data, err := os.ReadFile(filepath.Join(root, item.name))
		if err != nil {
			t.Fatalf("expected face catalog fixture %s, got %v", item.name, err)
		}
		err = json.Unmarshal(data, item.out)
		if err != nil {
			t.Fatalf("expected valid face catalog fixture %s, got %v", item.name, err)
		}
	}
	catalog["count"] = 2
	newSpecs := make([]map[string]any, 0, 2)
	newAssets := make([]map[string]any, 0, 2)
	for index, sourceID := range []string{"face-000013", "face-000037"} {
		id := fmt.Sprintf("face-%06d", index+1)
		for _, spec := range specs {
			if spec["id"] == sourceID {
				spec["id"] = id
				newSpecs = append(newSpecs, spec)
				break
			}
		}
		for _, item := range manifest["assets"].([]any) {
			asset := item.(map[string]any)
			if asset["id"] == sourceID {
				asset["id"] = id
				asset["spec_id"] = id
				asset["file"] = "masters/" + id + ".webp"
				newAssets = append(newAssets, asset)
				break
			}
		}
		for _, size := range []int{64, 128, 256, 512, 1024} {
			folder := filepath.Join("derived", strconv.Itoa(size))
			if size == 1024 {
				folder = "masters"
			}
			data, err := os.ReadFile(filepath.Join(root, folder, sourceID+".webp"))
			if err != nil {
				t.Fatalf("expected face fixture %s/%s, got %v", folder, sourceID, err)
			}
			path := filepath.Join(directory, folder)
			err = os.MkdirAll(path, 0o755)
			if err != nil {
				t.Fatalf("expected face fixture directory %s, got %v", path, err)
			}
			err = os.WriteFile(filepath.Join(path, id+".webp"), data, 0o644)
			if err != nil {
				t.Fatalf("expected face fixture asset %s, got %v", id, err)
			}
		}
	}
	manifest["assets"] = newAssets
	for _, item := range []struct {
		name  string
		value any
	}{{"catalog.json", catalog}, {"specs.json", newSpecs}, {"manifest.json", manifest}} {
		data, err := json.Marshal(item.value)
		if err != nil {
			t.Fatalf("expected face fixture JSON %s, got %v", item.name, err)
		}
		data, err = json.Canonicalize(data)
		if err != nil {
			t.Fatalf("expected canonical face fixture JSON %s, got %v", item.name, err)
		}
		err = os.WriteFile(filepath.Join(directory, item.name), data, 0o644)
		if err != nil {
			t.Fatalf("expected face fixture metadata %s, got %v", item.name, err)
		}
	}
	result, err := fakedata.LoadFaceCatalog(t.Context(), directory)
	if err != nil {
		t.Fatalf("expected complete face catalog, got %v", err)
	}
	return result
}

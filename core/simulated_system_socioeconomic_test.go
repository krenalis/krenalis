// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/simulated"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSimulatedSystemSocioeconomicDocuments checks all six activity branches
// against canonical facts, while preserving the complete legacy observations.
func TestSimulatedSystemSocioeconomicDocuments(t *testing.T) {

	model := simulatedSystemTestModel(t)
	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "account-a", 512, "50")
	legacy, _, err := newSimulatedSystemPlan("workspace-a", account, catalog, nil)
	if err != nil {
		t.Fatalf("expected legacy plan, got %v", err)
	}
	upgraded := *account
	upgraded.GenerationPolicyVersion = simulatedSystemPolicyVersion
	plan, _, err := newSimulatedSystemPlan("workspace-a", &upgraded, catalog, model)
	if err != nil {
		t.Fatalf("expected socioeconomic plan, got %v", err)
	}
	base := reflect.ValueOf(plan.world).Elem().FieldByName("base").Pointer()
	if reflect.ValueOf(plan.socioeconomic).Elem().FieldByName("base").Pointer() != base {
		t.Fatal("expected the exact SourceWorld base pointer, got a different socioeconomic base")
	}
	application := &simulated.Application{}
	schema, err := application.RecordSchema(t.Context(), connectors.TargetUser, connectors.Source)
	if err != nil {
		t.Fatalf("expected Source schema, got %v", err)
	}

	activities := map[fakedata.PrimaryActivity]bool{}
	education := map[fakedata.EducationLevel]bool{}
	hasZero, hasMaximum := false, false
	for slot := range plan.people {

		index := plan.personIndex(slot)
		facts, err := plan.socioeconomic.Facts(index)
		if err != nil {
			t.Fatalf("expected canonical facts at %d, got %v", index, err)
		}
		activities[facts.PrimaryActivity] = true
		education[facts.Education] = true
		hasZero = hasZero || facts.Income == 0
		hasMaximum = hasMaximum || facts.Income == 1<<32-1
		want := map[string]any{
			"education_level": string(facts.Education), "primary_activity": string(facts.PrimaryActivity),
			"annual_income_eur": uint(facts.Income),
		}
		switch facts.PrimaryActivity {
		case fakedata.PrimaryActivityEmployee:
			company, err := plan.socioeconomic.Company(facts.Employee.Employer)
			if err != nil {
				t.Fatalf("expected selected Employer Company, got %v", err)
			}
			want["job_family"] = string(facts.Employee.JobFamily)
			want["industry"] = string(company.Industry)
			want["job_level"] = string(facts.Employee.JobLevel)
		case fakedata.PrimaryActivitySelfEmployed:
			want["job_family"] = string(facts.SelfEmployed.JobFamily)
			want["industry"] = string(facts.SelfEmployed.Industry)
		}

		var firstID string
		for representation := range 2 {

			record, err := plan.record(slot, representation)
			if err != nil {
				t.Fatalf("expected socioeconomic record, got %v", err)
			}
			attributes, err := types.Decode[map[string]any](bytes.NewReader(record.Data), schema)
			if err != nil {
				t.Fatalf("expected generated document to decode against Source schema, got %v", err)
			}
			for _, name := range []string{"education_level", "primary_activity", "annual_income_eur",
				"job_family", "industry", "job_level"} {
				actual, present := attributes[name]
				expected, applicable := want[name]
				if present != applicable || actual != expected {
					t.Fatalf("expected %s=%v present=%t for %s, got %v present=%t",
						name, expected, applicable, facts.PrimaryActivity, actual, present)
				}
			}
			fields := simulatedSystemDocumentFields(t, record.Data)
			if string(fields["annual_income_eur"]) != strconv.FormatUint(uint64(facts.Income), 10) {
				t.Fatalf("expected exact whole-EUR JSON integer %d, got %s", facts.Income, fields["annual_income_eur"])
			}
			for name := range want {
				delete(fields, name)
			}
			old, err := legacy.record(slot, representation)
			if err != nil {
				t.Fatalf("expected legacy observation, got %v", err)
			}
			if record.ID != old.ID || !reflect.DeepEqual(fields, simulatedSystemDocumentFields(t, old.Data)) {
				t.Fatal("expected applicable socioeconomic additions, got changed observations or extra properties")
			}
			truth, err := plan.oracle(slot, representation)
			if err != nil {
				t.Fatalf("expected unchanged Oracle, got %v", err)
			}
			oldTruth, err := legacy.oracle(slot, representation)
			if err != nil {
				t.Fatalf("expected legacy Oracle, got %v", err)
			}
			if truth != oldTruth || bytes.Contains(record.Data, []byte(truth.PersonID)) {
				t.Fatal("expected unchanged, separate Oracle, got changed or serialized canonical identity")
			}
			if representation == 0 {
				firstID = record.ID
			}
			if representation == 1 && record.ID == firstID {
				t.Fatalf("expected distinct duplicate IDs, got %s", record.ID)
			}

		}

	}
	for _, activity := range []fakedata.PrimaryActivity{
		fakedata.PrimaryActivityEmployee, fakedata.PrimaryActivitySelfEmployed, fakedata.PrimaryActivityUnemployed,
		fakedata.PrimaryActivityStudent, fakedata.PrimaryActivityRetired, fakedata.PrimaryActivityInactive,
	} {
		if !activities[activity] {
			t.Fatalf("expected generated %s coverage, got none", activity)
		}
	}
	if !hasZero || !hasMaximum || len(education) < 2 {
		t.Fatalf("expected zero, maximum Income and varied Education, got zero=%t maximum=%t Education=%v",
			hasZero, hasMaximum, education)
	}

}

// TestSimulatedSystemSocioeconomicFailures checks that required facts cannot
// silently disappear and legacy generation does not require a model.
func TestSimulatedSystemSocioeconomicFailures(t *testing.T) {

	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "account-a", 1, "0")
	legacy, _, err := newSimulatedSystemPlan("workspace-a", account, catalog, nil)
	if err != nil {
		t.Fatalf("expected legacy plan without socioeconomic model, got %v", err)
	}
	if legacy.socioeconomic != nil {
		t.Fatalf("expected legacy plan without socioeconomic world, got %v", legacy.socioeconomic)
	}
	account.GenerationPolicyVersion = simulatedSystemPolicyVersion
	_, _, err = newSimulatedSystemPlan("workspace-a", account, catalog, nil)
	if err == nil {
		t.Fatal("expected missing new-policy model to fail, got success")
	}
	model := simulatedSystemTestModel(t)
	plan, _, err := newSimulatedSystemPlan("workspace-a", account, catalog, model)
	if err != nil {
		t.Fatalf("expected socioeconomic plan, got %v", err)
	}
	for _, test := range []struct {
		name  string
		facts fakedata.SocioeconomicFacts
	}{
		{"missing education", fakedata.SocioeconomicFacts{PrimaryActivity: fakedata.PrimaryActivityInactive}},
		{"missing activity", fakedata.SocioeconomicFacts{Education: fakedata.EducationLevelTertiary}},
		{"missing employee", fakedata.SocioeconomicFacts{
			Education: fakedata.EducationLevelTertiary, PrimaryActivity: fakedata.PrimaryActivityEmployee}},
		{"missing self-employed", fakedata.SocioeconomicFacts{
			Education: fakedata.EducationLevelTertiary, PrimaryActivity: fakedata.PrimaryActivitySelfEmployed}},
		{"unexpected occupation", fakedata.SocioeconomicFacts{
			Education: fakedata.EducationLevelTertiary, PrimaryActivity: fakedata.PrimaryActivityStudent,
			Employee: &fakedata.EmployeeFacts{Employer: 1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var document simulatedSystemDocument
			err := setSimulatedSystemSocioeconomicFacts(&document, plan.socioeconomic, test.facts)
			if err == nil {
				t.Fatal("expected inconsistent facts to fail, got success")
			}
		})
	}
	var document simulatedSystemDocument
	err = setSimulatedSystemSocioeconomicFacts(&document, plan.socioeconomic, fakedata.SocioeconomicFacts{
		Education: fakedata.EducationLevelTertiary, PrimaryActivity: fakedata.PrimaryActivityEmployee,
		Employee: &fakedata.EmployeeFacts{Employer: 0,
			JobFamily: fakedata.JobFamilyAdministration, JobLevel: fakedata.JobLevelManager},
	})
	if err != nil {
		if !errors.Is(err, fakedata.ErrInvalidIndex) {
			t.Fatalf("expected Company lookup error, got %v", err)
		}
		return
	}

	t.Fatal("expected failed Company lookup to propagate, got success")

}

// TestSimulatedSystemSocioeconomicResume checks model reuse, persisted cursors
// and overlapping canonical truth across systems in the same workspace.
func TestSimulatedSystemSocioeconomicResume(t *testing.T) {

	model := simulatedSystemTestModel(t)
	catalog := simulatedSystemTestCatalog(t)
	account := simulatedSystemPlanFixture(t, catalog, "account-a", 513, "50")
	account.GenerationPolicyVersion = simulatedSystemPolicyVersion
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, catalog, model)
	if err != nil {
		t.Fatalf("expected socioeconomic plan, got %v", err)
	}
	baseline, oracle := simulatedSystemPlanRecords(t, plan, checkpoint, 256)
	for range 257 {
		checkpoint = plan.advance(checkpoint)
	}
	account.GeneratedRecordCount = 257
	account.GenerationCheckpoint, err = json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("expected persisted checkpoint, got %v", err)
	}
	resumed, next, err := newSimulatedSystemPlan("workspace-a", account, catalog, model)
	if err != nil {
		t.Fatalf("expected new-policy resume at duplicate boundary, got %v", err)
	}
	remaining, remainingOracle := simulatedSystemPlanRecords(t, resumed, next, 17)
	if !reflect.DeepEqual(remaining, baseline[257:]) || !reflect.DeepEqual(remainingOracle, oracle[257:]) {
		t.Fatal("expected exact resumed IDs, documents and Oracle, got changed suffix")
	}

	first := simulatedSystemPlanFixture(t, catalog, "account-a", 30_000, "0")
	second := simulatedSystemPlanFixture(t, catalog, "account-b", 30_000, "0")
	first.GenerationPolicyVersion = simulatedSystemPolicyVersion
	second.GenerationPolicyVersion = simulatedSystemPolicyVersion
	a, _, err := newSimulatedSystemPlan("workspace-a", first, catalog, model)
	if err != nil {
		t.Fatalf("expected first system plan, got %v", err)
	}
	b, _, err := newSimulatedSystemPlan("workspace-a", second, catalog, model)
	if err != nil {
		t.Fatalf("expected second system plan, got %v", err)
	}
	selected := map[fakedata.PersonIndex]int{}
	for slot := range a.people {
		selected[a.personIndex(slot)] = slot
	}
	for slot := range b.people {

		index := b.personIndex(slot)
		other, ok := selected[index]
		if !ok {
			continue
		}
		factsA, err := a.socioeconomic.Person(index)
		if err != nil {
			t.Fatalf("expected first canonical person and facts, got %v", err)
		}
		factsB, err := b.socioeconomic.Person(index)
		if err != nil {
			t.Fatalf("expected second canonical person and facts, got %v", err)
		}
		canonical, err := a.world.Person(index)
		if err != nil {
			t.Fatalf("expected SourceWorld canonical person, got %v", err)
		}
		if !reflect.DeepEqual(factsA, factsB) || !reflect.DeepEqual(factsA.Person, canonical.Person) {
			t.Fatalf("expected shared workspace canonical truth at %d, got different Person or facts", index)
		}
		left, err := a.record(other, 0)
		if err != nil {
			t.Fatalf("expected first overlapping record, got %v", err)
		}
		right, err := b.record(slot, 0)
		if err != nil {
			t.Fatalf("expected second overlapping record, got %v", err)
		}
		l, r := simulatedSystemDocumentFields(t, left.Data), simulatedSystemDocumentFields(t, right.Data)
		for _, name := range []string{"education_level", "primary_activity", "annual_income_eur",
			"job_family", "industry", "job_level"} {
			if !bytes.Equal(l[name], r[name]) {
				t.Fatalf("expected equal %s for overlapping index %d, got %s and %s", name, index, l[name], r[name])
			}
		}
		if left.ID == right.ID {
			t.Fatalf("expected system-specific observation IDs, got %s", left.ID)
		}

		return
	}

	t.Fatal("expected overlapping canonical indices, got none")

}

func simulatedSystemDocumentFields(t *testing.T, data json.Value) map[string]json.Value {
	t.Helper()
	var fields map[string]json.Value
	err := json.Unmarshal(data, &fields)
	if err != nil {
		t.Fatalf("expected document JSON, got %v", err)
	}
	return fields
}

// simulatedSystemTestModel acquires a small artificial canonical model with
// varied laws, then proves that the Core owner reuses it after origins close.
func simulatedSystemTestModel(t *testing.T) *fakedata.SocioeconomicModel {

	t.Helper()
	fixture, err := os.ReadFile("testdata/socioeconomic-model.json")
	if err != nil {
		t.Fatalf("expected artificial socioeconomic fixture, got %v", err)
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/synthetic-world/it-2026a/model.json" {
			t.Errorf("expected fixed model path, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)
	source, err := fakedata.NewSocioeconomicModelSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected acquisition owner, got %v", err)
	}
	core := &Core{simulatedModelSource: source}
	if calls.Load() != 0 {
		t.Fatalf("expected no request during source construction, got %d", calls.Load())
	}
	model, err := core.simulatedModelSource.Load(t.Context())
	if err != nil {
		t.Fatalf("expected acquired socioeconomic model, got %v", err)
	}
	server.Close()
	reused, err := core.simulatedModelSource.Load(t.Context())
	if err != nil {
		t.Fatalf("expected retained model after origin closure, got %v", err)
	}
	if model != reused || calls.Load() != 1 {
		t.Fatalf("expected same acquired model and one request, got same=%t requests=%d", model == reused, calls.Load())
	}

	return model
}

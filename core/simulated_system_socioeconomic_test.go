// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/simulated"
	"github.com/krenalis/krenalis/test/syntheticworldtest"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestSimulatedSystemSocioeconomicDocuments checks all six activity branches
// against canonical facts and the observations of the shared SourceWorld.
func TestSimulatedSystemSocioeconomicDocuments(t *testing.T) {

	data := simulatedSystemTestData(t)
	account := simulatedSystemPlanFixture(t, data, "account-a", 512, "50")
	plan, _, err := newSimulatedSystemPlan("workspace-a", account, data)
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
			observations, err := plan.source.Records(index)
			if err != nil {
				t.Fatalf("expected Source observations, got %v", err)
			}
			canonical, err := plan.world.Person(index)
			if err != nil {
				t.Fatalf("expected canonical Person, got %v", err)
			}
			var photo *string
			if observations[representation].PhotoID != nil {
				asset, err := plan.catalog.Asset(*observations[representation].PhotoID, fakedata.PhotoSize256)
				if err != nil {
					t.Fatalf("expected declared rendition, got %v", err)
				}
				value := plan.photoBaseURL.JoinPath(asset.Path).String()
				photo = &value
			}
			baseJSON, err := json.Marshal(newSimulatedSystemDocument(canonical, observations[representation], photo))
			if err != nil {
				t.Fatalf("expected base document, got %v", err)
			}
			if record.ID != observations[representation].ID ||
				!reflect.DeepEqual(fields, simulatedSystemDocumentFields(t, baseJSON)) {
				t.Fatal("expected applicable socioeconomic additions, got changed observations or extra properties")
			}
			truth, err := plan.oracle(slot, representation)
			if err != nil {
				t.Fatalf("expected separate Oracle, got %v", err)
			}
			if truth.PersonID != canonical.Person.ID || bytes.Contains(record.Data, []byte(truth.PersonID)) {
				t.Fatal("expected separate canonical identity, got wrong or serialized Oracle")
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
// silently disappear and incomplete prepared data cannot be admitted.
func TestSimulatedSystemSocioeconomicFailures(t *testing.T) {

	data := simulatedSystemTestData(t)
	account := simulatedSystemPlanFixture(t, data, "account-a", 1, "0")
	_, _, err := newSimulatedSystemPlan("workspace-a", account, nil)
	if err == nil {
		t.Fatal("expected missing prepared data to fail, got success")
	}
	plan, _, err := newSimulatedSystemPlan("workspace-a", account, data)
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

	data := simulatedSystemTestData(t)
	account := simulatedSystemPlanFixture(t, data, "account-a", 513, "50")
	account.GenerationPolicyVersion = simulatedSystemPolicyVersion
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, data)
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
	resumed, next, err := newSimulatedSystemPlan("workspace-a", account, data)
	if err != nil {
		t.Fatalf("expected new-policy resume at duplicate boundary, got %v", err)
	}
	remaining, remainingOracle := simulatedSystemPlanRecords(t, resumed, next, 17)
	if !reflect.DeepEqual(remaining, baseline[257:]) || !reflect.DeepEqual(remainingOracle, oracle[257:]) {
		t.Fatal("expected exact resumed IDs, documents and Oracle, got changed suffix")
	}

	first := simulatedSystemPlanFixture(t, data, "account-a", 30_000, "0")
	second := simulatedSystemPlanFixture(t, data, "account-b", 30_000, "0")
	first.GenerationPolicyVersion = simulatedSystemPolicyVersion
	second.GenerationPolicyVersion = simulatedSystemPolicyVersion
	a, _, err := newSimulatedSystemPlan("workspace-a", first, data)
	if err != nil {
		t.Fatalf("expected first system plan, got %v", err)
	}
	b, _, err := newSimulatedSystemPlan("workspace-a", second, data)
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

// simulatedSystemTestData verifies lazy, shared acquisition of all four JSONs.
func simulatedSystemTestData(t *testing.T) *fakedata.ItalianWorldData {

	t.Helper()
	assets := syntheticworldtest.Assets(t)
	var calls atomic.Int32
	handler := syntheticworldtest.Handler(t, assets, "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	source, err := fakedata.NewItalianWorldSource([]string{server.URL})
	if err != nil {
		t.Fatalf("expected acquisition owner, got %v", err)
	}
	core := &Core{simulatedWorldSource: source}
	if calls.Load() != 0 {
		t.Fatalf("expected no request during source construction, got %d", calls.Load())
	}
	data, err := core.simulatedWorldSource.Load(t.Context())
	if err != nil {
		t.Fatalf("expected prepared Italian data, got %v", err)
	}
	server.Close()
	reused, err := core.simulatedWorldSource.Load(t.Context())
	if err != nil {
		t.Fatalf("expected retained data after origin closure, got %v", err)
	}
	if data != reused || calls.Load() != 4 {
		t.Fatalf("expected shared prepared data and four JSON requests, got same=%t requests=%d", data == reused, calls.Load())
	}

	return data
}

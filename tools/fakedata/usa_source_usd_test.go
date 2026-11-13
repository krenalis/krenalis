// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"testing"
)

// decode verifies a bounded closed transport against its original logical
// record. It is not an application importer or an official checkpoint reader.
func (a *usaSourceUSDAdapter) decode(ctx context.Context, encoded []byte, expected usaSourceUSDRecord) (usaSourceUSDRecord, error) {

	if len(encoded) > usaSourceUSDRecordBytes {
		return usaSourceUSDRecord{}, fmt.Errorf("USA source record exceeds byte limit")
	}
	var record usaSourceUSDRecord
	err := decodeInternationalJSON(ctx, bytes.NewReader(encoded), &record)
	if err != nil {
		return usaSourceUSDRecord{}, err
	}
	err = a.validate(record)
	if err != nil {
		return usaSourceUSDRecord{}, err
	}
	if !reflect.DeepEqual(record, expected) {
		return usaSourceUSDRecord{}, fmt.Errorf("decoded USA observation differs from its logical source record")
	}

	return record, nil
}

// TestUSASourceUSDDispatchAndGates rejects incompatible runtime source admission.
func TestUSASourceUSDDispatchAndGates(t *testing.T) {

	t.Run("official-World-gate", func(t *testing.T) {
		_, err := NewSourceWorld(nil, []CountryShare{{Code: "US", Version: usaSourceUSDVersion, Weight: 1}})
		if err != nil {
			return
		}
		t.Fatal("expected official World rejection without prepared base, got admission")
	})
	t.Run("official-photo-gate", func(t *testing.T) {
		_, err := NewSourceCSVWriter(io.Discard, nil)
		if err != nil {
			return
		}
		t.Fatal("expected official CSV photo catalog gate, got writer")
	})
	t.Run("USA-unavailable-in-Italian-source", func(t *testing.T) {
		base := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
		_, err := NewSourceWorld(base, []CountryShare{{Code: "US", Version: usaSourceUSDVersion, Weight: 1}})
		if err != nil {
			return
		}
		t.Fatal("expected unavailable US generator rejection, got Italian fallback")
	})
	a := readUSASourceUSDAdapter(t)
	t.Run("no-Italian-fallback", func(t *testing.T) {
		identity, err := StringComponent("internal-world-id", string(a.world.id))
		if err != nil {
			t.Fatalf("expected internal identity component, got %v", err)
		}
		_, err = newCountrySourceWorld(nil, a.source.world.namespace, identity,
			[]CountryShare{{Code: "IT", Version: MarketDataVersion, Weight: 1}})
		if err != nil {
			return
		}
		t.Fatal("expected USA owner rejection of Italian source fallback, got admission")
	})
	t.Run("source-version", func(t *testing.T) {
		config := a.source.config
		config.Version = "generation-v5"
		_, err := newUSASourceUSDAdapter(a.world, config)
		if err != nil {
			return
		}
		t.Fatal("expected incompatible source version rejection, got adapter")
	})
	if reflect.TypeFor[usaSourceUSDRecord]().ConvertibleTo(reflect.TypeFor[SourceRecord]()) {
		t.Fatal("expected distinct diagnostic type, got implicit official Source conversion")
	}

}

// TestUSASourceUSDIdentity reuses source observations and isolated opaque IDs.
func TestUSASourceUSDIdentity(t *testing.T) {

	a := readUSASourceUSDAdapter(t)
	w := a.world
	if a.source.world.base != nil || a.source.world.namespace != w.sampler.categorical.base.namespace {
		t.Fatal("expected USA owner without Italian base, got surrogate or different namespace")
	}
	g := a.source.world.countries[0].Generator.(usaSourceUSDGenerator)
	if g.world != w || &w.sampler.income.numeric.income[0].laws[0].values[0] !=
		&w.data.socioeconomic.model.numeric.income[0].laws[0].values[0] {
		t.Fatal("expected one authoritative World and shared compiled Income, got duplicated state")
	}
	other, err := newUSASourceUSDAdapter(w, SourceInstanceConfig{
		ID: "other", Version: usaSourceUSDVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateNumerator: 1, DuplicateDenominator: 1,
	})
	if err != nil {
		t.Fatalf("expected second source scope, got %v", err)
	}
	for _, coordinate := range []PersonIndex{1, 2, 36, 20000, Layer1MaxPersonIndex} {
		person, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected USA Person, got %v", err)
		}
		view, err := a.source.world.Person(coordinate)
		if err != nil {
			t.Fatalf("expected existing CountryGenerator view, got %v", err)
		}
		if view.Country != "US" || view.Person != person.Person {
			t.Fatal("expected identical Layer 1 view from the same USA World, got mismatch")
		}
		records, truth, err := a.bind(person)
		if err != nil {
			t.Fatalf("expected bound USA observation, got %v", err)
		}
		ordinary, err := a.source.Records(coordinate)
		if err != nil {
			t.Fatalf("expected existing source-v1 projection, got %v", err)
		}
		expected, err := a.source.world.Oracle(a.source, coordinate)
		if err != nil {
			t.Fatalf("expected existing hidden Oracle, got %v", err)
		}
		if !reflect.DeepEqual(truth, expected) || records[0].Observation.ID != ordinary[0].ID ||
			!reflect.DeepEqual(records[0].Observation.Email, ordinary[0].Email) ||
			!reflect.DeepEqual(records[0].Observation.Phone, ordinary[0].Phone) {
			t.Fatal("expected reused source-v1 IDs, observations and Oracle, got replacement policy")
		}
		duplicates, duplicateTruth, err := other.bind(person)
		if err != nil {
			t.Fatalf("expected independently scoped duplicates, got %v", err)
		}
		if len(duplicates) != 2 || duplicates[0].Observation.ID == duplicates[1].Observation.ID ||
			duplicates[0].Observation.ID == records[0].Observation.ID ||
			duplicateTruth[0].PersonID != truth[0].PersonID || duplicateTruth[1].PersonID != truth[0].PersonID ||
			duplicates[0].Observation.Email == nil && duplicates[1].Observation.Email != nil {
			t.Fatal("expected distinct source IDs and shared hidden Person, got identity contamination")
		}
		for _, record := range append(records, duplicates...) {
			encoded, err := internationalCanonical(record)
			if err != nil {
				t.Fatalf("expected ordinary diagnostic bytes, got %v", err)
			}
			for _, hidden := range []string{person.Person.ID, `"coordinate"`, `"PersonID"`, `"annual_income_eur"`} {
				if bytes.Contains(encoded, []byte(hidden)) {
					t.Fatalf("expected ordinary observation without hidden %s, got leak", hidden)
				}
			}
		}
	}

}

// TestUSASourceUSDRecordReject rejects altered records and invalid USD
// transport.
func TestUSASourceUSDRecordReject(t *testing.T) {

	a := readUSASourceUSDAdapter(t)
	var employee usaSourceUSDRecord
	for coordinate := PersonIndex(1); employee.Employee == nil; coordinate++ {
		person, err := a.world.person(coordinate)
		if err != nil {
			t.Fatalf("expected sampled Person, got %v", err)
		}
		records, _, err := a.bind(person)
		if err != nil {
			t.Fatalf("expected sampled employee, got %v", err)
		}
		employee = records[0]
	}
	original, err := internationalCanonical(employee)
	if err != nil {
		t.Fatalf("expected original USD bytes, got %v", err)
	}
	for _, test := range []struct {
		name   string
		change func(*usaSourceUSDRecord)
	}{
		{"schema", func(r *usaSourceUSDRecord) { r.Schema = "sw-source-v1" }},
		{"Italian-country", func(r *usaSourceUSDRecord) { r.Country = "IT" }},
		{"unknown-country", func(r *usaSourceUSDRecord) { r.Country = "XX" }},
		{"EUR", func(r *usaSourceUSDRecord) { r.Currency = "EUR" }},
		{"profile", func(r *usaSourceUSDRecord) { r.Profile = "it-2026a-socioeconomic-v1" }},
		{"world", func(r *usaSourceUSDRecord) { r.WorldID += "0" }},
		{"source-instance", func(r *usaSourceUSDRecord) { r.SourceInstanceID = "other" }},
		{"source-version", func(r *usaSourceUSDRecord) { r.SourceVersion = "generation-v5" }},
		{"source-protocol", func(r *usaSourceUSDRecord) { r.SourceProtocol = "source-v2" }},
		{"canonical-ID", func(r *usaSourceUSDRecord) { r.Observation.ID = "person_v1_usa-layer1" }},
		{"other-source-ID", func(r *usaSourceUSDRecord) {
			r.Observation.ID = "sr_v2_other_00000000000000000000000000000000"
		}},
		{"tampered-Income", func(r *usaSourceUSDRecord) { *r.AnnualIncomeUSD++ }},
		{"missing-Income", func(r *usaSourceUSDRecord) { r.AnnualIncomeUSD = nil }},
		{"missing-Education", func(r *usaSourceUSDRecord) { r.Education = "" }},
		{"unknown-activity", func(r *usaSourceUSDRecord) { r.PrimaryActivity = "unknown" }},
		{"state", func(r *usaSourceUSDRecord) { r.Observation.State = "IT" }},
		{"ZIP", func(r *usaSourceUSDRecord) { r.Observation.ZIP = "00000" }},
		{"Division", func(r *usaSourceUSDRecord) { r.Observation.Division = "Lazio" }},
		{"locality", func(r *usaSourceUSDRecord) { r.Observation.LocalityID = "unknown" }},
		{"age-overflow-domain", func(r *usaSourceUSDRecord) { r.Observation.CurrentAge = 76 }},
		{"photo", func(r *usaSourceUSDRecord) { photo := "invented"; r.Observation.PhotoID = &photo }},
		{"missing-employee", func(r *usaSourceUSDRecord) { r.Employee = nil }},
		{"both-payloads", func(r *usaSourceUSDRecord) { r.SelfEmployed = &usaSourceUSDSelfEmployed{} }},
		{"nonworking-payload", func(r *usaSourceUSDRecord) { r.PrimaryActivity = PrimaryActivityRetired }},
		{"unknown-family", func(r *usaSourceUSDRecord) { r.Employee.JobFamily = "unknown" }},
		{"unknown-level", func(r *usaSourceUSDRecord) { r.Employee.JobLevel = "unknown" }},
		{"nonexistent-Company", func(r *usaSourceUSDRecord) {
			id, err := entityIDForIndex(a.source.world.namespace, "company", 65537)
			if err != nil {
				t.Fatalf("expected nonmember Company ID fixture, got %v", err)
			}
			r.Employee.EmployerCompanyID = id
		}},
		{"Industry-relation", func(r *usaSourceUSDRecord) {
			r.Employee.Industry = socioeconomicIndustries[(slices.Index(socioeconomicIndustries[:], r.Employee.Industry)+1)%14]
		}},
		{"SizeBand-relation", func(r *usaSourceUSDRecord) { r.Employee.SizeBand = CompanySizeBandZero }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var changed usaSourceUSDRecord
			err := json.Unmarshal(original, &changed)
			if err != nil {
				t.Fatalf("expected independent negative fixture, got %v", err)
			}
			test.change(&changed)
			encoded, err := json.Marshal(changed)
			if err != nil {
				t.Fatalf("expected negative JSON fixture, got %v", err)
			}
			decoded, err := a.decode(t.Context(), encoded, employee)
			if err != nil {
				if !reflect.DeepEqual(decoded, usaSourceUSDRecord{}) {
					t.Fatal("expected empty rejected record, got partial record")
				}
				return
			}
			t.Fatal("expected altered USD source record rejection, got admission")
		})
	}
	for _, value := range []string{"-1", "4294967296", "1.0", "1e2", `"0"`, "null"} {
		t.Run("integer-"+value, func(t *testing.T) {
			var changed map[string]json.RawMessage
			err := json.Unmarshal(original, &changed)
			if err != nil {
				t.Fatalf("expected original record object, got %v", err)
			}
			changed["annualIncomeUSD"] = json.RawMessage(value)
			encoded, err := json.Marshal(changed)
			if err != nil {
				t.Fatalf("expected invalid numeric fixture, got %v", err)
			}
			_, err = a.decode(t.Context(), encoded, employee)
			if err != nil {
				return
			}
			t.Fatal("expected exact uint32 Income rejection, got admission")
		})
	}
	for _, field := range []string{"annual_income_eur", "annualIncomeEUR", "coordinate", "PersonID", "release"} {
		t.Run("forbidden-"+field, func(t *testing.T) {
			encoded := append([]byte(fmt.Sprintf(`{"%s":1,`, field)), original[1:]...)
			_, err := a.decode(t.Context(), encoded, employee)
			if err != nil {
				return
			}
			t.Fatal("expected closed diagnostic schema rejection, got admission")
		})
	}
	for _, name := range []string{"missing", "duplicate", "oversized", "trailing", "null-payload"} {
		t.Run(name, func(t *testing.T) {
			encoded := original
			switch name {
			case "missing":
				encoded = bytes.Replace(original,
					[]byte(fmt.Sprintf(`"annualIncomeUSD":%d,`, *employee.AnnualIncomeUSD)), nil, 1)
			case "duplicate":
				encoded = append([]byte(`{"currency":"USD",`), original[1:]...)
			case "oversized":
				encoded = bytes.Repeat([]byte(" "), usaSourceUSDRecordBytes+1)
			case "trailing":
				encoded = append(bytes.Clone(original), []byte("{}")...)
			case "null-payload":
				encoded = append([]byte(`{"selfEmployed":null,`), original[1:]...)
			}
			_, err := a.decode(t.Context(), encoded, employee)
			if err != nil {
				return
			}
			t.Fatal("expected malformed diagnostic transport rejection, got admission")
		})
	}

}

// TestUSASourceUSDRoundTrip preserves typed facts and branch applicability.
func TestUSASourceUSDRoundTrip(t *testing.T) {

	a := readUSASourceUSDAdapter(t)
	activities := map[PrimaryActivity]bool{}
	zero := false
	var employee usaSourceUSDRecord
	for coordinate := PersonIndex(1); coordinate <= 512; coordinate++ {
		person, err := a.world.person(coordinate)
		if err != nil {
			t.Fatalf("expected sampled USA Person, got %v", err)
		}
		records, truth, err := a.bind(person)
		if err != nil {
			t.Fatalf("expected complete USA projection, got %v", err)
		}
		record := records[0]
		encoded, err := serializeUSASourceUSDDiagnostic(t.Context(), a, record)
		if err != nil {
			t.Fatalf("expected exact USD round-trip at %d, got %v", coordinate, err)
		}
		if truth[0].PersonID != person.Person.ID || record.AnnualIncomeUSD == nil ||
			*record.AnnualIncomeUSD != person.AnnualIncomeUSD || record.Education != person.Education ||
			record.PrimaryActivity != person.PrimaryActivity || record.Observation.Division != person.Division ||
			record.Observation.ZIP != person.Person.Address.PostalCode {
			t.Fatalf("expected exact original Person and USD facts at %d, got mismatch", coordinate)
		}
		activities[record.PrimaryActivity] = true
		zero = zero || *record.AnnualIncomeUSD == 0
		switch record.PrimaryActivity {
		case PrimaryActivityEmployee:
			e := record.Employee
			if e == nil || record.SelfEmployed != nil || e.EmployerCompanyID != person.EmployerCompany.ID ||
				e.Industry != person.EmployerCompany.Industry || e.SizeBand != person.EmployerCompany.SizeBand ||
				e.JobFamily != person.Employee.JobFamily || e.JobLevel != person.Employee.JobLevel {
				t.Fatal("expected Company-derived employee projection, got mismatch")
			}
			employee = record
		case PrimaryActivitySelfEmployed:
			if record.Employee != nil || record.SelfEmployed == nil ||
				record.SelfEmployed.JobFamily != person.SelfEmployed.JobFamily ||
				record.SelfEmployed.Industry != person.SelfEmployed.Industry ||
				bytes.Contains(encoded, []byte(`"jobLevel"`)) || bytes.Contains(encoded, []byte(`"employerCompanyID"`)) {
				t.Fatal("expected only applicable self-employed fields, got professional placeholders")
			}
		default:
			if record.Employee != nil || record.SelfEmployed != nil || bytes.Contains(encoded, []byte(`"employee"`)) ||
				bytes.Contains(encoded, []byte(`"selfEmployed"`)) || bytes.Contains(encoded, []byte(`"jobFamily"`)) {
				t.Fatal("expected absent nonworking occupational fields, got payload")
			}
		}
	}
	if len(activities) != 6 || !zero {
		t.Fatalf("expected six activities and present zero USD, got %d activities and zero=%t", len(activities), zero)
	}

	// Disable sampling after binding: transport can only read Company metadata.
	a.world.sampler.income = nil
	a.world.sampler.categorical.companies.model = nil
	a.world.sampler.categorical.companies.sampler.model = nil
	a.world.sampler.categorical.companies.sampler.employers = [40]socioeconomicEmployerIndex{}
	_, err := serializeUSASourceUSDDiagnostic(t.Context(), a, employee)
	if err != nil {
		t.Fatalf("expected serialization without Employer or Income sampler, got %v", err)
	}

}

func readUSASourceUSDAdapter(t *testing.T) *usaSourceUSDAdapter {

	t.Helper()
	world, err := newUSACountryWorld(readUSACountryData(t), IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected complete USA World, got %v", err)
	}
	adapter, err := newUSASourceUSDAdapter(world, SourceInstanceConfig{
		ID: "usa-usd-diagnostic", Version: usaSourceUSDVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateDenominator: 1,
	})
	if err != nil {
		t.Fatalf("expected USA source adapter, got %v", err)
	}

	return adapter
}

// serializeUSASourceUSDDiagnostic represents an existing logical observation
// and proves exact closed decoding; it never calls any population sampler.
func serializeUSASourceUSDDiagnostic(ctx context.Context, source *usaSourceUSDAdapter, record usaSourceUSDRecord) ([]byte, error) {

	err := source.validate(record)
	if err != nil {
		return nil, err
	}
	encoded, err := internationalCanonical(record)
	if err != nil {
		return nil, err
	}
	decoded, err := source.decode(ctx, encoded, record)
	if err != nil {
		return nil, err
	}
	reencoded, err := internationalCanonical(decoded)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(encoded, reencoded) {
		return nil, fmt.Errorf("USA source round-trip changed canonical bytes")
	}

	return encoded, nil
}

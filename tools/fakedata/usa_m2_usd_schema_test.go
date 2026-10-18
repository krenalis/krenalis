// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"
)

// TestUSAM2USDCandidateIsolation keeps both diagnostic and candidate documents
// outside the official World, Source and checkpoint contracts.
func TestUSAM2USDCandidateIsolation(t *testing.T) {

	a := readUSASourceUSDAdapter(t)
	c, err := newUSAM2USDCodec(a)
	if err != nil {
		t.Fatalf("expected prepared candidate codec, got %v", err)
	}
	record := readUSAM2USDRecords(t)[0]
	encoded, err := c.encode(record)
	if err != nil {
		t.Fatalf("expected candidate bytes, got %v", err)
	}
	t.Run("diagnostic-is-not-candidate", func(t *testing.T) {
		diagnostic, err := internationalCanonical(record)
		if err != nil {
			t.Fatalf("expected diagnostic bytes, got %v", err)
		}
		_, err = c.decode(t.Context(), diagnostic)
		if err != nil {
			return
		}
		t.Fatal("expected explicit candidate envelope requirement, got diagnostic admission")
	})
	t.Run("candidate-is-not-diagnostic", func(t *testing.T) {
		_, err := a.decode(t.Context(), encoded, record)
		if err != nil {
			return
		}
		t.Fatal("expected separate diagnostic contract, got candidate admission")
	})
	t.Run("missing-photos", func(t *testing.T) {
		_, err := NewSourceWorld(nil, []CountryShare{{
			Code: "US", Version: usaSourceUSDVersion, Weight: 1, Generator: usaSourceUSDGenerator{a.world},
		}})
		if err != nil {
			return
		}
		t.Fatal("expected official World rejection without photo snapshot, got admission")
	})
	if reflect.TypeFor[usaM2USDDocument]().ConvertibleTo(reflect.TypeFor[SourceRecord]()) ||
		reflect.TypeFor[usaM2USDDocument]().ConvertibleTo(reflect.TypeFor[usaSourceUSDRecord]()) {
		t.Fatal("expected distinct candidate envelope, got implicit official or diagnostic conversion")
	}
	for _, adapter := range []*usaSourceUSDAdapter{nil, {}, {world: &usaCountryWorld{}}} {
		_, err := newUSAM2USDCodec(adapter)
		if err != nil {
			continue
		}
		t.Fatal("expected incomplete provenance rejection, got codec")
	}
	var uninitialized *usaM2USDCodec
	_, err = uninitialized.encode(record)
	if err != nil {
		return
	}

	t.Fatal("expected uninitialized encoder rejection, got bytes")
}

// TestUSAM2USDCandidateReject checks domains independently of replay equality.
func TestUSAM2USDCandidateReject(t *testing.T) {

	a := readUSASourceUSDAdapter(t)
	c, err := newUSAM2USDCodec(a)
	if err != nil {
		t.Fatalf("expected prepared candidate codec, got %v", err)
	}
	records := readUSAM2USDRecords(t)
	i := slices.IndexFunc(records, func(r usaSourceUSDRecord) bool { return r.Employee != nil })
	original, err := c.encode(records[i])
	if err != nil {
		t.Fatalf("expected employee candidate fixture, got %v", err)
	}
	for _, test := range []struct {
		name   string
		change func(*usaM2USDDocument)
	}{
		{"schema", func(d *usaM2USDDocument) { d.Schema = usaSourceUSDSchema }},
		{"version", func(d *usaM2USDDocument) { d.DocumentVersion++ }},
		{"base-world", func(d *usaM2USDDocument) { d.Provenance.BaseWorldID += "0" }},
		{"base-data", func(d *usaM2USDDocument) { d.Provenance.BaseDataDigest += "0" }},
		{"phone-rule", func(d *usaM2USDDocument) { d.Provenance.PhoneRuleDigest += "0" }},
		{"model", func(d *usaM2USDDocument) { d.Provenance.ModelID += "0" }},
		{"socioeconomic-data", func(d *usaM2USDDocument) { d.Provenance.SocioeconomicDataDigest += "0" }},
		{"source-assets", func(d *usaM2USDDocument) { d.Provenance.SourceAssetsSHA256 += "0" }},
		{"anchor", func(d *usaM2USDDocument) { d.Provenance.AnchorDate = "2027-01-01" }},
		{"generation", func(d *usaM2USDDocument) { d.Provenance.GenerationProtocol = "generation-v5" }},
		{"IT", func(d *usaM2USDDocument) { d.SourceRecord.Country = "IT" }},
		{"unknown-country", func(d *usaM2USDDocument) { d.SourceRecord.Country = "XX" }},
		{"EUR", func(d *usaM2USDDocument) { d.SourceRecord.Currency = "EUR" }},
		{"unknown-currency", func(d *usaM2USDDocument) { d.SourceRecord.Currency = "XXX" }},
		{"source-schema", func(d *usaM2USDDocument) { d.SourceRecord.Schema = usaM2USDSchema }},
		{"source-instance", func(d *usaM2USDDocument) { d.SourceRecord.SourceInstanceID = "other" }},
		{"source-world", func(d *usaM2USDDocument) { d.SourceRecord.WorldID += "0" }},
		{"source-record", func(d *usaM2USDDocument) { d.SourceRecord.Observation.ID = "other" }},
		{"income-absent", func(d *usaM2USDDocument) { d.SourceRecord.AnnualIncomeUSD = nil }},
		{"education", func(d *usaM2USDDocument) { d.SourceRecord.Education = "unknown" }},
		{"activity", func(d *usaM2USDDocument) { d.SourceRecord.PrimaryActivity = "unknown" }},
		{"employee-absent", func(d *usaM2USDDocument) { d.SourceRecord.Employee = nil }},
		{"both-payloads", func(d *usaM2USDDocument) { d.SourceRecord.SelfEmployed = &usaSourceUSDSelfEmployed{} }},
		{"nonworking-payload", func(d *usaM2USDDocument) { d.SourceRecord.PrimaryActivity = PrimaryActivityInactive }},
		{"nonexistent-company", func(d *usaM2USDDocument) {
			id, err := entityIDForIndex(a.source.world.namespace, "company", 65537)
			if err != nil {
				t.Fatalf("expected out-of-roster Company ID, got %v", err)
			}
			d.SourceRecord.Employee.EmployerCompanyID = id
		}},
		{"industry-relation", func(d *usaM2USDDocument) {
			industry := &d.SourceRecord.Employee.Industry
			*industry = socioeconomicIndustries[(slices.Index(socioeconomicIndustries[:], *industry)+1)%14]
		}},
		{"size-relation", func(d *usaM2USDDocument) { d.SourceRecord.Employee.SizeBand = CompanySizeBandZero }},
		{"birth-date", func(d *usaM2USDDocument) { d.SourceRecord.Observation.BirthDate = "not-a-date" }},
		{"text-bound", func(d *usaM2USDDocument) {
			d.SourceRecord.Observation.Street = string(bytes.Repeat([]byte{'x'}, 1025))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			document, err := c.decode(t.Context(), original)
			if err != nil {
				t.Fatalf("expected independent valid candidate, got %v", err)
			}
			test.change(&document)
			encoded, err := internationalCanonical(document)
			if err != nil {
				t.Fatalf("expected negative fixture encoding, got %v", err)
			}
			decoded, err := c.decode(t.Context(), encoded)
			if err != nil {
				if !reflect.DeepEqual(decoded, usaM2USDDocument{}) {
					t.Fatal("expected empty rejected candidate, got partial state")
				}
				if !reflect.DeepEqual(document.SourceRecord, records[i]) {
					_, err = c.encode(document.SourceRecord)
					if err != nil {
						return
					}
					t.Fatal("expected invalid logical record rejection by encoder, got bytes")
				}
				return
			}
			t.Fatal("expected candidate domain rejection, got admission")
		})
	}

}

// TestUSAM2USDCandidateRoundTrip preserves existing facts without any sampler.
func TestUSAM2USDCandidateRoundTrip(t *testing.T) {

	a := readUSASourceUSDAdapter(t)
	c, err := newUSAM2USDCodec(a)
	if err != nil {
		t.Fatalf("expected prepared candidate codec, got %v", err)
	}
	records := readUSAM2USDRecords(t)
	// Remove Person, categorical, Employer and monetary sampling capabilities.
	a.world.sampler.income = nil
	a.world.sampler.categorical.base = nil
	a.world.sampler.categorical.model = nil
	a.world.sampler.categorical.companies.model = nil
	a.world.sampler.categorical.companies.sampler.model = nil
	a.world.sampler.categorical.companies.sampler.base = nil
	a.world.sampler.categorical.companies.sampler.employers = [40]socioeconomicEmployerIndex{}
	a.world.data.socioeconomic.model = nil
	a.source.world.countries = nil
	activities := map[PrimaryActivity]bool{}
	hasZero := false
	for _, record := range records {
		encoded, err := c.encode(record)
		if err != nil {
			t.Fatalf("expected candidate encoding without samplers, got %v", err)
		}
		decoded, err := c.decode(t.Context(), encoded)
		if err != nil {
			t.Fatalf("expected independent candidate decoding, got %v", err)
		}
		if !reflect.DeepEqual(decoded.SourceRecord, record) || decoded.Provenance != c.provenance {
			t.Fatal("expected exact original USD facts and provenance, got mismatch")
		}
		reencoded, err := c.encode(decoded.SourceRecord)
		if err != nil {
			t.Fatalf("expected stable candidate reencoding, got %v", err)
		}
		if !bytes.Equal(encoded, reencoded) || bytes.Contains(encoded, []byte("annual_income_eur")) ||
			bytes.Contains(encoded, []byte("annualIncomeEUR")) {
			t.Fatal("expected identical US/USD-only bytes, got mismatch")
		}
		if record.PrimaryActivity != PrimaryActivityEmployee && record.PrimaryActivity != PrimaryActivitySelfEmployed &&
			(bytes.Contains(encoded, []byte(`"employee"`)) || bytes.Contains(encoded, []byte(`"selfEmployed"`))) {
			t.Fatal("expected no nonworking professional payload, got placeholder")
		}
		activities[record.PrimaryActivity] = true
		hasZero = hasZero || *record.AnnualIncomeUSD == 0
		t.Logf("M2_FIXTURE %s", encoded)
	}
	if len(activities) != 6 || !hasZero {
		t.Fatalf("expected six branches and present zero USD, got %d branches and zero=%t", len(activities), hasZero)
	}
	for _, value := range []annualIncomeUSD{0, 1, 16777217, 4294967295} {
		record := records[0]
		record.AnnualIncomeUSD = &value
		encoded, err := c.encode(record)
		if err != nil {
			t.Fatalf("expected representable integer %d, got %v", value, err)
		}
		decoded, err := c.decode(t.Context(), encoded)
		if err != nil {
			t.Fatalf("expected exact integer decoding, got %v", err)
		}
		if *decoded.SourceRecord.AnnualIncomeUSD != value ||
			!bytes.Contains(encoded, fmt.Appendf(nil, `"annualIncomeUSD":%d`, value)) {
			t.Fatalf("expected exact whole USD %d, got %d", value, *decoded.SourceRecord.AnnualIncomeUSD)
		}
	}

}

// TestUSAM2USDCandidateTransport rejects malformed closed JSON at every level.
func TestUSAM2USDCandidateTransport(t *testing.T) {

	c, err := newUSAM2USDCodec(readUSASourceUSDAdapter(t))
	if err != nil {
		t.Fatalf("expected prepared candidate codec, got %v", err)
	}
	record := readUSAM2USDRecords(t)[0]
	original, err := c.encode(record)
	if err != nil {
		t.Fatalf("expected original candidate bytes, got %v", err)
	}
	income := fmt.Sprintf(`"annualIncomeUSD":%d`, *record.AnnualIncomeUSD)
	for _, value := range []string{"-1", "4294967296", "1.0", "1e2", `"0"`, "null", "true"} {
		t.Run("integer-"+value, func(t *testing.T) {
			encoded := bytes.Replace(original, []byte(income), []byte(`"annualIncomeUSD":`+value), 1)
			_, err := c.decode(t.Context(), encoded)
			if err != nil {
				return
			}
			t.Fatal("expected exact uint32 token rejection, got admission")
		})
	}
	for _, test := range []struct {
		name string
		from string
		to   string
	}{
		{"missing-income", income + ",", ""},
		{"unknown-root", `{"documentVersion":`, `{"unknown":0,"documentVersion":`},
		{"duplicate-root", `{"documentVersion":`, `{"documentVersion":1,"documentVersion":`},
		{"unknown-provenance", `"provenance":{`, `"provenance":{"unknown":0,`},
		{"duplicate-provenance", `"provenance":{`, `"provenance":{"anchorDate":"2026-01-01",`},
		{"EUR-snake", `"sourceRecord":{`, `"sourceRecord":{"annual_income_eur":0,`},
		{"EUR-camel", `"sourceRecord":{`, `"sourceRecord":{"annualIncomeEUR":0,`},
		{"USD-snake-unadopted", `"sourceRecord":{`, `"sourceRecord":{"annual_income_usd":0,`},
		{"duplicate-source", `"sourceRecord":{`, `"sourceRecord":{"currency":"USD",`},
		{"escaped-duplicate", `"sourceRecord":{`, `"sourceRecord":{"\u0063urrency":"USD",`},
		{"case-alias", `"sourceRecord":{`, `"sourceRecord":{"Country":"US",`},
		{"unknown-observation", `"observation":{`, `"observation":{"coordinate":1,`},
		{"duplicate-observation", `"observation":{`, `"observation":{"gender":"female",`},
		{"null-optional", `"observation":{`, `"observation":{"photoID":null,`},
		{"null-payload", `"sourceRecord":{`, `"sourceRecord":{"employee":null,`},
		{"missing-version", `"documentVersion":1,`, ""},
		{"null-version", `"documentVersion":1`, `"documentVersion":null`},
		{"fractional-version", `"documentVersion":1`, `"documentVersion":1.0`},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded := bytes.Replace(original, []byte(test.from), []byte(test.to), 1)
			if bytes.Equal(encoded, original) {
				t.Fatal("expected changed transport fixture, got unchanged bytes")
			}
			_, err := c.decode(t.Context(), encoded)
			if err != nil {
				return
			}
			t.Fatal("expected closed candidate transport rejection, got admission")
		})
	}
	for _, encoded := range [][]byte{nil, []byte("null"), []byte("[]"),
		bytes.Repeat([]byte(" "), usaM2USDDocumentBytes+1), append(bytes.Clone(original), []byte("{}")...)} {
		_, err := c.decode(t.Context(), encoded)
		if err != nil {
			continue
		}
		t.Fatal("expected bounded single candidate object, got admission")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = c.decode(ctx, original)
	if err != nil {
		return
	}

	t.Fatal("expected canceled candidate decode failure, got admission")
}

func readUSAM2USDRecords(t *testing.T) []usaSourceUSDRecord {

	t.Helper()
	path := "testdata/usa-source-usd-base-v2.jsonl"
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("expected existing diagnostic observations, got %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(nil, usaSourceUSDRecordBytes)
	var records []usaSourceUSDRecord
	for scanner.Scan() {
		if len(records) == 7 {
			t.Fatal("expected at most seven existing observations, got excess fixtures")
		}
		var record usaSourceUSDRecord
		err = decodeInternationalJSON(t.Context(), bytes.NewReader(scanner.Bytes()), &record)
		if err != nil {
			t.Fatalf("expected closed diagnostic fixture, got %v", err)
		}
		records = append(records, record)
	}
	err = scanner.Err()
	if err != nil {
		t.Fatalf("expected complete fixture read, got %v", err)
	}
	if len(records) != 7 {
		t.Fatalf("expected seven existing observations, got %d", len(records))
	}

	return records
}

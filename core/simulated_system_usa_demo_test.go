// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/simulated"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

// TestSimulatedSystemUSA checks candidate persistence, photos and isolated resume.
func TestSimulatedSystemUSA(t *testing.T) {

	data, err := simulatedUSATestData(t)
	if err != nil {
		t.Fatalf("expected verified USA demo data, got %v", err)
	}
	workspaceID := "123456789ABC"
	photoBaseURL := "http://localhost/v1/simulated/usa/photos/"
	account := &state.SimulatedSystem{ID: "23456789ABCD", UserCount: 512,
		Countries: json.Value(`{"US":100}`), DuplicateRecordPercent: decimal.MustInt(50),
		GenerationPolicyVersion: simulated.USAGenerationPolicy}
	plan, cursor, err := newUSASimulatedSystemPlan(workspaceID, account, data, photoBaseURL)
	if err != nil {
		t.Fatalf("expected USA generation plan, got %v", err)
	}
	if cursor.MarketDataVersion != "us-2026a-layer1-v2-candidate" || cursor.USA == nil ||
		cursor.USA.Country != "US" || cursor.USA.Currency != "USD" {
		t.Fatalf("expected country-aware BASE-v2 checkpoint, got %+v", cursor)
	}
	decode := func(id string, encoded json.Value) simulatedSystemUSADemoDocument {
		t.Helper()
		attributes, err := plan.usa.Decode(t.Context(), id, encoded)
		if err != nil {
			t.Fatalf("expected decoded M2 application document, got %v", err)
		}
		projection, err := json.Marshal(attributes)
		if err != nil {
			t.Fatalf("expected USA Source projection, got %v", err)
		}
		var document simulatedSystemUSADemoDocument
		err = json.Unmarshal(projection, &document)
		if err != nil {
			t.Fatalf("expected typed USA Source projection, got %v", err)
		}
		return document
	}
	var first []byte
	for slot := 0; slot < 256; slot++ {
		a, err := plan.record(slot, 0)
		if err != nil {
			t.Fatalf("expected USA record at %d, got %v", slot, err)
		}
		b, err := plan.record(slot, 1)
		if err != nil {
			t.Fatalf("expected duplicate USA record at %d, got %v", slot, err)
		}
		x := decode(a.ID, a.Data)
		y := decode(b.ID, b.Data)
		if x.Country != "US" || x.Currency != "USD" || x.PhotoID == "" || x.Photo == nil ||
			x.PhotoID != y.PhotoID || x.AnnualIncomeUSD == nil || x.CustomerID == y.CustomerID ||
			bytes.Contains(a.Data, []byte("annual_income_eur")) ||
			!bytes.Contains(a.Data, []byte("http://localhost/v1/simulated/usa/photos/"+workspaceID+"/face-")) {
			t.Fatalf("expected photographed USD documents with opaque source IDs at %d, got %s", slot, a.Data)
		}
		for _, name := range []*string{x.FirstName, x.LastName, y.FirstName, y.LastName} {
			if name != nil && *name == strings.ToUpper(*name) {
				t.Fatalf("expected natural names in generated JSON, got %s", a.Data)
			}
		}
		if slot == 0 {
			first = bytes.Clone(b.Data)
		}
	}
	cursor = plan.advance(cursor)
	account.GeneratedRecordCount = 1
	account.GenerationCheckpoint, err = json.Marshal(cursor)
	if err != nil {
		t.Fatalf("expected USA duplicate-boundary checkpoint, got %v", err)
	}
	resumed, restored, err := newUSASimulatedSystemPlan(workspaceID, account, data, photoBaseURL)
	if err != nil {
		t.Fatalf("expected USA checkpoint resume, got %v", err)
	}
	record, err := resumed.record(restored.Person, restored.Representation)
	if err != nil {
		t.Fatalf("expected resumed photographed record, got %v", err)
	}
	if !bytes.Equal(record.Data, first) {
		t.Fatal("expected byte-identical resumed USA observation, got drift")
	}
	for _, field := range []string{
		"usa", "catalogSHA256", "catalogRelease", "marketDataVersion", "sourceVersion",
	} {
		t.Run(field, func(t *testing.T) {
			changed := *account
			var fields map[string]any
			err := json.Unmarshal(account.GenerationCheckpoint, &fields)
			if err != nil {
				t.Fatalf("expected checkpoint fields, got %v", err)
			}
			fields[field] = "changed"
			changed.GenerationCheckpoint, err = json.Marshal(fields)
			if err != nil {
				t.Fatalf("expected changed checkpoint, got %v", err)
			}
			_, _, err = newUSASimulatedSystemPlan(workspaceID, &changed, data, photoBaseURL)
			if err != nil {
				return
			}
			t.Fatal("expected changed USA checkpoint rejection, got admission")
		})
	}
	_, _, err = newSimulatedSystemPlan(workspaceID, account, nil)
	if err != nil {
		return
	}

	t.Fatal("expected USA policy rejection by Italian V5 planner, got admission")

}

// TestSimulatedSystemUSACheckpoint rejects changes to the country, assets,
// source scope, generation configuration and durable cursor.
func TestSimulatedSystemUSACheckpoint(t *testing.T) {

	data, err := simulatedUSATestData(t)
	if err != nil {
		t.Fatalf("expected verified USA data, got %v", err)
	}
	const workspaceID = "123456789ABC"
	const photoURL = "http://localhost/v1/simulated/usa/photos/"
	account := &state.SimulatedSystem{ID: "23456789ABCD", UserCount: 12,
		Countries: json.Value(`{"US":100}`), DuplicateRecordPercent: decimal.MustInt(50),
		GenerationPolicyVersion: simulated.USAGenerationPolicy}
	_, initial, err := newUSASimulatedSystemPlan(workspaceID, account, data, photoURL)
	if err != nil {
		t.Fatalf("expected initial USA checkpoint, got %v", err)
	}
	account.GenerationCheckpoint, err = json.Marshal(initial)
	if err != nil {
		t.Fatalf("expected USA checkpoint JSON, got %v", err)
	}
	// PostgreSQL reloads numeric(5,2) with a fixed scale.
	reloaded := *account
	reloaded.DuplicateRecordPercent = decimal.MustParse("50.00")
	_, _, err = newUSASimulatedSystemPlan(workspaceID, &reloaded, data, photoURL)
	if err != nil {
		t.Fatalf("expected identical binding after PostgreSQL decimal reload, got %v", err)
	}
	for _, test := range []struct {
		name   string
		change func(*simulatedSystemCheckpoint)
		config func(*state.SimulatedSystem)
	}{
		{name: "country", change: func(c *simulatedSystemCheckpoint) { c.USA.Country = "IT" }},
		{name: "currency", change: func(c *simulatedSystemCheckpoint) { c.USA.Currency = "EUR" }},
		{name: "assets", change: func(c *simulatedSystemCheckpoint) { c.USA.AssetsSHA256 = "changed" }},
		{name: "document", change: func(c *simulatedSystemCheckpoint) { c.USA.DocumentSHA256 = "changed" }},
		{name: "configuration", change: func(c *simulatedSystemCheckpoint) { c.USA.ConfigSHA256 = "changed" }},
		{name: "absent country binding", change: func(c *simulatedSystemCheckpoint) { c.USA = nil }},
		{name: "legacy demo", change: func(c *simulatedSystemCheckpoint) { c.USADemoSHA256 = "legacy" }},
		{name: "person past end", change: func(c *simulatedSystemCheckpoint) { c.Person = 7 }},
		{name: "negative person", change: func(c *simulatedSystemCheckpoint) { c.Person = -1 }},
		{name: "third representation", change: func(c *simulatedSystemCheckpoint) { c.Representation = 2 }},
		{name: "invalid duplicate boundary", change: func(c *simulatedSystemCheckpoint) {
			c.Person, c.Representation = 6, 1
		}},
		{name: "counter ahead", config: func(a *state.SimulatedSystem) { a.GeneratedRecordCount = 1 }},
		{name: "count changed", config: func(a *state.SimulatedSystem) { a.UserCount++ }},
		{name: "duplicate changed", config: func(a *state.SimulatedSystem) { a.DuplicateRecordPercent = decimal.MustInt(25) }},
		{name: "weight changed", config: func(a *state.SimulatedSystem) { a.Countries = json.Value(`{"US":1}`) }},
		{name: "Italy", config: func(a *state.SimulatedSystem) { a.Countries = json.Value(`{"IT":100}`) }},
		{name: "system changed", config: func(a *state.SimulatedSystem) { a.ID = "3456789ABCDE" }},
		{name: "policy changed", config: func(a *state.SimulatedSystem) { a.GenerationPolicyVersion = simulatedSystemUSADemoPolicyVersion }},
	} {
		t.Run(test.name, func(t *testing.T) {

			changed := *account
			checkpoint := initial
			binding := *initial.USA
			checkpoint.USA = &binding
			if test.change != nil {
				test.change(&checkpoint)
			}
			changed.GenerationCheckpoint, err = json.Marshal(checkpoint)
			if err != nil {
				t.Fatalf("expected modified checkpoint, got %v", err)
			}
			if test.config != nil {
				test.config(&changed)
			}
			_, _, err := newUSASimulatedSystemPlan(workspaceID, &changed, data, photoURL)
			if err != nil {
				return
			}

			t.Fatal("expected incompatible USA checkpoint rejection, got admission")

		})
	}
	for _, test := range []struct {
		name string
		data json.Value
	}{
		{"missing binding", json.Value(`{}`)},
		{"null", json.Value(`null`)},
		{"unknown field", append([]byte(`{"unexpected":1,`), account.GenerationCheckpoint[1:]...)},
		{"duplicate field", append([]byte(`{"person":0,`), account.GenerationCheckpoint[1:]...)},
		{"oversize", bytes.Repeat([]byte(" "), 8193)},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := *account
			changed.GenerationCheckpoint = test.data
			_, _, err := newUSASimulatedSystemPlan(workspaceID, &changed, data, photoURL)
			if err != nil {
				return
			}
			t.Fatal("expected malformed USA checkpoint rejection, got admission")
		})
	}

	for _, workspace := range []string{"3456789ABCDE", workspaceID} {
		t.Run(workspace, func(t *testing.T) {
			url := photoURL
			if workspace == workspaceID {
				url = "http://changed.test/v1/simulated/usa/photos/"
			}
			_, _, err := newUSASimulatedSystemPlan(workspace, account, data, url)
			if err != nil {
				return
			}
			t.Fatal("expected changed workspace or photo URL rejection, got admission")
		})
	}

}

func simulatedUSATestData(t *testing.T) (*fakedata.USAWorldData, error) {
	t.Helper()
	origin := httptest.NewServer(http.FileServer(http.Dir("../tools/fakedata/testdata/published")))
	t.Cleanup(origin.Close)
	source, err := fakedata.NewUSAWorldSource([]string{origin.URL + "/"})
	if err != nil {
		return nil, err
	}
	return source.Load(t.Context())
}

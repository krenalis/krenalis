// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestSimulatedSystemM2USDCandidateIsolation rejects complete candidate and
// diagnostic observations at the unchanged public checkpoint boundary.
func TestSimulatedSystemM2USDCandidateIsolation(t *testing.T) {

	data := simulatedSystemTestData(t)
	for _, name := range []string{"valid-usd-employee.json", "diagnostic-employee.json"} {
		t.Run(name, func(t *testing.T) {
			path := "testdata/simulated-us/" + name
			encoded, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("expected complete USD boundary fixture, got %v", err)
			}
			account := simulatedSystemPlanFixture(t, data, "account-a", 512, "50")
			account.GenerationCheckpoint = json.Value(encoded)
			_, _, err = newSimulatedSystemPlan("workspace-a", account, data)
			if err != nil {
				return
			}
			t.Fatal("expected USD candidate or diagnostic checkpoint rejection, got admission")
		})
	}
	if _, exists := reflect.TypeFor[simulatedSystemDocument]().FieldByName("AnnualIncomeUSD"); exists {
		t.Fatal("expected unchanged Italian document fields, got automatic USD extension")
	}
	income := simulatedSystemDocument{}
	encoded := []byte(`{"customer_id":"it-fixture","annual_income_eur":0}`)
	err := json.Unmarshal(encoded, &income)
	if err != nil {
		t.Fatalf("expected existing Italian EUR fixture, got %v", err)
	}
	original, err := json.Marshal(income)
	if err != nil {
		t.Fatalf("expected original Italian bytes, got %v", err)
	}
	// Existing document decoding may ignore unknown fields; it cannot acquire
	// a USD value or reinterpret it as the EUR field through the new codec.
	injected := append([]byte(`{"annual_income_usd":4294967295,`), encoded[1:]...)
	err = json.Unmarshal(injected, &income)
	if err != nil {
		t.Fatalf("expected existing Italian decoder behavior, got %v", err)
	}
	actual, err := json.Marshal(income)
	if err != nil {
		t.Fatalf("expected unchanged Italian reencoding, got %v", err)
	}
	if !bytes.Equal(actual, original) || income.AnnualIncomeEUR == nil || *income.AnnualIncomeEUR != 0 {
		t.Fatalf("expected unchanged EUR zero and bytes %s, got %s", original, actual)
	}

}

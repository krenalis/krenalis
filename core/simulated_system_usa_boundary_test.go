// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestSimulatedSystemItalyUSDIsolation compares V5 documents and checkpoints
// with bytes measured using the Source implementation before the USD adapter.
func TestSimulatedSystemItalyUSDIsolation(t *testing.T) {

	data := simulatedSystemTestData(t)
	account := simulatedSystemPlanFixture(t, data, "account-a", 512, "50")
	plan, checkpoint, err := newSimulatedSystemPlan("workspace-a", account, data)
	if err != nil {
		t.Fatalf("expected Italian V5 plan, got %v", err)
	}
	documents := sha256.New()
	for slot := range plan.people {
		for representation := range 2 {
			record, err := plan.record(slot, representation)
			if err != nil {
				t.Fatalf("expected Italian V5 document, got %v", err)
			}
			if bytes.Contains(record.Data, []byte("USD")) || bytes.Contains(record.Data, []byte("annualIncomeUSD")) ||
				bytes.Contains(record.Data, []byte("annual_income_usd")) ||
				!bytes.Contains(record.Data, []byte("annual_income_eur")) {
				t.Fatal("expected only existing EUR projection, got altered Italian currency fields")
			}
			documents.Write(record.Data)
		}
	}
	encoded, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatalf("expected Italian checkpoint bytes, got %v", err)
	}
	t.Logf("V5 documents SHA256=%x checkpoint SHA256=%x", documents.Sum(nil), sha256.Sum256(encoded))
	if fmt.Sprintf("%x", documents.Sum(nil)) != "4a2561acecf8fcc5261be0c6544370d60b05e5ca10afaa30ee41e67d076a7c88" ||
		fmt.Sprintf("%x", sha256.Sum256(encoded)) != "9c54b8fefbc715cf3a90598cc9f6b0feffb4c5f56660c4e987caa032626e433c" {
		t.Fatal("expected byte-identical Italian V5 documents and checkpoint, got digest mismatch")
	}

	// A diagnostic envelope has no V5 checkpoint identity and cannot resume.
	account.GenerationCheckpoint = json.Value(`{"schema":"usa-internal-source-usd-diagnostic-v1",` +
		`"country":"US","currency":"USD","annualIncomeUSD":0}`)
	_, _, err = newSimulatedSystemPlan("workspace-a", account, data)
	if err != nil {
		return
	}

	t.Fatalf("expected diagnostic record rejection as V5 checkpoint, got admission for %s", account.ID)
}

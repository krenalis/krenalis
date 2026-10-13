// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

// TestSimulatedSystemCheckpoint rejects changes to every persisted V5 input
// and inconsistent cursors without changing the original checkpoint.
func TestSimulatedSystemCheckpoint(t *testing.T) {

	data := simulatedSystemTestData(t)
	account := simulatedSystemPlanFixture(t, data, "account-a", 12, "50")
	var initial simulatedSystemCheckpoint
	err := json.Unmarshal(account.GenerationCheckpoint, &initial)
	if err != nil {
		t.Fatalf("expected checkpoint JSON, got %v", err)
	}
	if initial.PersonSpecVersion != "person-v2" || initial.MarketDataVersion != "it-2026a-json1" ||
		initial.WorldModelVersion != "world-v1" || initial.CatalogRelease != "artificial-photos" ||
		initial.CatalogSHA256 != "3e1ad618649b79035b4d5c6110f6fabd0fe22171b861b1ce4c3839ef972f6aa7" ||
		initial.SourceVersion != "source-v1" || initial.NamespaceVersion != 1 ||
		initial.ReferenceDate != "2026-01-01" || initial.PhotoBaseURL != "https://krenalis.test/" {
		t.Fatalf("expected V5 identities including D_photos, got %+v", initial)
	}
	fields := simulatedSystemDocumentFields(t, account.GenerationCheckpoint)
	if len(fields) != 11 {
		t.Fatalf("expected 11 existing checkpoint fields without additional L2 identities, got %v", fields)
	}

	for _, test := range []struct {
		name   string
		change func(*simulatedSystemCheckpoint)
		count  int
	}{
		{"catalog digest", func(c *simulatedSystemCheckpoint) { c.CatalogSHA256 = "changed" }, 0},
		{"catalog release", func(c *simulatedSystemCheckpoint) { c.CatalogRelease = "changed" }, 0},
		{"world", func(c *simulatedSystemCheckpoint) { c.WorldModelVersion = "changed" }, 0},
		{"market", func(c *simulatedSystemCheckpoint) { c.MarketDataVersion = "it-2026a" }, 0},
		{"person", func(c *simulatedSystemCheckpoint) { c.PersonSpecVersion = "person-v1" }, 0},
		{"source", func(c *simulatedSystemCheckpoint) { c.SourceVersion = "changed" }, 0},
		{"namespace", func(c *simulatedSystemCheckpoint) { c.NamespaceVersion++ }, 0},
		{"reference", func(c *simulatedSystemCheckpoint) { c.ReferenceDate = "2027-01-01" }, 0},
		{"empty photo URL", func(c *simulatedSystemCheckpoint) { c.PhotoBaseURL = "" }, 0},
		{"photo query", func(c *simulatedSystemCheckpoint) { c.PhotoBaseURL += "?changed=1" }, 0},
		{"negative person", func(c *simulatedSystemCheckpoint) { c.Person = -1 }, 0},
		{"person past end", func(c *simulatedSystemCheckpoint) { c.Person = 7 }, 14},
		{"negative representation", func(c *simulatedSystemCheckpoint) { c.Representation = -1 }, 0},
		{"third representation", func(c *simulatedSystemCheckpoint) { c.Representation = 2 }, 2},
		{"duplicate past end", func(c *simulatedSystemCheckpoint) { c.Person, c.Representation = 6, 1 }, 12},
		{"counter ahead", func(c *simulatedSystemCheckpoint) {}, 1},
		{"cursor ahead", func(c *simulatedSystemCheckpoint) { c.Person = 1 }, 0},
	} {
		t.Run(test.name, func(t *testing.T) {

			changed := initial
			test.change(&changed)
			account := *account
			account.GeneratedRecordCount = test.count
			account.GenerationCheckpoint, err = json.Marshal(changed)
			if err != nil {
				t.Fatalf("expected modified checkpoint JSON, got %v", err)
			}
			before := string(account.GenerationCheckpoint)
			_, _, err := newSimulatedSystemPlan("workspace-a", &account, data)
			if err != nil {
				if string(account.GenerationCheckpoint) != before || account.GeneratedRecordCount != test.count {
					t.Fatal("expected unchanged checkpoint and count on rejection, got mutation")
				}
				return
			}

			t.Fatal("expected inconsistent checkpoint rejected, got success")
		})
	}

}

// TestSimulatedSystemNamespace checks reversible coordinates and isolation of
// identical selections in separate workspace namespaces.
func TestSimulatedSystemNamespace(t *testing.T) {

	data := simulatedSystemTestData(t)
	account := simulatedSystemPlanFixture(t, data, "account-a", 1, "0")
	var previous fakedata.Person
	for _, workspace := range []string{"workspace-a", "workspace-b"} {

		plan, _, err := newSimulatedSystemPlan(workspace, account, data)
		if err != nil {
			t.Fatalf("expected workspace plan, got %v", err)
		}
		person, err := plan.world.Person(plan.personIndex(0))
		if err != nil {
			t.Fatalf("expected canonical Person, got %v", err)
		}
		key := sha256.Sum256([]byte("krenalis/simulated-world/v1/" + workspace))
		namespace, err := fakedata.NewIdentityNamespace("w"+hex.EncodeToString(key[:10]), 1)
		if err != nil {
			t.Fatalf("expected workspace namespace, got %v", err)
		}
		index, err := fakedata.PersonIndexFromID(namespace, person.Person.ID)
		if err != nil {
			t.Fatalf("expected reversible PersonID, got %v", err)
		}
		if index != plan.personIndex(0) || person.Person.ID == previous.ID {
			t.Fatalf("expected isolated reversible coordinate %d, got %d and %s", plan.personIndex(0), index, person.Person.ID)
		}
		socioeconomic, err := plan.socioeconomic.Person(index)
		if err != nil {
			t.Fatalf("expected socioeconomic Person, got %v", err)
		}
		if !reflect.DeepEqual(socioeconomic.Person, person.Person) {
			t.Fatal("expected same canonical base for Source and socioeconomic facts, got different Person")
		}
		previous = person.Person

	}

}

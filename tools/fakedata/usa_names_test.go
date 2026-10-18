// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

const usaNamesDirectory = "../../synthetic-world-usa-l1-us-2026a-NAMES-v1-CANDIDATE/"

// TestUSAFirstNamesRealProduction records the missing source-dependent tests.
func TestUSAFirstNamesRealProduction(t *testing.T) {
	t.Skip("SSA official archive/readme unavailable (HTTP 403); ten real groups/admission/sampling not implemented")
}

// TestUSANominalEntries checks NameEntry rules with technical in-memory values.
func TestUSANominalEntries(t *testing.T) {

	atom := "same-atom"
	entries := []internationalName{{"B", &atom, 1}, {"É", &atom, 2}, {"Ê", &atom, 3}}
	err := validateUSANominalEntries(entries, 3, 6)
	if err != nil {
		t.Fatalf("expected Unicode-16 keys and ASCII atom collisions admitted, got %v", err)
	}
	for _, value := range []string{"E\u0301", " é", "É  E", "\u0301E", "Ł", "😀", "", strings.Repeat("A", 81)} {
		t.Run(fmt.Sprintf("display-%x", value), func(t *testing.T) {
			fixture := []internationalName{{Display: value, ASCIIAtom: &atom, Weight: 1}}
			err := validateUSANominalEntries(fixture, 1, 1)
			if err != nil {
				return
			}
			t.Fatal("expected invalid display rejected, got nil")
		})
	}
	for _, value := range []string{"", "Same", "-same", "same-", "same--atom", "sämé", strings.Repeat("a", 81)} {
		t.Run("atom-"+value, func(t *testing.T) {
			fixture := []internationalName{{Display: "A", ASCIIAtom: &value, Weight: 1}}
			err := validateUSANominalEntries(fixture, 1, 1)
			if err != nil {
				return
			}
			t.Fatal("expected invalid ASCII atom rejected, got nil")
		})
	}
	for _, fixture := range [][]internationalName{
		{{"É", &atom, 1}, {"é", &atom, 1}}, {{"B", &atom, 1}, {"A", &atom, 1}},
		{{"A", nil, 1}}, {{"A", &atom, 0}}, {{"A", &atom, math.MaxUint64}, {"B", &atom, 1}},
	} {
		err := validateUSANominalEntries(fixture, len(fixture), 2)
		if err != nil {
			continue
		}
		t.Fatal("expected duplicate, unsorted, missing, zero or overflowing fixture rejected, got nil")
	}
	for _, count := range []int{0, -1, 5001, 2} {
		err := validateUSANominalEntries(entries, count, 6)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid nominal count rejected, got nil")
	}
	err = validateUSANominalEntries(entries, 3, 7)
	if err != nil {
		return
	}
	t.Fatal("expected wrong nominal total rejected, got nil")

}

// TestUSASurnameCandidateAdmission checks the real partial asset and ownership.
func TestUSASurnameCandidateAdmission(t *testing.T) {

	document := readUSASurnameDocument(t)
	data, err := prepareUSASurnameCandidate(document)
	if err != nil {
		t.Fatalf("expected admitted real surname candidate, got %v", err)
	}
	if len(data.entries) != 5000 || data.total != usaNamesMass ||
		fmt.Sprintf("%x", data.identity) != usaSurnameAssetSHA256 {
		t.Fatalf("expected identified 5000-entry candidate, got %d/%d/%x", len(data.entries), data.total, data.identity)
	}
	var total uint64
	for i, entry := range data.entries {
		if entry.display != document.Entries[i].Display || entry.asciiAtom != *document.Entries[i].ASCIIAtom ||
			entry.weight != document.Entries[i].Weight || data.weights[i] != entry.weight {
			t.Fatalf("expected scalar NameEntry copy at %d, got %+v", i, entry)
		}
		total += entry.weight
	}
	if total != usaNamesMass {
		t.Fatalf("expected total %d, got %d", usaNamesMass, total)
	}
	first := data.entries[0]
	document.Entries[0].Display = "CHANGED"
	*document.Entries[0].ASCIIAtom = "changed"
	document.Entries[0].Weight++
	if data.entries[0] != first || data.weights[0] != first.weight {
		t.Fatal("expected owned entries, atoms and weights, got input alias")
	}

}

// TestUSASurnameCandidateDecoder checks closed transport and early bounds.
func TestUSASurnameCandidateDecoder(t *testing.T) {

	original, err := os.ReadFile(usaNamesDirectory + "assets/surnames.json")
	if err != nil {
		t.Fatalf("expected real candidate bytes, got %v", err)
	}
	document := readUSASurnameDocument(t)
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("expected reordered transport JSON, got %v", err)
	}
	indented, err := json.Indent(encoded, "", "  ")
	if err != nil {
		t.Fatalf("expected indented transport JSON, got %v", err)
	}
	data, err := decodeUSASurnameCandidate(t.Context(), bytes.NewReader(indented))
	if err != nil {
		t.Fatalf("expected equivalent JSON admitted, got %v", err)
	}
	if fmt.Sprintf("%x", data.identity) != usaSurnameAssetSHA256 {
		t.Fatalf("expected canonical identity %s, got %x", usaSurnameAssetSHA256, data.identity)
	}
	mutations := map[string][]byte{
		"unknown":    bytes.Replace(original, []byte(`"country":"US"`), []byte(`"country":"US","unknown":1`), 1),
		"duplicate":  bytes.Replace(original, []byte(`"country":"US"`), []byte(`"country":"US","country":"US"`), 1),
		"missing":    bytes.Replace(original, []byte(`"country":"US",`), nil, 1),
		"null":       bytes.Replace(original, []byte(`"country":"US"`), []byte(`"country":null`), 1),
		"wrong-type": bytes.Replace(original, []byte(`"country":"US"`), []byte(`"country":1`), 1),
		"negative":   bytes.Replace(original, []byte(`"total":4294967295`), []byte(`"total":-1`), 1),
		"fraction":   bytes.Replace(original, []byte(`"total":4294967295`), []byte(`"total":4294967295.0`), 1),
		"exponent":   bytes.Replace(original, []byte(`"total":4294967295`), []byte(`"total":4294967295e0`), 1),
		"overflow":   bytes.Replace(original, []byte(`"total":4294967295`), []byte(`"total":18446744073709551616`), 1),
		"truncated":  original[:len(original)-2],
		"trailing":   append(slices.Clone(original), []byte("{}")...),
		"size":       bytes.Repeat([]byte(" "), maxInternationalDocumentBytes+1),
	}
	document.Entries = append(document.Entries, document.Entries[0])
	mutations["oversized-array"], err = json.Marshal(document)
	if err != nil {
		t.Fatalf("expected oversized fixture JSON, got %v", err)
	}
	for name, value := range mutations {
		t.Run(name, func(t *testing.T) {
			_, err := decodeUSASurnameCandidate(t.Context(), bytes.NewReader(value))
			if err != nil {
				return
			}
			t.Fatal("expected invalid closed candidate rejected, got nil")
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = decodeUSASurnameCandidate(ctx, bytes.NewReader(original))
	if err != nil {
		return
	}
	t.Fatal("expected canceled decode rejected, got nil")

}

// TestUSASurnameCandidateHamilton independently checks all real weights against
// the selected source counts, including stable ranking and integer remainders.
func TestUSASurnameCandidateHamilton(t *testing.T) {

	encoded, err := os.ReadFile(usaNamesDirectory + "evidence/selected-surnames.csv")
	if err != nil {
		t.Fatalf("expected selected source counts, got %v", err)
	}
	records, err := csv.NewReader(bytes.NewReader(encoded)).ReadAll()
	if err != nil {
		t.Fatalf("expected source count CSV, got %v", err)
	}
	if len(records) != 5001 || !slices.Equal(records[0], []string{
		"selectionRank", "sourceRow", "sourceRank", "display", "displayKey", "asciiAtom", "sourceCount", "weight",
	}) {
		t.Fatalf("expected 5000 selected rows with count header, got %d", len(records))
	}
	type allocation struct {
		key       string
		count     uint64
		weight    uint64
		remainder uint64
	}
	allocations := make([]allocation, 5000)
	var sourceTotal, initialTotal uint64
	for i, row := range records[1:] {

		key, err := italianDisplayKey(row[3])
		if err != nil {
			t.Fatalf("expected qualified source display, got %v", err)
		}
		count, err := strconv.ParseUint(row[6], 10, 64)
		if err != nil {
			t.Fatalf("expected exact source count, got %v", err)
		}

		if row[0] != strconv.Itoa(i+1) || key != row[4] || count == 0 ||
			count > math.MaxUint64/(usaNamesMass-5000) || count > math.MaxUint64-sourceTotal {
			t.Fatalf("expected bounded source count and canonical key at %d, got %v", i, row)
		}
		if i > 0 && (count > allocations[i-1].count ||
			count == allocations[i-1].count && key <= allocations[i-1].key) {
			t.Fatalf("expected descending count and display-key tie break at %d, got %v", i, row)
		}
		sourceTotal += count
		allocations[i] = allocation{key: key, count: count}

	}
	if sourceTotal != 174828610 {
		t.Fatalf("expected selected source mass 174828610, got %d", sourceTotal)
	}
	for i := range allocations {
		product := (usaNamesMass - 5000) * allocations[i].count
		allocations[i].weight = 1 + product/sourceTotal
		allocations[i].remainder = product % sourceTotal
		initialTotal += allocations[i].weight
	}
	slices.SortFunc(allocations, func(a, b allocation) int {
		if a.remainder > b.remainder {
			return -1
		}
		if a.remainder < b.remainder {
			return 1
		}
		return strings.Compare(a.key, b.key)
	})
	for i := uint64(0); i < usaNamesMass-initialTotal; i++ {
		allocations[i].weight++
	}
	expected := map[string]uint64{}
	for _, entry := range allocations {
		expected[entry.key] = entry.weight
	}
	document := readUSASurnameDocument(t)
	for _, entry := range document.Entries {
		key, err := italianDisplayKey(entry.Display)
		if err != nil {
			t.Fatalf("expected admitted display, got %v", err)
		}
		if entry.Weight != expected[key] {
			t.Fatalf("expected Hamilton weight %d for %s, got %d", expected[key], key, entry.Weight)
		}
	}

}

// TestUSASurnameCandidateIdentity checks declared assets and local references.
func TestUSASurnameCandidateIdentity(t *testing.T) {

	encoded, err := os.ReadFile(usaNamesDirectory + "manifest.json")
	if err != nil {
		t.Fatalf("expected partial candidate manifest, got %v", err)
	}
	var manifest struct {
		Status        string `json:"status"`
		Approval      string `json:"approval"`
		ContentSHA256 string `json:"contentSHA256"`
		Files         map[string]struct {
			Size   uint64 `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
		References map[string]struct {
			Size   uint64 `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"references"`
	}
	err = json.Unmarshal(encoded, &manifest)
	if err != nil {
		t.Fatalf("expected decoded partial manifest, got %v", err)
	}
	canonical, err := internationalCanonical(manifest.Files)
	if err != nil {
		t.Fatalf("expected canonical declared file map, got %v", err)
	}
	if manifest.Status != "PARTIAL_BLOCKED_SSA" || manifest.Approval != "NOT OWNER-APPROVED" ||
		fmt.Sprintf("%x", sha256.Sum256(canonical)) != manifest.ContentSHA256 || len(manifest.Files) != 13 {
		t.Fatalf("expected 13 identified partial files, got %d/%s/%s",
			len(manifest.Files), manifest.Status, manifest.ContentSHA256)
	}
	for name, entry := range manifest.Files {
		data, err := os.ReadFile(usaNamesDirectory + name)
		if err != nil {
			t.Fatalf("expected declared file %s, got %v", name, err)
		}
		if uint64(len(data)) != entry.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != entry.SHA256 {
			t.Fatalf("expected declared size/digest for %s, got %d/%x", name, len(data), sha256.Sum256(data))
		}
	}
	if len(manifest.References) != 4 {
		t.Fatalf("expected four local references, got %d", len(manifest.References))
	}
	for name, entry := range manifest.References {
		data, err := os.ReadFile("../../" + name)
		if err != nil {
			t.Fatalf("expected local reference %s, got %v", name, err)
		}
		if uint64(len(data)) != entry.Size || fmt.Sprintf("%x", sha256.Sum256(data)) != entry.SHA256 {
			t.Fatalf("expected local reference size/digest for %s, got %d/%x", name, len(data), sha256.Sum256(data))
		}
	}
	policy, err := os.ReadFile(usaNamesDirectory + "policy.json")
	if err != nil {
		t.Fatalf("expected candidate policy, got %v", err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(policy)) != usaNamesPolicySHA256 {
		t.Fatalf("expected policy digest %s, got %x", usaNamesPolicySHA256, sha256.Sum256(policy))
	}

}

// TestUSASurnameCandidateInvalid checks real metadata and value mutations.
func TestUSASurnameCandidateInvalid(t *testing.T) {

	mutations := map[string]func(*usaSurnameDocument){
		"schema":        func(d *usaSurnameDocument) { d.Schema = "sw-base-data-v1" },
		"status":        func(d *usaSurnameDocument) { d.Status = "APPROVED" },
		"approval":      func(d *usaSurnameDocument) { d.Approval = "OWNER-APPROVED" },
		"country":       func(d *usaSurnameDocument) { d.Country = "IT" },
		"anchor":        func(d *usaSurnameDocument) { d.AnchorDate = "2026-01-02" },
		"vintage":       func(d *usaSurnameDocument) { d.Vintage = "2010" },
		"research":      func(d *usaSurnameDocument) { d.SourceVariant = "Without Negative Number Removal" },
		"source-file":   func(d *usaSurnameDocument) { d.SourceFile = "Names2020_FirstNames_Sex.xlsx" },
		"source-digest": func(d *usaSurnameDocument) { d.SourceSHA256 = strings.Repeat("0", 64) },
		"frequency":     func(d *usaSurnameDocument) { d.FrequencyColumn = "D:PROPORTION PER 100,000 POPULATION" },
		"policy":        func(d *usaSurnameDocument) { d.PolicySHA256 = strings.Repeat("0", 64) },
		"kind":          func(d *usaSurnameDocument) { d.WeightKind = "population" },
		"limit":         func(d *usaSurnameDocument) { d.SelectionLimit = 600 },
		"cardinality":   func(d *usaSurnameDocument) { d.SourceValidNames-- },
		"source-mass":   func(d *usaSurnameDocument) { d.SourceNamedMass-- },
		"other-mass":    func(d *usaSurnameDocument) { d.SourceOtherMass-- },
		"selected-mass": func(d *usaSurnameDocument) { d.SelectedSourceMass-- },
		"omitted-mass":  func(d *usaSurnameDocument) { d.OmittedNamedMass-- },
		"total":         func(d *usaSurnameDocument) { d.Total-- },
		"missing":       func(d *usaSurnameDocument) { d.Entries = d.Entries[:4999] },
		"oversized":     func(d *usaSurnameDocument) { d.Entries = append(d.Entries, d.Entries[0]) },
		"duplicate":     func(d *usaSurnameDocument) { d.Entries[1] = d.Entries[0] },
		"folded-duplicate": func(d *usaSurnameDocument) {
			d.Entries[1].Display = strings.ToLower(d.Entries[0].Display)
		},
		"unsorted":          func(d *usaSurnameDocument) { d.Entries[0], d.Entries[1] = d.Entries[1], d.Entries[0] },
		"display":           func(d *usaSurnameDocument) { d.Entries[0].Display = "" },
		"atom-missing":      func(d *usaSurnameDocument) { d.Entries[0].ASCIIAtom = nil },
		"atom-format":       func(d *usaSurnameDocument) { *d.Entries[0].ASCIIAtom = "UPPER" },
		"zero":              func(d *usaSurnameDocument) { d.Entries[0].Weight = 0 },
		"overflow":          func(d *usaSurnameDocument) { d.Entries[0].Weight = math.MaxUint64 },
		"sum":               func(d *usaSurnameDocument) { d.Entries[0].Weight++ },
		"digest-display":    func(d *usaSurnameDocument) { d.Entries[0].Display += "A" },
		"digest-atom":       func(d *usaSurnameDocument) { *d.Entries[0].ASCIIAtom = "different" },
		"digest-same-total": func(d *usaSurnameDocument) { d.Entries[0].Weight++; d.Entries[1].Weight-- },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			document := readUSASurnameDocument(t)
			mutate(document)
			_, err := prepareUSASurnameCandidate(document)
			if err != nil {
				return
			}
			t.Fatal("expected altered surname candidate rejected, got nil")
		})
	}
	_, err := prepareUSASurnameCandidate(nil)
	if err != nil {
		return
	}
	t.Fatal("expected nil candidate rejected, got success")

}

// TestUSASurnameCandidateSampling checks one frozen weighted draw and replay.
func TestUSASurnameCandidateSampling(t *testing.T) {

	data, err := prepareUSASurnameCandidate(readUSASurnameDocument(t))
	if err != nil {
		t.Fatalf("expected admitted candidate, got %v", err)
	}
	rng, replay, expected := splitMix64{state: 12345}, splitMix64{state: 12345}, splitMix64{state: 12345}
	for range 2000 {
		position, err := expected.WeightedIndex(data.weights)
		if err != nil {
			t.Fatalf("expected weighted position, got %v", err)
		}
		first, err := data.selectSurname(&rng)
		if err != nil {
			t.Fatalf("expected selected surname, got %v", err)
		}
		second, err := data.selectSurname(&replay)
		if err != nil {
			t.Fatalf("expected replay surname, got %v", err)
		}
		if first != data.entries[position] || first != second ||
			rng.state != expected.state || rng.state != replay.state {
			t.Fatalf("expected one weighted draw and exact replay, got %+v/%+v", first, second)
		}
		first.display = "CHANGED"
		if data.entries[position].display == first.display {
			t.Fatal("expected scalar output copy, got mutable output alias")
		}
	}
	for _, candidate := range []*usaSurnameCandidate{nil, {}, data} {
		_, err := candidate.selectSurname(nil)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid selector rejected, got nil")
	}

}

func readUSASurnameDocument(t *testing.T) *usaSurnameDocument {

	t.Helper()
	encoded, err := os.ReadFile(usaNamesDirectory + "assets/surnames.json")
	if err != nil {
		t.Fatalf("expected real surname document, got %v", err)
	}

	var document usaSurnameDocument
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), &document)
	if err != nil {
		t.Fatalf("expected decoded real surname document, got %v", err)
	}

	return &document
}

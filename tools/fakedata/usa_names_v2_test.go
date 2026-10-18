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
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	"github.com/krenalis/krenalis/tools/json"
)

const (
	usaSurname50KDirectory   = "../../synthetic-world-usa-l1-us-2026a-BASE-v2-CANDIDATE/"
	usaFirstNamesV2Directory = "../../synthetic-world-usa-l1-us-2026a-BASE-v2-CANDIDATE/"
)

// BenchmarkUSANamesV2Preparation distinguishes admission from full decoding.
func BenchmarkUSANamesV2Preparation(b *testing.B) {

	for _, size := range []int{5000, 50000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {

			path := usaNamesDirectory + "assets/surnames.json"
			if size == 50000 {
				path = usaSurname50KDirectory + "assets/surnames.json"
			}
			encoded, err := os.ReadFile(path)
			if err != nil {
				b.Fatalf("expected real asset, got %v", err)
			}
			var small usaSurnameDocument
			var large usaSurname50KDocument
			if size == 5000 {
				err = json.Unmarshal(encoded, &small)
			} else {
				err = json.Unmarshal(encoded, &large)
			}
			if err != nil {
				b.Fatalf("expected parsed real document, got %v", err)
			}
			for _, decode := range []bool{false, true} {
				label := "admit-parsed"
				if decode {
					label = "decode-and-admit"
				}
				b.Run(label, func(b *testing.B) {

					b.ReportAllocs()
					for b.Loop() {
						if size == 5000 {
							var prepared *usaSurnameCandidate
							if decode {
								prepared, err = decodeUSASurnameCandidate(b.Context(), bytes.NewReader(encoded))
							} else {
								prepared, err = prepareUSASurnameCandidate(&small)
							}
							runtime.KeepAlive(prepared)
						} else {
							var prepared *usaSurname50KCandidate
							if decode {
								prepared, err = decodeUSASurname50KCandidate(b.Context(), bytes.NewReader(encoded))
							} else {
								prepared, err = prepareUSASurname50KCandidate(&large)
							}
							runtime.KeepAlive(prepared)
						}
						if err != nil {
							b.Fatalf("expected valid admission, got %v", err)
						}
					}

				})
			}

		})
	}

}

// BenchmarkUSANamesV2Selection compares identical real weights at both sizes.
func BenchmarkUSANamesV2Selection(b *testing.B) {

	for _, size := range []int{5000, 50000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {

			path := usaNamesDirectory + "assets/surnames.json"
			if size == 50000 {
				path = usaSurname50KDirectory + "assets/surnames.json"
			}
			encoded, err := os.ReadFile(path)
			if err != nil {
				b.Fatalf("expected real asset, got %v", err)
			}
			var entries []internationalName
			if size == 5000 {
				var document usaSurnameDocument
				err = json.Unmarshal(encoded, &document)
				entries = document.Entries
			} else {
				var document usaSurname50KDocument
				err = json.Unmarshal(encoded, &document)
				entries = document.Entries
			}
			if err != nil {
				b.Fatalf("expected parsed real document, got %v", err)
			}
			index, err := newUSANominalIndex(entries, size, usaNamesMass)
			if err != nil {
				b.Fatalf("expected immutable index, got %v", err)
			}
			weights := make([]uint64, size)
			payload := size * (int(unsafe.Sizeof(usaNameRow{})) + 8)
			for i, entry := range entries {
				weights[i] = entry.Weight
				payload += len(entry.Display) + len(*entry.ASCIIAtom)
			}
			b.Run("linear", func(b *testing.B) {

				rng := &splitMix64{state: 2026}
				b.ReportAllocs()
				for b.Loop() {
					_, err := rng.WeightedIndex(weights)
					if err != nil {
						b.Fatalf("expected valid draw, got %v", err)
					}
				}
				b.ReportMetric(float64(payload), "retained-payload-B")

			})
			b.Run("indexed", func(b *testing.B) {

				rng := &splitMix64{state: 2026}
				b.ReportAllocs()
				for b.Loop() {
					_, err := index.selectName(rng)
					if err != nil {
						b.Fatalf("expected valid draw, got %v", err)
					}
				}
				b.ReportMetric(float64(payload), "retained-payload-B")

			})

		})
	}

}

// TestUSANamesV2Admission checks asset identities and immutable ownership.
func TestUSANamesV2Admission(t *testing.T) {

	var surnames usaSurname50KDocument
	readUSANamesV2Document(t, usaSurname50KDirectory+"assets/surnames.json", &surnames)
	large, err := prepareUSASurname50KCandidate(&surnames)
	if err != nil {
		t.Fatalf("expected admitted real 50K asset, got %v", err)
	}
	var firstNames usaFirstNamesV2Document
	readUSANamesV2Document(t, usaFirstNamesV2Directory+"assets/first-names.json", &firstNames)
	first, err := prepareUSAFirstNamesV2Candidate(&firstNames)
	if err != nil {
		t.Fatalf("expected admitted ten-group asset, got %v", err)
	}
	if fmt.Sprintf("%x", large.identity) != usaSurname50KAssetSHA256 || len(large.names.entries) != 50000 ||
		fmt.Sprintf("%x", first.identity) != usaFirstNamesV2AssetSHA256 || large.identity == first.identity {
		t.Fatalf("expected distinct exact asset identities, got %x/%x", large.identity, first.identity)
	}
	check := func(index *usaNominalIndex, entries []internationalName) {

		t.Helper()
		var total uint64
		for i, entry := range entries {
			total += entry.Weight
			if index.entries[i] != (usaNameRow{entry.Display, *entry.ASCIIAtom, entry.Weight}) || index.cumulative[i] != total {
				t.Fatalf("expected owned row and cumulative end at %d, got %+v/%d", i, index.entries[i], index.cumulative[i])
			}
		}
		original := index.entries[0]
		entries[0].Display = "CHANGED"
		*entries[0].ASCIIAtom = "changed"
		entries[0].Weight++
		if index.entries[0] != original || index.cumulative[0] != original.weight {
			t.Fatal("expected immutable scalar ownership, got input alias")
		}

	}
	check(large.names, surnames.Entries)
	for i, group := range firstNames.Groups {
		check(first.groups[i], group.Entries)
	}
	if first.groups[0] == first.groups[1] || &first.groups[0].entries[0] == &first.groups[1].entries[0] {
		t.Fatal("expected independently owned cohort tables, got mutable alias")
	}

}

// TestUSANamesV2ClosedDecoders checks transport errors and early array budgets.
func TestUSANamesV2ClosedDecoders(t *testing.T) {

	for _, isSurname := range []bool{true, false} {
		label, path := "surnames", usaSurname50KDirectory+"assets/surnames.json"
		if !isSurname {
			label, path = "first-names", usaFirstNamesV2Directory+"assets/first-names.json"
		}
		t.Run(label, func(t *testing.T) {

			encoded, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("expected real asset bytes, got %v", err)
			}
			decode := func(ctx context.Context, value []byte) error {
				if isSurname {
					_, err := decodeUSASurname50KCandidate(ctx, bytes.NewReader(value))
					return err
				}
				_, err := decodeUSAFirstNamesV2Candidate(ctx, bytes.NewReader(value))
				return err
			}
			var transport any = &usaFirstNamesV2Document{}
			if isSurname {
				transport = &usaSurname50KDocument{}
			}
			err = json.Unmarshal(encoded, transport)
			if err != nil {
				t.Fatalf("expected parsed transport, got %v", err)
			}
			reordered, err := json.Marshal(transport)
			if err != nil {
				t.Fatalf("expected reordered object fields, got %v", err)
			}
			indented, err := json.Indent(reordered, "", "  ")
			if err != nil {
				t.Fatalf("expected equivalent JSON, got %v", err)
			}
			err = decode(t.Context(), indented)
			if err != nil {
				t.Fatalf("expected equivalent canonical admission, got %v", err)
			}
			mutations := map[string][]byte{
				"unknown":   bytes.Replace(encoded, []byte(`"country":"US"`), []byte(`"country":"US","unknown":1`), 1),
				"duplicate": bytes.Replace(encoded, []byte(`"country":"US"`), []byte(`"country":"US","country":"US"`), 1),
				"missing":   bytes.Replace(encoded, []byte(`"country":"US",`), nil, 1),
				"null":      bytes.Replace(encoded, []byte(`"country":"US"`), []byte(`"country":null`), 1),
				"type":      bytes.Replace(encoded, []byte(`"country":"US"`), []byte(`"country":1`), 1),
				"negative":  bytes.Replace(encoded, []byte(`"total":4294967295`), []byte(`"total":-1`), 1),
				"fraction":  bytes.Replace(encoded, []byte(`"total":4294967295`), []byte(`"total":4294967295.0`), 1),
				"exponent":  bytes.Replace(encoded, []byte(`"total":4294967295`), []byte(`"total":4294967295e0`), 1),
				"overflow":  bytes.Replace(encoded, []byte(`"total":4294967295`), []byte(`"total":18446744073709551616`), 1),
				"truncated": encoded[:len(encoded)-2],
				"trailing":  append(slices.Clone(encoded), []byte("{}")...),
				"budget":    bytes.Repeat([]byte(" "), maxInternationalDocumentBytes+1),
			}
			if isSurname {
				var document usaSurname50KDocument
				readUSANamesV2Document(t, path, &document)
				document.Entries = append(document.Entries, document.Entries[0])
				mutations["array"], err = json.Marshal(document)
			} else {
				var document usaFirstNamesV2Document
				readUSANamesV2Document(t, path, &document)
				document.Groups = append(document.Groups, document.Groups[0])
				mutations["groups"], err = json.Marshal(document)
				if err != nil {
					t.Fatalf("expected excessive-group fixture, got %v", err)
				}
				document.Groups = document.Groups[:10]
				document.Groups[0].Entries = append(document.Groups[0].Entries, document.Groups[0].Entries[0])
				mutations["array"], err = json.Marshal(document)
			}
			if err != nil {
				t.Fatalf("expected excessive-array fixture, got %v", err)
			}
			for name, value := range mutations {
				t.Run(name, func(t *testing.T) {
					err := decode(t.Context(), value)
					if err != nil {
						return
					}
					t.Fatal("expected invalid document rejected, got nil")
				})
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			err = decode(ctx, encoded)
			if err != nil {
				return
			}
			t.Fatal("expected canceled decode rejected, got nil")

		})
	}

}

// TestUSANamesV2GroupSelection checks the supplied gender and every cohort.
func TestUSANamesV2GroupSelection(t *testing.T) {

	var document usaFirstNamesV2Document
	readUSANamesV2Document(t, usaFirstNamesV2Directory+"assets/first-names.json", &document)
	prepared, err := prepareUSAFirstNamesV2Candidate(&document)
	if err != nil {
		t.Fatalf("expected admitted groups, got %v", err)
	}
	for i, group := range document.Groups {
		t.Run(string(group.Gender)+"/"+group.Cohort, func(t *testing.T) {

			weights := make([]uint64, len(group.Entries))
			for j, entry := range group.Entries {
				weights[j] = entry.Weight
			}
			for _, seed := range []uint64{0, 1, 2026, math.MaxUint64} {
				a, b := &splitMix64{state: seed}, &splitMix64{state: seed}
				for range 128 {
					position, err := a.WeightedIndex(weights)
					if err != nil {
						t.Fatalf("expected reference draw, got %v", err)
					}
					row, err := prepared.selectFirstName(b, group.Gender, group.Cohort)
					if err != nil {
						t.Fatalf("expected indexed group draw, got %v", err)
					}
					if row != prepared.groups[i].entries[position] || a.state != b.state {
						t.Fatalf("expected same group row and RNG state, got %+v/%x/%x", row, a.state, b.state)
					}
				}
			}
			// Distinct test-only markers make incorrect cohort routing observable
			// even though Method B intentionally repeats each sex distribution.
			marked := *prepared
			for j, index := range prepared.groups {
				copyIndex := *index
				copyIndex.entries = slices.Clone(index.entries)
				for k := range copyIndex.entries {
					copyIndex.entries[k].display = strconv.Itoa(j)
				}
				marked.groups[j] = &copyIndex
			}
			row, err := marked.selectFirstName(&splitMix64{}, group.Gender, group.Cohort)
			if err != nil {
				t.Fatalf("expected marker selection, got %v", err)
			}
			if row.display != strconv.Itoa(i) {
				t.Fatalf("expected group %d, got %s", i, row.display)
			}

		})
	}
	for _, args := range [][2]string{{"", "1950-1964"}, {"female", "2006-2009"}, {"male", ""}} {
		rng := &splitMix64{state: 99}
		_, err := prepared.selectFirstName(rng, Gender(args[0]), args[1])
		if err != nil {
			if rng.state != 99 {
				t.Fatal("expected invalid group consumes no draws, got changed state")
			}
			continue
		}
		t.Fatal("expected invalid group rejected, got nil")
	}

}

// TestUSANamesV2Hamilton reconstructs weights from source ledgers.
func TestUSANamesV2Hamilton(t *testing.T) {

	var surnames usaSurname50KDocument
	readUSANamesV2Document(t, usaSurname50KDirectory+"assets/surnames.json", &surnames)
	checkUSANamesV2Hamilton(t,
		"../../synthetic-world-usa-l1-us-2026a-SURNAMES-50K-v2-CANDIDATE/evidence/selected-surnames.csv",
		50000, 239992615, 3, 4, 6, surnames.Entries, "")
	var first usaFirstNamesV2Document
	readUSANamesV2Document(t, usaFirstNamesV2Directory+"assets/first-names.json", &first)
	for _, group := range first.Groups {
		checkUSANamesV2Hamilton(t,
			"../../synthetic-world-usa-l1-us-2026a-FIRSTNAMES-v2-CANDIDATE/evidence/selected-first-names.csv", 1000,
			group.SelectedSourceMass, 5, 6, 8, group.Entries, string(group.Gender)+"/"+group.Cohort)
	}

}

// TestUSANamesV2IndexEquivalence checks boundaries, state and rejection draws.
func TestUSANamesV2IndexEquivalence(t *testing.T) {

	tables := [][]uint64{
		{1}, {1, 1, 1}, {1, 999999999, 1}, {1, usaNamesMass - 2, 1}, {math.MaxUint64 - 2, 1, 1}, {1, 1 << 63},
	}
	for _, size := range []int{5000, 50000} {
		weights := make([]uint64, size)
		for i := range weights {
			weights[i] = uint64(i%31 + 1)
		}
		tables = append(tables, weights)
	}
	for table, weights := range tables {
		t.Run(strconv.Itoa(table), func(t *testing.T) {

			index := &usaNominalIndex{entries: make([]usaNameRow, len(weights)), cumulative: make([]uint64, len(weights))}
			for i, weight := range weights {
				index.total += weight
				index.cumulative[i] = index.total
				index.entries[i] = usaNameRow{display: strconv.Itoa(i), weight: weight}
			}
			check := func(seed uint64) {

				t.Helper()
				a, b := &splitMix64{state: seed}, &splitMix64{state: seed}
				position, err := a.WeightedIndex(weights)
				if err != nil {
					t.Fatalf("expected reference selection, got %v", err)
				}
				row, err := index.selectName(b)
				if err != nil {
					t.Fatalf("expected indexed selection, got %v", err)
				}
				if row != index.entries[position] || a.state != b.state || a.Uint64() != b.Uint64() {
					t.Fatalf("expected equal index and exact final RNG state at seed %x, got %+v/%x/%x", seed, row, a.state, b.state)
				}

			}
			for _, seed := range []uint64{0, 1, 2, 42, 2026, 1 << 63, math.MaxUint64} {
				for i := uint64(0); i < 128; i++ {
					check(seed + i*0x9e3779b97f4a7c15)
				}
			}
			threshold := (uint64(0) - index.total) % index.total
			for i, end := range index.cumulative {
				if len(weights) > 10 && i%997 != 0 && i != len(weights)-1 {
					continue
				}
				start := end - weights[i]
				for _, draw := range []uint64{start, end - 1, end % index.total} {
					output := draw
					if output < threshold {
						output += index.total
					}
					check(usaNamesV2SeedForOutput(output))
				}
			}
			if threshold > 0 {
				seed := usaNamesV2SeedForOutput(0)
				probe := &splitMix64{state: seed}
				_, err := probe.Uint64n(index.total)
				if err != nil {
					t.Fatalf("expected rejection reference, got %v", err)
				}
				if probe.state == seed+0x9e3779b97f4a7c15 {
					t.Fatal("expected exercised rejection draw, got one draw")
				}
				check(seed)
			}

		})
	}

}

// TestUSANamesV2IndexInvalid rejects invalid entries and nil samplers.
func TestUSANamesV2IndexInvalid(t *testing.T) {

	atom := "same-atom"
	entries := []internationalName{{"B", &atom, 1}, {"É", &atom, 2}, {"Ê", &atom, 3}}
	_, err := newUSANominalIndex(entries, 3, 6)
	if err != nil {
		t.Fatalf("expected Unicode16 entries and atom collisions admitted, got %v", err)
	}
	for _, display := range []string{"E\u0301", " é", "A  B", "\u0301A", "Ł", "😀", "", strings.Repeat("A", 81)} {
		_, err := newUSANominalIndex([]internationalName{{display, &atom, 1}}, 1, 1)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid display rejected, got %q", display)
	}
	for _, atom := range []string{"", "A", "-a", "a-", "a--b", "é", strings.Repeat("a", 81)} {
		_, err := newUSANominalIndex([]internationalName{{"A", &atom, 1}}, 1, 1)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid atom rejected, got %q", atom)
	}
	for _, entries := range [][]internationalName{
		{{"A", nil, 1}}, {{"A", &atom, 0}}, {{"B", &atom, 1}, {"A", &atom, 1}},
		{{"É", &atom, 1}, {"é", &atom, 1}}, {{"A", &atom, math.MaxUint64}, {"B", &atom, 1}},
	} {
		_, err := newUSANominalIndex(entries, len(entries), 2)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid entries rejected, got nil")
	}
	for _, count := range []int{-1, 0, 50001, 2} {
		_, err := newUSANominalIndex(entries, count, 6)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid cardinality rejected, got nil")
	}
	t.Run("total", func(t *testing.T) {
		_, err := newUSANominalIndex(entries, 3, 7)
		if err != nil {
			return
		}
		t.Fatal("expected total mismatch rejected, got nil")
	})
	for _, index := range []*usaNominalIndex{nil, {}, {entries: []usaNameRow{{}}, cumulative: []uint64{0}}} {
		_, err := index.selectName(&splitMix64{})
		if err != nil {
			continue
		}
		t.Fatal("expected uninitialized index rejected, got nil")
	}
	for name, sample := range map[string]func() error{
		"surname-nil": func() error { _, err := (*usaSurname50KCandidate)(nil).selectSurname(&splitMix64{}); return err },
		"first-nil": func() error {
			_, err := (*usaFirstNamesV2Candidate)(nil).selectFirstName(&splitMix64{}, GenderFemale, "1950-1964")
			return err
		},
		"first-rng": func() error {
			_, err := (&usaFirstNamesV2Candidate{}).selectFirstName(nil, GenderFemale, "1950-1964")
			return err
		},
		"first-empty": func() error {
			_, err := (&usaFirstNamesV2Candidate{}).selectFirstName(&splitMix64{}, GenderFemale, "1950-1964")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := sample()
			if err != nil {
				return
			}
			t.Fatal("expected invalid sampler rejected, got nil")
		})
	}

}

// TestUSANamesV2MetadataMutations rejects altered provenance and groups.
func TestUSANamesV2MetadataMutations(t *testing.T) {

	for name, mutate := range map[string]func(*usaSurname50KDocument){
		"schema":        func(d *usaSurname50KDocument) { d.Schema = "sw-us-surnames-candidate-v1" },
		"status":        func(d *usaSurname50KDocument) { d.Status = "PUBLISHED" },
		"approval":      func(d *usaSurname50KDocument) { d.Approval = "NOT OWNER-APPROVED" },
		"source":        func(d *usaSurname50KDocument) { d.SourceSHA256 = usaFirstNamesSourceSHA256 },
		"column":        func(d *usaSurname50KDocument) { d.FrequencyColumn = "D" },
		"policy":        func(d *usaSurname50KDocument) { d.PolicySHA256 = usaNamesPolicySHA256 },
		"mass":          func(d *usaSurname50KDocument) { d.SelectedSourceMass++ },
		"limit":         func(d *usaSurname50KDocument) { d.SelectionLimit = 5000 },
		"count":         func(d *usaSurname50KDocument) { d.Entries = d.Entries[:49999] },
		"atom-digest":   func(d *usaSurname50KDocument) { *d.Entries[0].ASCIIAtom = "changed" },
		"weight-digest": func(d *usaSurname50KDocument) { d.Entries[0].Weight++; d.Entries[1].Weight-- },
	} {
		t.Run("surname/"+name, func(t *testing.T) {
			var d usaSurname50KDocument
			readUSANamesV2Document(t, usaSurname50KDirectory+"assets/surnames.json", &d)
			mutate(&d)
			_, err := prepareUSASurname50KCandidate(&d)
			if err != nil {
				return
			}
			t.Fatal("expected altered 50K asset rejected, got nil")
		})
	}
	for name, mutate := range map[string]func(*usaFirstNamesV2Document){
		"schema":          func(d *usaFirstNamesV2Document) { d.Schema = "other" },
		"approval":        func(d *usaFirstNamesV2Document) { d.Approval = "OWNER-APPROVED" },
		"source":          func(d *usaFirstNamesV2Document) { d.SourceSHA256 = usaSurnameSourceSHA256 },
		"columns":         func(d *usaFirstNamesV2Document) { d.FrequencyColumns = "D:MALE;E:FEMALE" },
		"policy":          func(d *usaFirstNamesV2Document) { d.PolicySHA256 = usaNamesPolicySHA256 },
		"ssa":             func(d *usaFirstNamesV2Document) { d.SSAAccessSHA256 = "invented" },
		"age":             func(d *usaFirstNamesV2Document) { d.AgeAssetSHA256 = "changed" },
		"birth":           func(d *usaFirstNamesV2Document) { d.BirthYearsSHA256 = "changed" },
		"method":          func(d *usaFirstNamesV2Document) { d.Method = "A:SSA" },
		"cohort-claim":    func(d *usaFirstNamesV2Document) { d.CohortMeaning = "observed" },
		"group-count":     func(d *usaFirstNamesV2Document) { d.Groups = d.Groups[:9] },
		"group-duplicate": func(d *usaFirstNamesV2Document) { d.Groups[1] = d.Groups[0] },
		"group-order":     func(d *usaFirstNamesV2Document) { d.Groups[0], d.Groups[5] = d.Groups[5], d.Groups[0] },
		"gender":          func(d *usaFirstNamesV2Document) { d.Groups[0].Gender = "unknown" },
		"cohort":          func(d *usaFirstNamesV2Document) { d.Groups[0].Cohort = "2006-2009" },
		"source-mass":     func(d *usaFirstNamesV2Document) { d.Groups[0].SourceNamedMass++ },
		"selected-mass":   func(d *usaFirstNamesV2Document) { d.Groups[0].SelectedSourceMass = math.MaxUint64 },
		"omitted-mass":    func(d *usaFirstNamesV2Document) { d.Groups[0].OmittedNamedMass++ },
		"count":           func(d *usaFirstNamesV2Document) { d.Groups[0].Entries = d.Groups[0].Entries[:999] },
		"generation": func(d *usaFirstNamesV2Document) {
			d.Groups[1].Entries[0].Weight++
			d.Groups[1].Entries[1].Weight--
		},
		"atom-digest": func(d *usaFirstNamesV2Document) { *d.Groups[0].Entries[0].ASCIIAtom = "changed" },
	} {
		t.Run("first/"+name, func(t *testing.T) {
			var d usaFirstNamesV2Document
			readUSANamesV2Document(t, usaFirstNamesV2Directory+"assets/first-names.json", &d)
			mutate(&d)
			_, err := prepareUSAFirstNamesV2Candidate(&d)
			if err != nil {
				return
			}
			t.Fatal("expected altered first-name asset rejected, got nil")
		})
	}

}

// TestUSANamesV2RetainedMemory measures GC deltas; run alone for comparison.
func TestUSANamesV2RetainedMemory(t *testing.T) {

	for _, size := range []int{5000, 50000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {

			path := usaNamesDirectory + "assets/surnames.json"
			if size == 50000 {
				path = usaSurname50KDirectory + "assets/surnames.json"
			}
			encoded, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("expected real asset bytes, got %v", err)
			}
			// Warm type/canonical caches before measuring a retained admission.
			decode := func() any {

				t.Helper()
				if size == 5000 {
					value, err := decodeUSASurnameCandidate(t.Context(), bytes.NewReader(encoded))
					if err != nil {
						t.Fatalf("expected 5K decode, got %v", err)
					}
					return value
				}
				value, err := decodeUSASurname50KCandidate(t.Context(), bytes.NewReader(encoded))
				if err != nil {
					t.Fatalf("expected 50K decode, got %v", err)
				}

				return value
			}
			warm := decode()
			runtime.GC()
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			prepared := decode()
			runtime.GC()
			runtime.GC()
			runtime.ReadMemStats(&after)
			t.Logf("retained heap %d names: %d bytes (GC delta, allocator/runtime dependent)",
				size, int64(after.HeapAlloc)-int64(before.HeapAlloc))
			runtime.KeepAlive(prepared)
			runtime.KeepAlive(encoded)
			runtime.KeepAlive(warm)

		})
	}

}

// TestUSANamesV2SurnameReplay checks sampling and cross-version refusal.
func TestUSANamesV2SurnameReplay(t *testing.T) {

	var document usaSurname50KDocument
	readUSANamesV2Document(t, usaSurname50KDirectory+"assets/surnames.json", &document)
	prepared, err := prepareUSASurname50KCandidate(&document)
	if err != nil {
		t.Fatalf("expected admitted 50K asset, got %v", err)
	}
	weights := make([]uint64, len(document.Entries))
	for i, entry := range document.Entries {
		weights[i] = entry.Weight
	}
	for _, seed := range []uint64{0, 1, 2026, 1 << 63, math.MaxUint64, usaNamesV2SeedForOutput(0)} {
		a, b := &splitMix64{state: seed}, &splitMix64{state: seed}
		for range 256 {
			position, err := a.WeightedIndex(weights)
			if err != nil {
				t.Fatalf("expected linear draw, got %v", err)
			}
			row, err := prepared.selectSurname(b)
			if err != nil {
				t.Fatalf("expected indexed draw, got %v", err)
			}
			if row != prepared.names.entries[position] || a.state != b.state {
				t.Fatalf("expected equal surname and final state, got %+v/%x/%x", row, a.state, b.state)
			}
		}
	}
	for _, pair := range []struct {
		path  string
		large bool
	}{
		{usaNamesDirectory + "assets/surnames.json", true}, {usaSurname50KDirectory + "assets/surnames.json", false},
	} {
		encoded, err := os.ReadFile(pair.path)
		if err != nil {
			t.Fatalf("expected cross-version bytes, got %v", err)
		}
		if pair.large {
			_, err = decodeUSASurname50KCandidate(t.Context(), bytes.NewReader(encoded))
		} else {
			_, err = decodeUSASurnameCandidate(t.Context(), bytes.NewReader(encoded))
		}
		if err != nil {
			continue
		}
		t.Fatal("expected incompatible version rejected, got nil")
	}

}

// checkUSANamesV2Hamilton independently allocates the column-indexed source
// ledger, optionally restricting its rows to one gender/cohort group.
func checkUSANamesV2Hamilton(t *testing.T, path string, count int, total uint64, displayColumn, keyColumn, countColumn int, entries []internationalName, group string) {

	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected source ledger, got %v", err)
	}
	records, err := csv.NewReader(bytes.NewReader(encoded)).ReadAll()
	if err != nil {
		t.Fatalf("expected CSV rows, got %v", err)
	}
	type allocation struct {
		key                      string
		count, weight, remainder uint64
	}
	allocations := make([]allocation, 0, count)
	var sourceTotal, initialTotal uint64
	for _, row := range records[1:] {

		if group != "" && row[0]+"/"+row[1] != group {
			continue
		}
		key, err := italianDisplayKey(row[displayColumn])
		if err != nil {
			t.Fatalf("expected Unicode16 source key, got %v", err)
		}
		mass, err := strconv.ParseUint(row[countColumn], 10, 64)
		if err != nil {
			t.Fatalf("expected integer source mass, got %v", err)
		}
		if key != row[keyColumn] || mass == 0 || mass > math.MaxUint64/(usaNamesMass-uint64(count)) ||
			mass > math.MaxUint64-sourceTotal {
			t.Fatalf("expected bounded source key and count, got %v", row)
		}
		if len(allocations) > 0 {
			previous := allocations[len(allocations)-1]
			if mass > previous.count || mass == previous.count && key <= previous.key {
				t.Fatalf("expected stable full mass ranking, got %v", row)
			}
		}
		sourceTotal += mass
		allocations = append(allocations, allocation{key: key, count: mass})

	}
	if len(allocations) != count || sourceTotal != total {
		t.Fatalf("expected %d rows and %d mass, got %d/%d", count, total, len(allocations), sourceTotal)
	}
	for i := range allocations {
		product := (usaNamesMass - uint64(count)) * allocations[i].count
		allocations[i].weight = 1 + product/total
		allocations[i].remainder = product % total
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
	slices.SortFunc(allocations, func(a, b allocation) int { return strings.Compare(a.key, b.key) })
	var actualTotal uint64
	for i, entry := range entries {
		key, err := italianDisplayKey(entry.Display)
		if err != nil {
			t.Fatalf("expected entry key, got %v", err)
		}
		if key != allocations[i].key || entry.Weight != allocations[i].weight || entry.Weight == 0 {
			t.Fatalf("expected exact positive Hamilton weight at %s, got %d", key, entry.Weight)
		}
		actualTotal += entry.Weight
	}
	if actualTotal != usaNamesMass {
		t.Fatalf("expected M=%d, got %d", usaNamesMass, actualTotal)
	}

}

// readUSANamesV2Document reads a real asset and verifies its canonical bytes.
func readUSANamesV2Document(t *testing.T, path string, document any) {

	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected real v2 asset, got %v", err)
	}
	err = decodeInternationalJSON(t.Context(), bytes.NewReader(encoded), document)
	if err != nil {
		t.Fatalf("expected closed v2 JSON, got %v", err)
	}
	canonical, err := internationalCanonical(document)
	if err != nil {
		t.Fatalf("expected canonical v2 JSON, got %v", err)
	}
	if !bytes.Equal(canonical, encoded) || sha256.Sum256(canonical) != sha256.Sum256(encoded) {
		t.Fatal("expected canonical delivered bytes, got mismatch")
	}

}

// usaNamesV2SeedForOutput inverts the frozen SplitMix64 permutation in tests.
func usaNamesV2SeedForOutput(output uint64) uint64 {
	output ^= output>>31 ^ output>>62
	output *= 0x319642b2d24d8ec3
	output ^= output>>27 ^ output>>54
	output *= 0x96de1b173f119089
	output ^= output>>30 ^ output>>60
	return output - 0x9e3779b97f4a7c15
}

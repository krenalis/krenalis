// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type testCountryGenerator struct {
	namespace IdentityNamespace
}

func (g testCountryGenerator) Person(index PersonIndex) (Person, error) {
	id, err := PersonIDForIndex(g.namespace, index)
	if err != nil {
		return Person{}, err
	}
	return Person{ID: id, FirstName: "Camille", LastName: "Martin", Email: "camille@example.test",
		Phone: "+33123456789", Address: Address{Country: "FR"}}, nil
}

type failingCSVWriter struct{}

func (failingCSVWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

type countingCSVWriter struct{ bytes int }

func (w *countingCSVWriter) Write(p []byte) (int, error) {
	w.bytes += len(p)
	return len(p), nil
}

func mustSourceWorld(t *testing.T) *SourceWorld {
	t.Helper()
	world, err := NewSourceWorld(mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed), []CountryShare{
		{Code: "IT", Version: MarketDataVersion, Weight: 1},
	})
	if err != nil {
		t.Fatalf("expected source world, got %v", err)
	}
	return world
}

func mustSource(t *testing.T, world *SourceWorld, id string, coverage, duplicate uint64) *SourceInstance {
	t.Helper()
	source, err := NewSourceInstance(world, SourceInstanceConfig{
		ID: id, Version: "demo-v1", CoverageNumerator: coverage, CoverageDenominator: 100,
		DuplicateNumerator: duplicate, DuplicateDenominator: 100,
	})
	if err != nil {
		t.Fatalf("expected source instance, got %v", err)
	}
	return source
}

// TestSourceSimulationItaly checks exact Layer 1 delegation.
func TestSourceSimulationItaly(t *testing.T) {
	base := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	world, err := NewSourceWorld(base, []CountryShare{{Code: "IT", Version: MarketDataVersion, Weight: 1}})
	if err != nil {
		t.Fatalf("expected IT source world, got %v", err)
	}
	for _, index := range []PersonIndex{1, 42, 1024, Layer1MaxPersonIndex} {
		got, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected country Person at %d, got %v", index, err)
		}
		want, err := base.Person(index)
		if err != nil {
			t.Fatalf("expected Layer 1 Person at %d, got %v", index, err)
		}
		if got.Country != "IT" || !reflect.DeepEqual(got.Person, want) {
			t.Fatalf("expected exact IT Layer 1 Person at %d, got %#v", index, got)
		}
	}
}

// TestSourceSimulationCountries checks selection and a test generator.
func TestSourceSimulationCountries(t *testing.T) {

	base := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	fake := testCountryGenerator{namespace: base.namespace}
	shares := []CountryShare{
		{Code: "FR", Version: "test-v1", Weight: 1, Generator: fake},
		{Code: "IT", Version: MarketDataVersion, Weight: 1},
	}
	world, err := NewSourceWorld(base, shares)
	if err != nil {
		t.Fatalf("expected mixed source world, got %v", err)
	}
	slices.Reverse(shares)
	reversed, err := NewSourceWorld(base, shares)
	if err != nil {
		t.Fatalf("expected reversed source world, got %v", err)
	}
	seen := map[string]bool{}
	for _, index := range []PersonIndex{19, 3, 71, 1, 42, 500, 2, 19, 3} {
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected mixed Person at %d, got %v", index, err)
		}
		other, err := reversed.Person(index)
		if err != nil {
			t.Fatalf("expected reversed Person at %d, got %v", index, err)
		}
		if !reflect.DeepEqual(person, other) {
			t.Fatalf("expected order-independent Person at %d, got %#v and %#v", index, person, other)
		}
		seen[person.Country] = true
	}
	if !seen["IT"] || !seen["FR"] {
		t.Fatalf("expected both country generators, got %v", seen)
	}

	source := mustSource(t, world, "mixed", 100, 0)
	for index := PersonIndex(1); index <= 40; index++ {
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected canonical Person, got %v", err)
		}
		records, err := source.Records(index)
		if err != nil {
			t.Fatalf("expected source record, got %v", err)
		}
		truth, err := world.Oracle(source, index)
		if err != nil {
			t.Fatalf("expected Oracle record, got %v", err)
		}
		if len(records) != 1 || len(truth) != 1 || truth[0].PersonID != person.Person.ID ||
			truth[0].SourceRecordID != records[0].ID || *records[0].FirstName != person.Person.FirstName {
			t.Fatalf("expected shared source/Oracle path at %d, got %#v and %#v", index, records, truth)
		}
	}

	frWorld, err := NewSourceWorld(base, []CountryShare{{Code: "FR", Version: "test-v1", Weight: 1, Generator: fake}})
	if err != nil {
		t.Fatalf("expected French source world, got %v", err)
	}
	duplicated := mustSource(t, frWorld, "french", 100, 100)
	records, err := duplicated.Records(1)
	if err != nil {
		t.Fatalf("expected French source records, got %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two French observations, got %d", len(records))
	}
	for _, record := range records {
		if record.Country == nil || *record.Country != "FR" {
			t.Fatalf("expected observed French address country, got %#v", record.Country)
		}
	}

}

// TestSourceSimulationMembership checks all two-source subsets and count cases.
func TestSourceSimulationMembership(t *testing.T) {

	world := mustSourceWorld(t)
	a := mustSource(t, world, "a", 60, 25)
	b := mustSource(t, world, "b", 55, 25)
	seen := map[string]PersonIndex{}
	counts := map[int]bool{}
	for index := PersonIndex(1); index <= 1000; index++ {
		ar, err := a.Records(index)
		if err != nil {
			t.Fatalf("expected A records at %d, got %v", index, err)
		}
		br, err := b.Records(index)
		if err != nil {
			t.Fatalf("expected B records at %d, got %v", index, err)
		}
		key := ""
		if len(ar) > 0 {
			key += "A"
		}
		if len(br) > 0 {
			key += "B"
		}
		seen[key] = index
		counts[len(ar)] = true
		counts[len(br)] = true
	}

	for _, key := range []string{"", "A", "B", "AB"} {
		if seen[key] == 0 {
			t.Fatalf("expected membership %q, got %v", key, seen)
		}
	}
	for _, count := range []int{0, 1, 2} {
		if !counts[count] {
			t.Fatalf("expected record count %d, got %v", count, counts)
		}
	}

}

// TestSourceSimulationDuplicates checks local IDs and shared truth.
func TestSourceSimulationDuplicates(t *testing.T) {

	world := mustSourceWorld(t)
	a := mustSource(t, world, "a", 100, 100)
	b := mustSource(t, world, "b", 100, 100)
	for _, source := range []*SourceInstance{a, b} {
		records, err := source.Records(42)
		if err != nil {
			t.Fatalf("expected duplicate records, got %v", err)
		}
		truth, err := world.Oracle(source, 42)
		if err != nil {
			t.Fatalf("expected duplicate Oracle records, got %v", err)
		}
		if len(records) != 2 || len(truth) != 2 || records[0].ID == records[1].ID ||
			truth[0].PersonID != truth[1].PersonID || truth[0].SourceRecordID != records[0].ID ||
			truth[1].SourceRecordID != records[1].ID || reflect.DeepEqual(records[0], records[1]) ||
			records[1].Email != nil || *records[0].Phone == *records[1].Phone ||
			*records[1].FirstName != strings.ToUpper(*records[0].FirstName) {
			t.Fatalf("expected distinct imperfect duplicates with shared truth, got %#v and %#v", records, truth)
		}
	}

	ar, err := a.Records(42)
	if err != nil {
		t.Fatalf("expected A records, got %v", err)
	}
	br, err := b.Records(42)
	if err != nil {
		t.Fatalf("expected B records, got %v", err)
	}
	if ar[0].ID == br[0].ID || !strings.HasPrefix(ar[0].ID, "sr_v2_a_") ||
		!strings.HasPrefix(br[0].ID, "sr_v2_b_") {
		t.Fatalf("expected source-separated local IDs, got %q and %q", ar[0].ID, br[0].ID)
	}

}

// TestSourceRecordIDNoAlgebraicJoin exercises the former public unmix64
// shortcut on every eight-byte window of the token. It checks grouping and
// one-pair cross-source calibration using only ordinary IDs.
func TestSourceRecordIDNoAlgebraicJoin(t *testing.T) {

	world := mustSourceWorld(t)
	a := mustSource(t, world, "a", 100, 100)
	b := mustSource(t, world, "b", 100, 100)
	decode := func(id string) []byte {
		t.Helper()
		separator := strings.LastIndexByte(id, '_')
		if separator < 0 {
			t.Fatalf("expected token separator in %q, got none", id)
		}
		token, err := hex.DecodeString(id[separator+1:])
		if err != nil {
			t.Fatalf("expected hexadecimal token in %q, got %v", id, err)
		}
		if len(token) < 8 {
			t.Fatalf("expected at least eight token bytes in %q, got %d", id, len(token))
		}
		return token
	}
	group := func(token []byte, offset int) uint64 {
		return unmix64(binary.BigEndian.Uint64(token[offset:offset+8])) >> 1
	}

	aTokens := make([][2][]byte, 0, 200)
	bTokens := make([][2][]byte, 0, 200)
	for index := PersonIndex(1); index <= 200; index++ {
		ar, err := a.Records(index)
		if err != nil {
			t.Fatalf("expected A records at %d, got %v", index, err)
		}
		br, err := b.Records(index)
		if err != nil {
			t.Fatalf("expected B records at %d, got %v", index, err)
		}
		if len(ar) != 2 || len(br) != 2 {
			t.Fatalf("expected two records per source at %d, got %d and %d", index, len(ar), len(br))
		}
		aTokens = append(aTokens, [2][]byte{decode(ar[0].ID), decode(ar[1].ID)})
		bTokens = append(bTokens, [2][]byte{decode(br[0].ID), decode(br[1].ID)})
	}

	for _, test := range []struct {
		name   string
		tokens [][2][]byte
	}{{"A", aTokens}, {"B", bTokens}} {
		t.Run("duplicates "+test.name, func(t *testing.T) {
			for offset := 0; offset <= len(test.tokens[0][0])-8; offset++ {
				allGrouped := true
				for _, pair := range test.tokens {
					if group(pair[0], offset) != group(pair[1], offset) {
						allGrouped = false
						break
					}
				}
				if allGrouped {
					t.Fatalf("expected no ID-only duplicate grouping at byte %d, got all 200 pairs grouped", offset)
				}
			}
		})
	}
	t.Run("cross source", func(t *testing.T) {
		for aOffset := 0; aOffset <= len(aTokens[0][0])-8; aOffset++ {
			for bOffset := 0; bOffset <= len(bTokens[0][0])-8; bOffset++ {
				delta := group(aTokens[0][0], aOffset) ^ group(bTokens[0][0], bOffset)
				allJoined := true
				for i := 1; i < len(aTokens); i++ {
					if group(aTokens[i][0], aOffset)^group(bTokens[i][0], bOffset) != delta {
						allJoined = false
						break
					}
				}
				if allJoined {
					t.Fatalf("expected no one-pair cross-source delta at bytes %d/%d, got all 200 pairs joined",
						aOffset, bOffset)
				}
			}
		}
	})

}

// TestSourceRecordIDDomainEdges checks both ends of the Layer 1 domain.
func TestSourceRecordIDDomainEdges(t *testing.T) {

	world := mustSourceWorld(t)
	for _, source := range []*SourceInstance{
		mustSource(t, world, "a", 100, 100),
		mustSource(t, world, "b", 100, 100),
	} {
		seen := map[string]struct{}{}
		for _, index := range []PersonIndex{1, 2, Layer1MaxPersonIndex - 1, Layer1MaxPersonIndex} {
			person, err := world.Person(index)
			if err != nil {
				t.Fatalf("expected canonical Person at %d, got %v", index, err)
			}
			records, err := source.Records(index)
			if err != nil {
				t.Fatalf("expected records at %d, got %v", index, err)
			}
			truth, err := world.Oracle(source, index)
			if err != nil {
				t.Fatalf("expected Oracle at %d, got %v", index, err)
			}
			if len(records) != 2 || len(truth) != 2 {
				t.Fatalf("expected two records and mappings at %d, got %d and %d", index, len(records), len(truth))
			}
			for ordinal, record := range records {
				if _, exists := seen[record.ID]; exists {
					t.Fatalf("expected unique ID in source %s, got duplicate %q", source.ID(), record.ID)
				}
				seen[record.ID] = struct{}{}
				if truth[ordinal].SourceInstanceID != source.ID() || truth[ordinal].SourceRecordID != record.ID ||
					truth[ordinal].PersonID != person.Person.ID {
					t.Fatalf("expected Oracle key for source %s at %d, got %#v", source.ID(), index, truth[ordinal])
				}
			}
		}
	}

}

type sourceResult struct {
	source string
	index  PersonIndex
	rows   []SourceRecord
	truth  []OracleRecord
	err    error
}

// TestSourceSimulationOrderAndConcurrency checks logical results across visits.
func TestSourceSimulationOrderAndConcurrency(t *testing.T) {

	world := mustSourceWorld(t)
	sources := []*SourceInstance{mustSource(t, world, "a", 60, 25), mustSource(t, world, "b", 55, 25)}
	baseline := map[string]sourceResult{}
	key := func(source string, index PersonIndex) string {
		return source + "/" + strconv.FormatUint(uint64(index), 10)
	}
	for index := PersonIndex(1); index <= 100; index++ {
		for _, source := range sources {
			rows, err := source.Records(index)
			if err != nil {
				t.Fatalf("expected baseline records, got %v", err)
			}
			truth, err := world.Oracle(source, index)
			if err != nil {
				t.Fatalf("expected baseline Oracle, got %v", err)
			}
			baseline[key(source.ID(), index)] = sourceResult{rows: rows, truth: truth}
		}
	}

	check := func(result sourceResult) {
		t.Helper()
		if result.err != nil {
			t.Fatalf("expected deterministic result, got %v", result.err)
		}
		want := baseline[key(result.source, result.index)]
		if !reflect.DeepEqual(result.rows, want.rows) || !reflect.DeepEqual(result.truth, want.truth) {
			t.Fatalf("expected logical result for %s/%d, got %#v and %#v", result.source, result.index,
				result.rows, result.truth)
		}
	}

	for _, index := range []PersonIndex{1, 42, 100} {
		for _, source := range sources {
			rows, err := source.Records(index)
			if err != nil {
				t.Fatalf("expected repeated records, got %v", err)
			}
			truth, err := world.Oracle(source, index)
			if err != nil {
				t.Fatalf("expected repeated Oracle, got %v", err)
			}
			check(sourceResult{source: source.ID(), index: index, rows: rows, truth: truth})
		}
	}

	for start := PersonIndex(100); start > 0; {
		end := start
		if start > 12 {
			start -= 12
		} else {
			start = 0
		}
		for index := end; index > start; index-- {
			for i := len(sources) - 1; i >= 0; i-- {
				source := sources[i]
				rows, err := source.Records(index)
				if err != nil {
					t.Fatalf("expected reordered records, got %v", err)
				}
				truth, err := world.Oracle(source, index)
				if err != nil {
					t.Fatalf("expected reordered Oracle, got %v", err)
				}
				check(sourceResult{source: source.ID(), index: index, rows: rows, truth: truth})
			}
		}
	}

	results := make(chan sourceResult, 200)
	var group sync.WaitGroup
	for index := PersonIndex(1); index <= 100; index++ {
		for _, source := range sources {
			group.Add(1)
			go func(source *SourceInstance, index PersonIndex) {
				defer group.Done()
				rows, err := source.Records(index)
				if err != nil {
					results <- sourceResult{source: source.ID(), index: index, err: err}
					return
				}
				truth, err := world.Oracle(source, index)
				results <- sourceResult{source: source.ID(), index: index, rows: rows, truth: truth, err: err}
			}(source, index)
		}
	}
	group.Wait()
	close(results)
	for result := range results {
		check(result)
	}

}

// TestSourceSimulationCSVAndOracle exercises two independent outputs and truth.
func TestSourceSimulationCSVAndOracle(t *testing.T) {

	world := mustSourceWorld(t)
	sources := []*SourceInstance{mustSource(t, world, "a", 60, 25), mustSource(t, world, "b", 55, 25)}
	outputs := [2]bytes.Buffer{}
	allTruth := []OracleRecord{}
	for i, source := range sources {
		csvWriter, err := NewSourceCSVWriter(&outputs[i], world.base.faceCatalog)
		if err != nil {
			t.Fatalf("expected CSV writer, got %v", err)
		}
		for index := PersonIndex(1); index <= 50; index++ {
			records, err := source.Records(index)
			if err != nil {
				t.Fatalf("expected generated records, got %v", err)
			}
			for _, record := range records {
				err = csvWriter.Write(t.Context(), record)
				if err != nil {
					t.Fatalf("expected CSV row, got %v", err)
				}
			}
			truth, err := world.Oracle(source, index)
			if err != nil {
				t.Fatalf("expected separate Oracle, got %v", err)
			}
			allTruth = append(allTruth, truth...)
		}
		err = csvWriter.Flush(t.Context())
		if err != nil {
			t.Fatalf("expected flushed CSV, got %v", err)
		}
		parsed, err := csv.NewReader(bytes.NewReader(outputs[i].Bytes())).ReadAll()
		if err != nil {
			t.Fatalf("expected parseable CSV, got %v", err)
		}
		if !reflect.DeepEqual(parsed[0], sourceCSVHeader) || len(parsed) < 2 {
			t.Fatalf("expected fixed CSV header and records, got %#v", parsed)
		}
		for _, row := range parsed[1:] {
			if len(row) != 7 || !strings.HasPrefix(row[0], "sr_v2_"+source.ID()+"_") {
				t.Fatalf("expected ordinary source-local row, got %#v", row)
			}
		}
	}

	if len(allTruth) == 0 || bytes.Contains(outputs[0].Bytes(), []byte(allTruth[0].PersonID)) ||
		bytes.Contains(outputs[1].Bytes(), []byte(allTruth[0].PersonID)) {
		t.Fatalf("expected hidden Oracle truth, got %d mappings", len(allTruth))
	}

	literal := "literal@example.test"
	var received bytes.Buffer
	adapter, err := NewSourceCSVWriter(&received, world.base.faceCatalog)
	if err != nil {
		t.Fatalf("expected standalone CSV adapter, got %v", err)
	}
	err = adapter.Write(t.Context(), SourceRecord{ID: "local", Email: &literal})
	if err != nil {
		t.Fatalf("expected supplied logical row, got %v", err)
	}
	err = adapter.Flush(t.Context())
	if err != nil {
		t.Fatalf("expected supplied row flush, got %v", err)
	}
	if !strings.Contains(received.String(), "local,,,literal@example.test,,,") {
		t.Fatalf("expected unchanged supplied observation and empty absences, got %q", received.String())
	}

}

func TestSourceCSVPhotoURL(t *testing.T) {

	world := mustSourceWorld(t)
	source := mustSource(t, world, "photos", 100, 100)
	records, err := source.Records(1)
	if err != nil {
		t.Fatalf("expected photo records, got %v", err)
	}

	person, err := world.Person(1)
	if err != nil {
		t.Fatalf("expected canonical person, got %v", err)
	}
	for _, record := range records {
		if record.PhotoID == nil || *record.PhotoID != person.Person.PhotoID {
			t.Fatalf("expected observed photo %q, got %#v", person.Person.PhotoID, record.PhotoID)
		}
	}

	asset, err := world.base.faceCatalog.Asset(person.Person.PhotoID, PhotoSize256)
	if err != nil {
		t.Fatalf("expected 256 pixel asset, got %v", err)
	}

	var output bytes.Buffer
	writer, err := NewSourceCSVWriter(&output, world.base.faceCatalog)
	if err != nil {
		t.Fatalf("expected photo CSV writer, got %v", err)
	}
	for _, record := range records {
		err = writer.Write(t.Context(), record)
		if err != nil {
			t.Fatalf("expected photo URL row, got %v", err)
		}
	}
	err = writer.Write(t.Context(), SourceRecord{ID: "without-photo"})
	if err != nil {
		t.Fatalf("expected absent photo row, got %v", err)
	}
	missing := "face-unavailable"
	err = writer.Write(t.Context(), SourceRecord{ID: "bad-photo", PhotoID: &missing})
	if err != nil {
		if !errors.Is(err, ErrInvalidPhotoAsset) {
			t.Fatalf("expected unresolved photo error, got %v", err)
		}
	}
	if err == nil {
		t.Fatalf("expected unresolved photo error, got nil")
	}

	err = writer.Flush(t.Context())
	if err != nil {
		t.Fatalf("expected flushed photo CSV, got %v", err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatalf("expected parseable photo CSV, got %v", err)
	}

	if len(rows) != 4 || rows[1][5] != asset.Path || rows[2][5] != rows[1][5] || rows[3][5] != "" ||
		rows[3][6] != "" || strings.HasPrefix(rows[1][5], "//") {
		t.Fatalf("expected two root-relative asset paths and empty absent cells, got %#v", rows)
	}

	request := httptest.NewRequest(http.MethodGet, "http://photos.example.test"+rows[1][5], nil)
	response := httptest.NewRecorder()
	world.base.faceCatalog.Handler().ServeHTTP(response, request)
	contentSHA := sha256.Sum256(response.Body.Bytes())
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "image/webp" ||
		hex.EncodeToString(contentSHA[:]) != asset.ContentSHA256 {
		t.Fatalf("expected CSV photo path to resolve to verified photo bytes, got %d %s %x",
			response.Code, response.Header().Get("Content-Type"), contentSHA)
	}
	for _, bounded := range []struct {
		photoID string
		lookup  bool
	}{
		{strings.Repeat("a", 128), true},
		{strings.Repeat("a", 129), false},
	} {
		err = writer.Write(t.Context(), SourceRecord{ID: "bounded-photo", PhotoID: &bounded.photoID})
		if err != nil {
			if errors.Is(err, ErrInvalidPhotoAsset) != bounded.lookup {
				t.Fatalf("expected lookup error %t for %d-byte photo reference, got %v",
					bounded.lookup, len(bounded.photoID), err)
			}
			continue
		}
		t.Fatalf("expected invalid %d-byte photo reference, got nil", len(bounded.photoID))
	}

	_, err = NewSourceCSVWriter(&bytes.Buffer{}, nil)
	if err != nil {
		return
	}
	t.Fatal("expected invalid nil catalog, got nil")

}

// TestSourceSimulationValidationAndStreaming checks limits and writer failures.
func TestSourceSimulationValidationAndStreaming(t *testing.T) {

	base := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	good := CountryShare{Code: "IT", Version: MarketDataVersion, Weight: 1}
	for _, shares := range [][]CountryShare{
		nil,
		{{Code: "FR", Version: "test-v1", Weight: 1}},
		{{Code: "IT", Version: MarketDataVersion, Weight: 0}},
		{{Code: "IT", Version: MarketDataVersion, Weight: math.MaxUint64},
			{Code: "FR", Version: "test-v1", Weight: 1, Generator: testCountryGenerator{base.namespace}}},
		{good, good},
		{{Code: "It", Version: MarketDataVersion, Weight: 1}},
	} {
		_, err := NewSourceWorld(base, shares)
		if err == nil {
			t.Fatalf("expected invalid country configuration, got %#v", shares)
		}
	}
	tooMany := make([]CountryShare, 17)
	for i := range tooMany {
		tooMany[i] = good
	}
	_, err := NewSourceWorld(base, tooMany)
	if err == nil {
		t.Fatal("expected country-count limit error, got nil")
	}

	_, err = NewSourceWorld(nil, []CountryShare{good})
	if err == nil {
		t.Fatal("expected nil World error, got nil")
	}

	world := mustSourceWorld(t)
	for _, config := range []SourceInstanceConfig{
		{},
		{ID: "Bad", Version: "v1", CoverageDenominator: 1, DuplicateDenominator: 1},
		{ID: "a", Version: "", CoverageDenominator: 1, DuplicateDenominator: 1},
		{ID: "a", Version: "v1", CoverageNumerator: 2, CoverageDenominator: 1, DuplicateDenominator: 1},
		{ID: "a", Version: "v1", CoverageDenominator: 1, DuplicateNumerator: 2, DuplicateDenominator: 1},
	} {
		_, err := NewSourceInstance(world, config)
		if err == nil {
			t.Fatalf("expected invalid source configuration, got %#v", config)
		}
	}

	source := mustSource(t, world, "a", 100, 0)
	for _, index := range []PersonIndex{0, Layer1MaxPersonIndex + 1} {
		_, err := source.Records(index)
		if err == nil {
			t.Fatalf("expected invalid source index %d, got nil", index)
		}
		_, err = world.Oracle(source, index)
		if err == nil {
			t.Fatalf("expected invalid Oracle index %d, got nil", index)
		}
	}
	maximum := mustSource(t, world, "maximum", 100, 100)
	records, err := maximum.Records(Layer1MaxPersonIndex)
	if err != nil || len(records) != 2 || records[0].ID == records[1].ID {
		t.Fatalf("expected two distinct records at maximum index, got %#v and %v", records, err)
	}

	other := mustSourceWorld(t)
	_, err = other.Oracle(source, 1)
	if err == nil {
		t.Fatal("expected cross-World Oracle rejection, got nil")
	}

	failing, err := NewSourceCSVWriter(failingCSVWriter{}, world.base.faceCatalog)
	if err != nil {
		t.Fatalf("expected buffered CSV writer, got %v", err)
	}
	err = failing.Flush(t.Context())
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("expected propagated writer error, got %v", err)
	}

	counting := &countingCSVWriter{}
	adapter, err := NewSourceCSVWriter(counting, world.base.faceCatalog)
	if err != nil {
		t.Fatalf("expected counting CSV writer, got %v", err)
	}
	for index := PersonIndex(1); index <= 5; index++ {
		records, err := source.Records(index)
		if err != nil {
			t.Fatalf("expected incremental records, got %v", err)
		}
		for _, record := range records {
			err = adapter.Write(t.Context(), record)
			if err != nil {
				t.Fatalf("expected incremental CSV write, got %v", err)
			}
		}
		err = adapter.Flush(t.Context())
		if err != nil || counting.bytes == 0 {
			t.Fatalf("expected emitted bytes before next person, got %d and %v", counting.bytes, err)
		}
	}

	bad := strings.Repeat("x", 1025)
	err = adapter.Write(t.Context(), SourceRecord{ID: "local", Email: &bad})
	if err == nil {
		t.Fatal("expected bounded CSV field error, got nil")
	}

}

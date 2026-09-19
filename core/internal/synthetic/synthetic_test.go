// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package synthetic

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/json"
)

// completeCatalogFixture copies only the two selected images into a complete
// catalog with contiguous IDs, as required by the public loader.
func completeCatalogFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "..", "tools", "fakedata", "testdata", "fakefacegen-v1-selected")
	dir := t.TempDir()
	var catalog map[string]any
	var specs []map[string]any
	var manifest map[string]any
	for _, item := range []struct {
		name string
		out  any
	}{{"catalog.json", &catalog}, {"specs.json", &specs}, {"manifest.json", &manifest}} {
		data, err := os.ReadFile(filepath.Join(root, item.name))
		if err != nil {
			t.Fatalf("expected fixture metadata, got %v", err)
		}
		err = json.Unmarshal(data, item.out)
		if err != nil {
			t.Fatalf("expected parsed fixture metadata, got %v", err)
		}
	}
	catalog["count"] = 2
	selected := []string{"face-000013", "face-000037"}
	newSpecs := make([]map[string]any, 0, 2)
	newAssets := make([]map[string]any, 0, 2)
	assets := manifest["assets"].([]any)
	for i, sourceID := range selected {
		id := fmt.Sprintf("face-%06d", i+1)
		for _, spec := range specs {
			if spec["id"] == sourceID {
				spec["id"] = id
				newSpecs = append(newSpecs, spec)
				break
			}
		}
		for _, item := range assets {
			asset := item.(map[string]any)
			if asset["id"] == sourceID {
				asset["id"] = id
				asset["spec_id"] = id
				asset["file"] = "masters/" + id + ".webp"
				newAssets = append(newAssets, asset)
				break
			}
		}
		for _, size := range []int{64, 128, 256, 512, 1024} {
			folder := filepath.Join("derived", strconv.Itoa(size))
			if size == 1024 {
				folder = "masters"
			}
			data, err := os.ReadFile(filepath.Join(root, folder, sourceID+".webp"))
			if err != nil {
				t.Fatalf("expected source asset, got %v", err)
			}
			path := filepath.Join(dir, folder)
			err = os.MkdirAll(path, 0o755)
			if err != nil {
				t.Fatalf("expected asset directory, got %v", err)
			}
			err = os.WriteFile(filepath.Join(path, id+".webp"), data, 0o644)
			if err != nil {
				t.Fatalf("expected copied asset, got %v", err)
			}
		}
	}
	manifest["assets"] = newAssets
	for _, item := range []struct {
		name  string
		value any
	}{{"catalog.json", catalog}, {"specs.json", newSpecs}, {"manifest.json", manifest}} {
		data, err := json.Marshal(item.value)
		if err != nil {
			t.Fatalf("expected serialized fixture metadata, got %v", err)
		}
		err = os.WriteFile(filepath.Join(dir, item.name), data, 0o644)
		if err != nil {
			t.Fatalf("expected fixture metadata, got %v", err)
		}
	}
	return dir
}

func testScenario(t *testing.T) (*Scenario, Config, *fakedata.FaceCatalog) {
	t.Helper()
	catalog, err := fakedata.LoadFaceCatalog(t.Context(), completeCatalogFixture(t))
	if err != nil {
		t.Fatalf("expected complete catalog, got %v", err)
	}
	config := Config{
		Namespace: "synthetic-test", Generation: 1, Seed: 726381, ReferenceDate: "2026-01-01", PersonCount: 20,
		PhotoOrigin: "https://photos.example.test", SourceAID: "a", SourceAVersion: "demo-v1",
		SourceACoverageNumerator: 1, SourceACoverageDenominator: 1,
		SourceADuplicateNumerator: 1, SourceADuplicateDenominator: 1,
		SourceBID: "b", SourceBVersion: "demo-v1", SourceBCoverageNumerator: 1,
		SourceBCoverageDenominator: 1, SourceBDuplicateNumerator: 0, SourceBDuplicateDenominator: 1,
	}
	scenario, err := New(config, catalog)
	if err != nil {
		t.Fatalf("expected scenario, got %v", err)
	}
	return scenario, config, catalog
}

func TestScenarioValidationAndPaths(t *testing.T) {
	scenario, config, catalog := testScenario(t)
	for _, name := range []string{"", "../customers-a.csv", "//customers-a.csv", "customers-c.csv", "customers-a.csv/"} {
		_, err := Path(name)
		if err == nil {
			t.Fatalf("expected rejected path %q, got nil", name)
		}
		_, _, err = scenario.Reader(t.Context(), name)
		if err == nil {
			t.Fatalf("expected rejected reader path %q, got nil", name)
		}
	}
	path, err := Path("/customers-b.csv")
	if err != nil || path != "customers-b.csv" {
		t.Fatalf("expected B path, got %q, %v", path, err)
	}
	limit := config
	limit.Seed = maxExactInteger
	limit.SourceACoverageNumerator, limit.SourceACoverageDenominator = maxExactInteger, maxExactInteger
	limit.SourceADuplicateNumerator, limit.SourceADuplicateDenominator = maxExactInteger, maxExactInteger
	limit.SourceBCoverageNumerator, limit.SourceBCoverageDenominator = maxExactInteger, maxExactInteger
	limit.SourceBDuplicateNumerator, limit.SourceBDuplicateDenominator = maxExactInteger, maxExactInteger
	_, err = New(limit, catalog)
	if err != nil {
		t.Fatalf("expected accepted exact-integer limits, got %v", err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.Seed = maxExactInteger + 1 },
		func(c *Config) { c.SourceACoverageNumerator = maxExactInteger + 1 },
		func(c *Config) { c.SourceACoverageDenominator = maxExactInteger + 1 },
		func(c *Config) { c.SourceADuplicateNumerator = maxExactInteger + 1 },
		func(c *Config) { c.SourceADuplicateDenominator = maxExactInteger + 1 },
		func(c *Config) { c.SourceBCoverageNumerator = maxExactInteger + 1 },
		func(c *Config) { c.SourceBCoverageDenominator = maxExactInteger + 1 },
		func(c *Config) { c.SourceBDuplicateNumerator = maxExactInteger + 1 },
		func(c *Config) { c.SourceBDuplicateDenominator = maxExactInteger + 1 },
	} {
		invalid := limit
		change(&invalid)
		_, err := New(invalid, catalog)
		if err == nil {
			t.Fatal("expected exact-integer limit rejection, got nil")
		}
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.PersonCount = fakedata.Layer1MaxPersonIndex + 1 },
		func(c *Config) { c.SourceBID = c.SourceAID },
		func(c *Config) { c.SourceACoverageDenominator = 0 },
		func(c *Config) { c.SourceBDuplicateNumerator = 2 },
		func(c *Config) { c.ReferenceDate = "2031-01-01" },
		func(c *Config) { c.PhotoOrigin = "https://" + strings.Repeat("a", 254) },
	} {
		invalid := config
		change(&invalid)
		_, err := New(invalid, catalog)
		if err == nil {
			t.Fatalf("expected invalid scenario error, got nil")
		}
	}
	_, err = New(config, nil)
	if err == nil {
		t.Fatal("expected missing catalog error, got nil")
	}
}

func TestScenarioReaderMatchesGeneration(t *testing.T) {
	scenario, config, catalog := testScenario(t)
	first, modified, err := scenario.Reader(t.Context(), "customers-a.csv")
	if err != nil {
		t.Fatalf("expected first reader, got %v", err)
	}
	defer first.Close()
	second, modified2, err := scenario.Reader(t.Context(), "/customers-a.csv")
	if err != nil {
		t.Fatalf("expected second reader, got %v", err)
	}
	defer second.Close()
	if !modified.Equal(modified2) || !modified.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected static snapshot date, got %v and %v", modified, modified2)
	}
	part := make([]byte, 7)
	_, err = first.Read(part)
	if err != nil {
		t.Fatalf("expected first bytes, got %v", err)
	}
	secondBytes, err := io.ReadAll(second)
	if err != nil {
		t.Fatalf("expected second bytes, got %v", err)
	}
	firstTail, err := io.ReadAll(first)
	if err != nil {
		t.Fatalf("expected first tail, got %v", err)
	}
	if !bytes.Equal(append(part, firstTail...), secondBytes) {
		t.Fatal("expected independent readers, got different bytes")
	}
	expected := expectedCSV(t, config, catalog, config.sourceA())
	if !bytes.Equal(secondBytes, expected) {
		t.Fatal("expected fakedata writer bytes, got different bytes")
	}
	rows, err := csv.NewReader(bytes.NewReader(secondBytes)).ReadAll()
	if err != nil || len(rows) != 41 {
		t.Fatalf("expected header and 40 rows, got %d, %v", len(rows), err)
	}
}

func TestScenarioReaderEmptyCancellationAndBound(t *testing.T) {
	scenario, config, catalog := testScenario(t)
	config.PersonCount = 0
	scenario, err := New(config, catalog)
	if err != nil {
		t.Fatalf("expected empty scenario, got %v", err)
	}
	r, _, err := scenario.Reader(t.Context(), "customers-b.csv")
	if err != nil {
		t.Fatalf("expected empty reader, got %v", err)
	}
	data, err := io.ReadAll(r)
	if err != nil || string(data) != "source_record_id,first_name,last_name,email,phone,photo_url\n" {
		t.Fatalf("expected header-only CSV, got %q, %v", data, err)
	}
	_ = r.Close()
	_, err = r.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("expected closed reader error, got nil")
	}
	config.PersonCount = 20
	config.SourceBCoverageNumerator = 0
	scenario, err = New(config, catalog)
	if err != nil {
		t.Fatalf("expected zero-coverage scenario, got %v", err)
	}
	r, _, err = scenario.Reader(t.Context(), "customers-b.csv")
	if err != nil {
		t.Fatalf("expected zero-coverage reader, got %v", err)
	}
	data, err = io.ReadAll(r)
	if err != nil || string(data) != "source_record_id,first_name,last_name,email,phone,photo_url\n" {
		t.Fatalf("expected header-only zero-coverage CSV, got %q, %v", data, err)
	}
	_ = r.Close()
	config.SourceBCoverageNumerator = 1
	config.PersonCount = fakedata.Layer1MaxPersonIndex
	scenario, err = New(config, catalog)
	if err != nil {
		t.Fatalf("expected bounded scenario, got %v", err)
	}
	r, _, err = scenario.Reader(t.Context(), "customers-a.csv")
	if err != nil {
		t.Fatalf("expected bounded reader, got %v", err)
	}
	reader := r.(*csvReader)
	defer reader.Close()
	early, _, err := scenario.Reader(t.Context(), "customers-a.csv")
	if err != nil {
		t.Fatalf("expected early-close reader, got %v", err)
	}
	_, err = early.Read(make([]byte, 8))
	if err != nil {
		t.Fatalf("expected early bytes, got %v", err)
	}
	err = early.Close()
	if err != nil {
		t.Fatalf("expected early close, got %v", err)
	}
	for range 2000 {
		_, err = reader.Read(make([]byte, 1))
		if err != nil {
			t.Fatalf("expected bounded read, got %v", err)
		}
		if reader.buffer.Len() > 10_000 || reader.next > 10 {
			t.Fatalf("expected bounded buffering and generation, got %d bytes and next %d", reader.buffer.Len(), reader.next)
		}
	}
	reader.next = fakedata.Layer1MaxPersonIndex + 1
	reader.count = reader.next
	reader.buffer.Reset()
	_, err = reader.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("expected generator error, got nil")
	}
	ctx, cancel := context.WithCancel(t.Context())
	r, _, err = scenario.Reader(ctx, "customers-b.csv")
	if err != nil {
		t.Fatalf("expected cancellable reader, got %v", err)
	}
	defer r.Close()
	cancel()
	_, err = r.Read(nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation on empty read, got %v", err)
	}
	_, err = r.Read(make([]byte, 1))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestScenarioConcurrentReaders(t *testing.T) {
	scenario, config, catalog := testScenario(t)
	expected := expectedCSV(t, config, catalog, config.sourceB())
	results := make([][]byte, 4)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, _, err := scenario.Reader(t.Context(), "customers-b.csv")
			if err != nil {
				t.Errorf("expected concurrent reader, got %v", err)
				return
			}
			defer r.Close()
			results[i], err = io.ReadAll(r)
			if err != nil {
				t.Errorf("expected concurrent bytes, got %v", err)
			}
		}()
	}
	wg.Wait()
	for _, data := range results {
		if !bytes.Equal(data, expected) {
			t.Fatal("expected identical concurrent bytes, got different bytes")
		}
	}
}

func expectedCSV(t *testing.T, config Config, catalog *fakedata.FaceCatalog, sourceConfig fakedata.SourceInstanceConfig) []byte {
	t.Helper()
	namespace, err := fakedata.NewIdentityNamespace(config.Namespace, config.Generation)
	if err != nil {
		t.Fatalf("expected namespace, got %v", err)
	}
	base, err := fakedata.NewWorld(fakedata.WorldConfig{IdentityNamespace: namespace, WorldSeed: config.Seed,
		ReferenceDate: config.ReferenceDate, FaceCatalog: catalog})
	if err != nil {
		t.Fatalf("expected World, got %v", err)
	}
	world, err := fakedata.NewSourceWorld(base, []fakedata.CountryShare{{Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1}})
	if err != nil {
		t.Fatalf("expected SourceWorld, got %v", err)
	}
	source, err := fakedata.NewSourceInstance(world, sourceConfig)
	if err != nil {
		t.Fatalf("expected SourceInstance, got %v", err)
	}
	var output bytes.Buffer
	writer, err := fakedata.NewSourceCSVWriter(&output, catalog, config.PhotoOrigin)
	if err != nil {
		t.Fatalf("expected writer, got %v", err)
	}
	for index := fakedata.PersonIndex(1); index <= config.PersonCount; index++ {
		records, err := source.Records(index)
		if err != nil {
			t.Fatalf("expected records, got %v", err)
		}
		for _, record := range records {
			err = writer.Write(t.Context(), record)
			if err != nil {
				t.Fatalf("expected written record, got %v", err)
			}
		}
	}
	err = writer.Flush(t.Context())
	if err != nil {
		t.Fatalf("expected flushed CSV, got %v", err)
	}
	return output.Bytes()
}

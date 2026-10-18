// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"
	"unsafe"

	"github.com/krenalis/krenalis/tools/fakedata/internal/usadiagnostic"
)

// usaSourceUSDSink keeps only bounded file positions for canonical-order
// hashes.
// Traversal order never becomes a sampling or source-observation input.
type usaSourceUSDSink struct {
	file      *os.File
	writer    *bufio.Writer
	positions []usaSourceUSDPosition
	offset    int64
}

func newUSASourceUSDSink(directory, name string, count int) (*usaSourceUSDSink, error) {
	file, err := os.Create(filepath.Join(directory, name))
	if err != nil {
		return nil, err
	}
	return &usaSourceUSDSink{file: file, writer: bufio.NewWriter(file),
		positions: make([]usaSourceUSDPosition, count)}, nil
}

func (s *usaSourceUSDSink) digest(ctx context.Context) (string, error) {

	err := s.writer.Flush()
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	buffer := make([]byte, usaSourceUSDRecordBytes)
	for _, position := range s.positions {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if position.size == 0 || position.size > len(buffer) {
			return "", fmt.Errorf("missing or oversized diagnostic record position")
		}
		n, err := s.file.ReadAt(buffer[:position.size], position.offset)
		if err != nil {
			return "", err
		}
		if n != position.size {
			return "", fmt.Errorf("incomplete diagnostic record read")
		}
		hash.Write(buffer[:n])
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (s *usaSourceUSDSink) write(slot int, encoded []byte) error {
	if slot < 0 || slot >= len(s.positions) || s.positions[slot].size != 0 ||
		len(encoded) == 0 || len(encoded) > usaSourceUSDRecordBytes {
		return fmt.Errorf("invalid or duplicate diagnostic output slot")
	}
	_, err := s.writer.Write(encoded)
	if err != nil {
		return err
	}
	s.positions[slot] = usaSourceUSDPosition{s.offset, len(encoded)}
	s.offset += int64(len(encoded))
	return nil
}

type usaSourceUSDPosition struct {
	offset int64
	size   int
}

func executeUSASourceUSDDiagnostic(ctx context.Context, options usadiagnostic.Options) error {

	provider, err := newUSASourceUSDProvider(options)
	if err != nil {
		return err
	}
	if options.People < 20000 || options.People > 50000 || options.Directory == "" {
		return fmt.Errorf("diagnostic output directory and 20000–50000 people are required")
	}
	started := time.Now()
	world, resources, err := provider.prepare(ctx)
	if err != nil {
		return err
	}
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	source, err := newUSASourceUSDAdapter(world, SourceInstanceConfig{
		ID: "usa-usd-diagnostic", Version: usaSourceUSDVersion,
		CoverageNumerator: 1, CoverageDenominator: 1, DuplicateDenominator: 1,
	})
	if err != nil {
		return err
	}
	runtime.GC()
	var adapted runtime.MemStats
	runtime.ReadMemStats(&adapted)
	resources["adapterHeapAllocDeltaBytes"] = int64(adapted.HeapAlloc) - int64(before.HeapAlloc)
	resources["adapterTotalAllocBytes"] = adapted.TotalAlloc - before.TotalAlloc
	resources["adapterMallocs"] = adapted.Mallocs - before.Mallocs
	resources["adapterShallowBytes"] = unsafe.Sizeof(*source) + unsafe.Sizeof(*source.source) +
		unsafe.Sizeof(*source.source.world)

	// A new directory prevents overwriting M1 or a previous diagnostic run.
	err = os.MkdirAll(filepath.Dir(options.Directory), 0o755)
	if err != nil {
		return err
	}
	err = os.Mkdir(options.Directory, 0o755)
	if err != nil {
		return err
	}
	err = os.Mkdir(filepath.Join(options.Directory, "hidden"), 0o755)
	if err != nil {
		return err
	}
	var sinks []*usaSourceUSDSink
	defer func() {
		for _, sink := range sinks {
			sink.file.Close()
		}
	}()
	for _, specification := range []struct {
		name  string
		count int
	}{
		{"records.jsonl", options.People}, {"hidden/oracle.jsonl", options.People},
		{".canonical-people.jsonl", options.People}, {"companies.tsv", 65536},
		{"sparse-records.jsonl", 1007}, {"hidden/sparse-oracle.jsonl", 1007},
	} {
		sink, err := newUSASourceUSDSink(options.Directory, specification.name, specification.count)
		if err != nil {
			return err
		}
		sinks = append(sinks, sink)
	}
	defer os.Remove(sinks[2].file.Name())
	resources["diagnosticFilePositionBytes"] = (2*options.People + options.People + 65536 + 2*1007) *
		int(unsafe.Sizeof(usaSourceUSDPosition{}))

	rosterStarted := time.Now()
	for step := range 65536 {
		coordinate := CompanyIndex(step + 1)
		if options.Order == "descending" {
			coordinate = CompanyIndex(65536 - step)
		}
		company, err := world.sampler.categorical.companies.company(coordinate)
		if err != nil {
			return err
		}
		encoded := fmt.Appendf(nil, "%d\t%s\t%s\t%d\t%s\n",
			coordinate, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)
		err = sinks[3].write(int(coordinate)-1, encoded)
		if err != nil {
			return err
		}
	}
	resources["rosterSerializationSeconds"] = time.Since(rosterStarted).Seconds()
	branches := map[PrimaryActivity]*usaCountryBranchDiagnostic{}
	minBytes, maxBytes, roundTrips := usaSourceUSDRecordBytes, 0, 0
	generated := time.Now()
	var generationInitial runtime.MemStats
	runtime.ReadMemStats(&generationInitial)
	var generationTime, serializationTime time.Duration
	for step := range options.People {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		coordinate := PersonIndex(step + 1)
		if options.Order == "descending" {
			coordinate = PersonIndex(options.People - step)
		}
		tick := time.Now()
		person, err := world.person(coordinate)
		if err != nil {
			return err
		}
		generationTime += time.Since(tick)
		canonical, err := internationalCanonical(person)
		if err != nil {
			return err
		}
		err = sinks[2].write(int(coordinate)-1, canonical)
		if err != nil {
			return err
		}
		tick = time.Now()
		records, oracle, err := source.bind(person)
		if err != nil {
			return err
		}
		if len(records) != 1 || len(oracle) != 1 {
			return fmt.Errorf("diagnostic full coverage requires exactly one record per coordinate")
		}
		encoded, err := serializeUSASourceUSDDiagnostic(ctx, source, records[0])
		if err != nil {
			return err
		}
		err = sinks[0].write(int(coordinate)-1, encoded)
		if err != nil {
			return err
		}
		truth, err := internationalCanonical(oracle[0])
		if err != nil {
			return err
		}
		err = sinks[1].write(int(coordinate)-1, truth)
		if err != nil {
			return err
		}
		serializationTime += time.Since(tick)
		minBytes, maxBytes = min(minBytes, len(encoded)), max(maxBytes, len(encoded))
		roundTrips++
		branch := branches[person.PrimaryActivity]
		if branch == nil {
			branch = &usaCountryBranchDiagnostic{}
			branches[person.PrimaryActivity] = branch
		}
		branch.Count++
		branch.incomes = append(branch.incomes, uint32(person.AnnualIncomeUSD))
		if person.AnnualIncomeUSD == 0 {
			branch.ZeroIncome++
		}

	}
	resources["prefixGenerationBindingRoundTripSeconds"] = time.Since(generated).Seconds()
	resources["personGenerationSeconds"] = generationTime.Seconds()
	resources["sourceBindingSerializationRoundTripSeconds"] = serializationTime.Seconds()
	var generationFinal runtime.MemStats
	runtime.ReadMemStats(&generationFinal)
	resources["prefixTotalAllocBytes"] = generationFinal.TotalAlloc - generationInitial.TotalAlloc
	resources["prefixMallocs"] = generationFinal.Mallocs - generationInitial.Mallocs
	for _, activity := range socioeconomicActivities {
		branch := branches[activity]
		if branch == nil || branch.Count == 0 {
			return fmt.Errorf("USA source diagnostic did not exercise every activity")
		}
		slices.Sort(branch.incomes)
		var sum uint64
		for _, value := range branch.incomes {
			sum += uint64(value)
		}
		branch.MeanUSD = float64(sum) / float64(branch.Count)
		branch.ZeroShare = float64(branch.ZeroIncome) / float64(branch.Count)
		branch.QuantilesUSD = map[string]uint32{}
		for _, percentile := range []int{10, 50, 90, 99} {
			branch.QuantilesUSD[fmt.Sprint(percentile)] = branch.incomes[(percentile*branch.Count+99)/100-1]
		}
		branch.incomes = nil
	}

	// Reuse M1's fixed sparse coordinates and valid boundaries in two orders.
	sparse := make([]PersonIndex, 0, 1007)
	for i := range 1000 {
		sparse = append(sparse, PersonIndex(1+(uint64(i)*60457+12345)%uint64(Layer1MaxPersonIndex)))
	}
	sparse = append(sparse, 1, 2, 35, 36, 20000, Layer1MaxPersonIndex-1, Layer1MaxPersonIndex)
	for step := range len(sparse) {
		slot := step
		if options.Order == "descending" {
			slot = (step * 97) % len(sparse)
		}
		person, err := world.person(sparse[slot])
		if err != nil {
			return err
		}
		records, oracle, err := source.bind(person)
		if err != nil {
			return err
		}
		encoded, err := serializeUSASourceUSDDiagnostic(ctx, source, records[0])
		if err != nil {
			return err
		}
		err = sinks[4].write(slot, encoded)
		if err != nil {
			return err
		}
		truth, err := internationalCanonical(oracle[0])
		if err != nil {
			return err
		}
		err = sinks[5].write(slot, truth)
		if err != nil {
			return err
		}
		roundTrips++
	}
	digests := map[string]string{}
	names := []string{"records", "oracle", "canonicalPeople", "companyRoster", "sparseRecords", "sparseOracle"}
	for i, name := range names {
		digests[name], err = sinks[i].digest(ctx)
		if err != nil {
			return err
		}
		err = sinks[i].file.Close()
		if err != nil {
			return err
		}
	}
	if digests["companyRoster"] != "b6715b2f3208208371d6d867c487f9133cf7a24095f5c8ef3e3e766b1753a28d" ||
		(options.People == 20000 && digests["canonicalPeople"] !=
			"db2e6e021e6270eee3c731e03e72a3746c434dc4e7dee1eabae899ed9ddd15ca") {
		return fmt.Errorf("USA source adapter changed historical canonical people or Company replay")
	}
	runtime.GC()
	var completed runtime.MemStats
	runtime.ReadMemStats(&completed)
	resources["heapAllocAtCompletionBytes"] = completed.HeapAlloc
	resources["totalAllocBytes"] = completed.TotalAlloc
	resources["mallocs"] = completed.Mallocs
	resources["gcCycles"] = completed.NumGC
	resources["totalSeconds"] = time.Since(started).Seconds()
	runtime.KeepAlive(source)
	err = writeUSACountryDiagnosticJSON(options.Directory, "resources.json", resources)
	if err != nil {
		return err
	}
	err = writeUSACountryDiagnosticJSON(options.Directory, "manifest.json", map[string]any{
		"schema": "usa-internal-source-usd-diagnostic-manifest-v1", "recordSchema": usaSourceUSDSchema,
		"status": "INTERNAL DIAGNOSTIC ONLY", "country": "US", "currency": "USD", "profile": usaSourceUSDProfile,
		"incomeUnit": world.data.socioeconomic.document.IncomeUnit,
		"source":     source.source.config, "sourceProtocol": SourceSimulationVersion,
		"sourceWorldIdentityComponent": "internal-world-id", "sourceWorldIdentityValue": world.id,
		"baseWorldID": world.baseID, "socioeconomicWorldID": world.id,
		"modelID":           fmt.Sprintf("%x", world.data.socioeconomic.model.digest),
		"baseLogicalDigest": fmt.Sprintf("%x", world.data.base.digest),
		"modelAssetSHA256":  usaSourceUSDModelSHA256, "modelAssetBytes": usaSourceUSDModelBytes,
		"sourceLock": usaCountrySourcePins(), "namespace": world.sampler.categorical.base.namespace, "seed": 2026,
		"anchorDate": PopulationAnchorDate, "referenceDate": PopulationAnchorDate,
		"personVersion": "person-us-v1", "generationVersion": usaSourceUSDVersion,
		"people": options.People, "records": options.People, "companyCount": 65536, "order": options.Order,
		"sparseCoordinates": sparse, "sparseRecordCount": len(sparse), "roundTrips": roundTrips,
		"roundTripMismatches": 0, "canonicalSHA256": digests,
		"canonicalOrder": "prefix numeric coordinate; roster numeric CompanyIndex; sparse listed coordinate order",
		"recordBytes":    map[string]any{"total": sinks[0].offset, "minimum": minBytes, "maximum": maxBytes},
		"branches":       branches, "photoRights": "PHOTO RIGHTS — NOT QUALIFIED", "officialSnapshotID": nil,
		"officialSourceCSV": false, "publicAdmission": false, "officialCheckpoint": false,
		"observationPolicy": "source-v1 names/contact; exact canonical geography and USD socioeconomic passthrough",
		"oracle":            "hidden/oracle.jsonl; excluded from ordinary records and ingestion",
	})
	if err != nil {
		return err
	}
	fmt.Printf("executed internal source-usd: order=%s records=%d companies=65536 sparse=%d roundTrips=%d "+
		"mismatches=0 recordsSHA256=%s peopleSHA256=%s\n",
		options.Order, options.People, len(sparse), roundTrips, digests["records"], digests["canonicalPeople"])

	return nil
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

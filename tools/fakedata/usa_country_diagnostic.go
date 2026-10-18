// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"github.com/krenalis/krenalis/tools/fakedata/internal/usadiagnostic"
)

type usaCountryBranchDiagnostic struct {
	Count        int               `json:"count"`
	ZeroIncome   int               `json:"zeroIncome"`
	ZeroShare    float64           `json:"zeroShare"`
	MeanUSD      float64           `json:"meanUSD"`
	QuantilesUSD map[string]uint32 `json:"quantilesUSD"`
	incomes      []uint32
}

// usaCountryDiagnosticOutput checks canonical roster coordinates and streams
// complete technical people without exposing a public/application format.
type usaCountryDiagnosticOutput struct {
	world     *usaCountryWorld
	companies *os.File
	next      CompanyIndex
}

func (o *usaCountryDiagnosticOutput) company(index CompanyIndex, company Company) error {

	if o == nil || o.world == nil || o.companies == nil || index != o.next || index == 0 || index > 65536 {
		return fmt.Errorf("canonical USA roster coordinate is required")
	}
	decoded, err := entityIndexFromID(o.world.sampler.categorical.base.namespace, "company", company.ID)
	if err != nil {
		return err
	}
	if decoded != uint64(index) {
		return fmt.Errorf("duplicate or noncanonical USA CompanyID")
	}
	expected, err := o.world.sampler.categorical.companies.company(index)
	if err != nil {
		return err
	}
	if expected != company {
		return fmt.Errorf("USA roster Company differs from admitted random access")
	}
	_, err = fmt.Fprintf(o.companies, "%d\t%s\t%s\t%d\t%s\n",
		index, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)
	if err != nil {
		return err
	}
	o.next++

	return nil
}

func assembleUSACountryDiagnostic(ctx context.Context, root, directory string) error {

	assets, err := loadUSACountryAssets(ctx, root)
	if err != nil {
		return err
	}
	data, err := assembleUSACountryModel(ctx, assets)
	if err != nil {
		return err
	}
	encoded, err := data.canonicalJSON()
	if err != nil {
		return err
	}
	projection, err := data.model.canonicalProjection()
	if err != nil {
		return err
	}
	logical, err := data.logicalDigest()
	if err != nil {
		return err
	}
	for _, name := range []string{"assets", "numeric"} {
		err = os.MkdirAll(filepath.Join(directory, name), 0o755)
		if err != nil {
			return err
		}
	}
	for name, value := range map[string][]byte{
		"assets/socioeconomic.json": encoded, "numeric/model-projection.json": projection,
	} {
		err = os.WriteFile(filepath.Join(directory, name), value, 0o644)
		if err != nil {
			return err
		}
	}
	lock := map[string]any{"schema": "usa-internal-approved-assets-lock-v1", "assets": usaCountrySourcePins()}
	lockBytes, err := internationalCanonical(lock)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(directory, "numeric/source-lock.json"), lockBytes, 0o644)
	if err != nil {
		return err
	}
	var nodes []map[string]any
	for node, path := range usaSocioeconomicPaths {
		asset, err := data.model.numeric.appendAsset(nil, node)
		if err != nil {
			return err
		}
		nodes = append(nodes, map[string]any{
			"node": node, "path": path, "assetSHA256": fmt.Sprintf("%x", sha256.Sum256(append(asset, '\n'))),
			"fingerprint": fmt.Sprintf("%x", data.model.fingerprints[node]),
		})
	}
	var ages []map[string]any
	for age, name := range socioeconomicAgeNames {
		canonical := append(appendSocioeconomicAge(nil, data.model.numeric.ages[age]), '\n')
		ages = append(ages, map[string]any{"name": name, "sha256": fmt.Sprintf("%x", sha256.Sum256(canonical))})
	}
	err = writeUSACountryDiagnosticJSON(directory, "numeric/model-identity.json", map[string]any{
		"schema": "usa-internal-country-model-identity-v1", "modelID": fmt.Sprintf("%x", data.model.digest),
		"assetSHA256": fmt.Sprintf("%x", sha256.Sum256(encoded)), "assetBytes": len(encoded),
		"logicalDocumentDigest": fmt.Sprintf("%x", logical), "nodes": nodes, "ageMaps": ages,
		"companyCount": 65536, "employeeCountLaws": 14, "personalModels": 6, "incomeLaws": 6,
		"incomeStructuralSlots": 34272, "incomeApplicableSlots": 29232, "incomeInapplicableSlots": 5040,
		"categoricalWeights": 6496, "incomeSupportsPerLaw": 385,
		"sourceLockSHA256": data.document.SourceModel.ByteSHA256,
	})
	if err != nil {
		return err
	}
	fmt.Printf("assembled complete US/USD model: modelID=%x assetSHA256=%x bytes=%d\n",
		data.model.digest, sha256.Sum256(encoded), len(encoded))

	return nil
}

func executeUSACountryDiagnostic(ctx context.Context, options usadiagnostic.Options) error {

	if options.Publish {
		return fmt.Errorf("USA public emission is not authorized; photo rights and public USD integration remain open")
	}
	if options.Country != "US" || options.Currency != "USD" || options.Directory == "" || options.Root == "" ||
		options.People < 20000 || options.People > 50000 {
		return fmt.Errorf("US/USD, repository, staging and 20000–50000 people are required")
	}
	switch options.Operation {
	case "assemble":
		return assembleUSACountryDiagnostic(ctx, options.Root, options.Directory)
	case "run":
		return replayUSACountryDiagnostic(ctx, options)
	case "source-usd":
		return executeUSASourceUSDDiagnostic(ctx, options)
	default:
		return fmt.Errorf("unknown USA diagnostic operation")
	}

}

func init() {
	usadiagnostic.Register(executeUSACountryDiagnostic)
}

// loadUSACountryDiagnostic returns only prepared data; acquisition buffers are
// unreachable before roster construction and are not retained by generation.
func loadUSACountryDiagnostic(ctx context.Context, root, directory string) (*usaCountryData, error) {

	assets, err := loadUSACountryAssets(ctx, root)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(directory, "assets/socioeconomic.json"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var document internationalSocioeconomic
	err = decodeInternationalJSON(ctx, file, &document)
	if err != nil {
		return nil, err
	}
	encoded, err := internationalCanonical(document)
	if err != nil {
		return nil, err
	}

	return prepareUSACountryData(ctx, assets, encoded)
}

// replayUSACountryDiagnostic releases the first roster before independently
// loading the second, so verification does not retain two Employer universes.
func replayUSACountryDiagnostic(ctx context.Context, options usadiagnostic.Options) error {

	started := time.Now()
	resources := map[string]any{"goVersion": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH}
	data, err := loadUSACountryDiagnostic(ctx, options.Root, options.Directory)
	if err != nil {
		return err
	}
	resources["coldLoadASeconds"] = time.Since(started).Seconds()
	runtime.GC()
	var loaded runtime.MemStats
	runtime.ReadMemStats(&loaded)
	resources["preparedHeapAllocBytes"] = loaded.HeapAlloc
	namespace := IdentityNamespace{Name: "usa-layer1", Generation: 1}
	constructed := time.Now()
	w, err := newUSACountryWorld(data, namespace, 2026)
	if err != nil {
		return err
	}
	resources["rosterASeconds"] = time.Since(constructed).Seconds()
	baseID, worldID, modelID := w.baseID, w.id, data.socioeconomic.model.digest
	data = nil
	runtime.GC()
	var prepared runtime.MemStats
	runtime.ReadMemStats(&prepared)
	resources["worldHeapAllocBytes"] = prepared.HeapAlloc
	resources["employerIndexPayloadBytes"] = 40 * 65536 * 16
	resources["employerIndexLogicalBytes"] = 64 + 32*40 + 16*40*65536

	runDirectory := filepath.Join(options.Directory, "run")
	err = os.MkdirAll(runDirectory, 0o755)
	if err != nil {
		return err
	}
	companyFile, err := os.Create(filepath.Join(runDirectory, "companies.tsv"))
	if err != nil {
		return err
	}
	defer companyFile.Close()
	output := usaCountryDiagnosticOutput{world: w, companies: companyFile, next: 1}
	companyHashes := make([][32]byte, 65536)
	rosterHash := sha256.New()
	var workforce uint64
	industryCompanies, bandCompanies := map[Industry]int{}, map[CompanySizeBand]int{}
	for coordinate := CompanyIndex(1); coordinate <= 65536; coordinate++ {
		company, err := w.sampler.categorical.companies.company(coordinate)
		if err != nil {
			return err
		}
		err = output.company(coordinate, company)
		if err != nil {
			return err
		}
		canonical := fmt.Appendf(nil, "%d\t%s\t%s\t%d\t%s\n",
			coordinate, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)
		rosterHash.Write(canonical)
		companyHashes[coordinate-1] = sha256.Sum256(canonical)
		workforce += uint64(company.EmployeeCount)
		industryCompanies[company.Industry]++
		bandCompanies[company.SizeBand]++
	}
	output.world = nil
	rosterDigest := fmt.Sprintf("%x", rosterHash.Sum(nil))
	if rosterDigest != "b6715b2f3208208371d6d867c487f9133cf7a24095f5c8ef3e3e766b1753a28d" {
		return fmt.Errorf("USA roster differs from historical local-stream result")
	}
	people, err := os.Create(filepath.Join(runDirectory, "people.jsonl"))
	if err != nil {
		return err
	}
	defer people.Close()
	prefixHash := sha256.New()
	hashes := make(map[PersonIndex][32]byte, options.People+1008)
	branches := map[PrimaryActivity]*usaCountryBranchDiagnostic{}
	education, family := map[EducationLevel]int{}, map[JobFamily]int{}
	level, industry := map[JobLevel]int{}, map[Industry]int{}
	employers := map[CompanyIndex]bool{}
	generated := time.Now()
	for coordinate := PersonIndex(1); coordinate <= PersonIndex(options.People); coordinate++ {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		person, err := w.person(coordinate)
		if err != nil {
			return err
		}
		encoded, err := internationalCanonical(person)
		if err != nil {
			return err
		}
		_, err = people.Write(encoded)
		if err != nil {
			return err
		}
		prefixHash.Write(encoded)
		hashes[coordinate] = sha256.Sum256(encoded)
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
		education[person.Education]++
		if person.Employee != nil {
			if person.SelfEmployed != nil || person.EmployerCompany == nil ||
				person.PrimaryActivity != PrimaryActivityEmployee {
				return fmt.Errorf("invalid employee presence contract")
			}
			selected, err := w.sampler.categorical.companies.company(person.Employee.Employer)
			if err != nil {
				return err
			}
			if selected != *person.EmployerCompany || selected.EmployeeCount == 0 {
				return fmt.Errorf("employee Industry or SizeBand differs from the roster relation")
			}
			family[person.Employee.JobFamily]++
			level[person.Employee.JobLevel]++
			industry[selected.Industry]++
			employers[person.Employee.Employer] = true
		}
		if person.SelfEmployed != nil {
			if person.Employee != nil || person.EmployerCompany != nil ||
				person.PrimaryActivity != PrimaryActivitySelfEmployed {
				return fmt.Errorf("invalid self-employed presence contract")
			}
			family[person.SelfEmployed.JobFamily]++
			industry[person.SelfEmployed.Industry]++
		}
		if person.Person.PhotoID != "" || person.Person.Address.Country != "US" {
			return fmt.Errorf("non-photo USA record contract violated")
		}

	}
	resources["ascendingPeopleSeconds"] = time.Since(generated).Seconds()
	for _, activity := range socioeconomicActivities {
		branch := branches[activity]
		if branch == nil || branch.Count == 0 {
			return fmt.Errorf("USA diagnostic did not exercise every activity")
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

	// Sparse coordinates cover the entire operational domain; the stride is
	// coprime to its size. No seed search or frequency adjustment is involved.
	sparse := make([]PersonIndex, 0, 1008)
	for i := range 1000 {
		sparse = append(sparse, PersonIndex(1+(uint64(i)*60457+12345)%uint64(Layer1MaxPersonIndex)))
	}
	sparse = append(sparse, 1, 2, 35, 36, 20000, Layer1MaxPersonIndex-1, Layer1MaxPersonIndex)
	sparseHash := sha256.New()
	for _, coordinate := range sparse {
		person, err := w.person(coordinate)
		if err != nil {
			return err
		}
		encoded, err := internationalCanonical(person)
		if err != nil {
			return err
		}
		hashes[coordinate] = sha256.Sum256(encoded)
		sparseHash.Write(encoded)
	}
	conditions, err := usaCountryEmployerDiagnostics(w)
	if err != nil {
		return err
	}
	w = nil
	runtime.GC()

	independent := time.Now()
	secondData, err := loadUSACountryDiagnostic(ctx, options.Root, options.Directory)
	if err != nil {
		return err
	}
	resources["coldLoadBSeconds"] = time.Since(independent).Seconds()
	constructed = time.Now()
	second, err := newUSACountryWorld(secondData, namespace, 2026)
	if err != nil {
		return err
	}
	resources["rosterBSeconds"] = time.Since(constructed).Seconds()
	if second.id != worldID || second.baseID != baseID || secondData.socioeconomic.model.digest != modelID {
		return fmt.Errorf("independent USA preparation identity mismatch")
	}
	replayed := 0
	for coordinate := PersonIndex(options.People); coordinate > 0; coordinate-- {
		person, err := second.person(coordinate)
		if err != nil {
			return err
		}
		encoded, err := internationalCanonical(person)
		if err != nil {
			return err
		}
		if sha256.Sum256(encoded) != hashes[coordinate] {
			return fmt.Errorf("USA reverse replay mismatch at %d", coordinate)
		}
		replayed++
	}
	for _, step := range []int{1, 97} {
		for i := range len(sparse) {
			coordinate := sparse[(i*step)%len(sparse)]
			person, err := second.person(coordinate)
			if err != nil {
				return err
			}
			encoded, err := internationalCanonical(person)
			if err != nil {
				return err
			}
			if sha256.Sum256(encoded) != hashes[coordinate] {
				return fmt.Errorf("USA sparse replay mismatch at %d", coordinate)
			}
			replayed++
		}
	}
	for coordinate := CompanyIndex(65536); coordinate > 0; coordinate-- {
		company, err := second.sampler.categorical.companies.company(coordinate)
		if err != nil {
			return err
		}
		encoded := fmt.Appendf(nil, "%d\t%s\t%s\t%d\t%s\n",
			coordinate, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)
		if sha256.Sum256(encoded) != companyHashes[coordinate-1] {
			return fmt.Errorf("USA Company replay mismatch at %d", coordinate)
		}
	}
	var completed runtime.MemStats
	runtime.ReadMemStats(&completed)
	resources["totalAllocBytes"] = completed.TotalAlloc
	resources["heapSysBytes"] = completed.HeapSys
	resources["heapAllocAtCompletionBytes"] = completed.HeapAlloc
	resources["gcCycles"] = completed.NumGC
	resources["totalSeconds"] = time.Since(started).Seconds()
	err = writeUSACountryDiagnosticJSON(options.Directory, "run/resources.json", resources)
	if err != nil {
		return err
	}
	err = writeUSACountryDiagnosticJSON(options.Directory, "run/summary.json", map[string]any{
		"schema": "usa-internal-country-diagnostic-summary-v1", "status": "M1 INTERNAL NON-PHOTO",
		"namespace": namespace, "seed": 2026, "baseWorldID": baseID, "socioeconomicWorldID": worldID,
		"modelID": fmt.Sprintf("%x", modelID), "people": options.People, "companyCount": 65536,
		"fullOutputSHA256":    fmt.Sprintf("%x", prefixHash.Sum(nil)),
		"sparseSHA256":        fmt.Sprintf("%x", sparseHash.Sum(nil)),
		"companyRosterSHA256": rosterDigest, "replayComparisons": replayed, "replayMismatches": 0,
		"companyReplayComparisons": 65536, "branches": branches, "educationCounts": education,
		"jobFamilyCounts": family, "jobLevelCounts": level, "industryCounts": industry,
		"distinctSelectedCompanies": len(employers), "companyIndustryCounts": industryCompanies,
		"companySizeBandCounts": bandCompanies, "companyTotalWorkforce": workforce,
		"employerConditions": conditions, "photoID": "", "sourceRecords": false,
		"validation": "REVIEW", "V07": "WARNING", "nationalCensusSample": false, "D0Replaced": false,
	})
	if err != nil {
		return err
	}
	fmt.Printf("executed US/USD: companies=65536 people=%d branches=6 replay=%d mismatches=0 sha256=%x\n",
		options.People, replayed, prefixHash.Sum(nil))

	return nil
}

// usaCountryEmployerDiagnostics reports raw finite-roster masses for every
// condition, separately from the approved analytical Employer distribution.
func usaCountryEmployerDiagnostics(w *usaCountryWorld) ([]map[string]any, error) {

	var result []map[string]any
	for condition, index := range w.sampler.categorical.companies.sampler.employers {
		if len(index.companies) != 65536 || len(index.cumulative) != 65536 {
			return nil, fmt.Errorf("incomplete USA Employer condition")
		}
		var masses [14][5]uint64
		var previous uint64
		for offset, coordinate := range index.companies {
			if coordinate != CompanyIndex(offset+1) || index.cumulative[offset] <= previous {
				return nil, fmt.Errorf("invalid USA Employer coordinate or cumulative mass")
			}
			sampler := &w.sampler.categorical.companies.sampler
			industry := int(sampler.companyIndustries[coordinate-1])
			band := socioeconomicSizeBand(sampler.companyCounts[coordinate-1])
			masses[industry][band] += index.cumulative[offset] - previous
			previous = index.cumulative[offset]
		}
		result = append(result, map[string]any{
			"jobFamily": socioeconomicJobFamilies[condition/4], "jobLevel": socioeconomicJobLevels[condition%4],
			"companies": len(index.companies), "total": previous, "industrySizeMasses": masses,
		})
	}

	return result, nil
}

func writeUSACountryDiagnosticJSON(directory, name string, value any) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, name), append(bytes.TrimSpace(encoded), '\n'), 0o644)
}

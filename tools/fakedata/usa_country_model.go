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
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const usaCountryBaseDirectory = "synthetic-world-usa-l1-us-2026a-BASE-v2-CANDIDATE/"
const usaCountryCompanyDirectory = "synthetic-world-usa-l2-company-employer-demo-v1-CANDIDATE/"

// usaCountryAssets contains verified aggregate assets only, never microdata.
type usaCountryAssets map[string][]byte

type usaCountryData struct {
	baseDocument  usaBaseDocument
	base          *usaBaseCandidate
	phones        *usaNPACandidate
	socioeconomic *usaSocioeconomicData
}

// usaCountrySourcePin binds acquisition/provenance, independently of M_all_US.
type usaCountrySourcePin struct {
	Path   string `json:"path"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// assembleUSACountryModel joins the admitted components through the actual
// international transport and full USA compiler, without monetary authoring.
func assembleUSACountryModel(ctx context.Context, assets usaCountryAssets) (*usaSocioeconomicData, error) {

	err := verifyUSACountryAssets(assets)
	if err != nil {
		return nil, err
	}
	var companyAssets usaCompanyEmployerAssets
	for i, pin := range usaCompanyEmployerPins {
		companyAssets[i] = assets[usaCountryCompanyDirectory+pin.name]
	}
	company, err := prepareUSACompanyEmployerCandidate(companyAssets)
	if err != nil {
		return nil, err
	}
	var incomeAssets usaIncomeAssets
	for i, pin := range usaIncomePins {
		incomeAssets[i] = assets[pin.path]
	}
	income, err := prepareUSAIncomeCandidate(ctx, incomeAssets)
	if err != nil {
		return nil, err
	}

	// Project the pinned personal components; the historical fixture reader is
	// deliberately not part of this full-current-register admission path.
	var personal [6]internationalCategorical
	err = json.Unmarshal(assets[usaPersonalCategoricalPins[4].path], &personal[0])
	if err != nil {
		return nil, err
	}
	var activity struct {
		Axes    []string   `json:"axes"`
		Weights [][]uint32 `json:"primaryActivity"`
	}
	err = json.Unmarshal(assets[usaPersonalCategoricalPins[5].path], &activity)
	if err != nil {
		return nil, err
	}
	personal[1].Axes, personal[1].Outcome = activity.Axes, "activity"
	for _, row := range activity.Weights {
		personal[1].Weights = append(personal[1].Weights, row...)
	}
	for i, outcome := range []string{"jobFamily", "jobLevel", "jobFamily", "industry"} {
		err = json.Unmarshal(assets[usaPersonalCategoricalPins[i+6].path], &personal[i+2])
		if err != nil {
			return nil, err
		}
		personal[i+2].Outcome = outcome
	}
	categorical, err := compileUSAPersonalCategorical(personal)
	if err != nil {
		return nil, err
	}

	// Recover the exact raw integers through the established subtree encoder.
	combined := &usaSocioeconomicModel{numeric: company.numeric,
		interpretation: company.interpretation, protocol: usaSocioeconomicProtocol}
	n := &combined.numeric
	n.education, n.primaryActivity = categorical.tables[0], categorical.tables[1]
	n.employeeJobFamily, n.employeeJobLevel = categorical.tables[2], categorical.tables[3]
	n.selfEmployedJobFamily, n.selfEmployedIndustry = categorical.tables[4], categorical.tables[5]
	copy(n.ages[:3], categorical.ages[:])
	copy(n.ages[3:], income.numeric.ages[3:])
	n.income = income.numeric.income
	projection, err := combined.canonicalProjection()
	if err != nil {
		return nil, err
	}
	var p usaSocioeconomicProjection
	err = json.Unmarshal(projection, &p)
	if err != nil {
		return nil, err
	}
	lock, err := internationalCanonical(map[string]any{
		"schema": "usa-internal-approved-assets-lock-v1", "assets": usaCountrySourcePins(),
	})
	if err != nil {
		return nil, err
	}
	d := internationalSocioeconomic{
		Schema: "sw-socioeconomic-data-v1", Release: "us-2026a-m1-internal-v1",
		Profile: p.Profile, Country: p.Country, AnchorDate: p.AnchorDate, AgeDomain: p.AgeDomain,
		SourceModel: &internationalSourceModel{
			Schema: "usa-internal-approved-assets-lock-v1", ByteSHA256: fmt.Sprintf("%x", sha256.Sum256(lock)),
		},
		Dimensions: p.Dimensions, AgePartitions: p.AgePartitions, CompanyCount: p.CompanyCount,
		IncomeUnit: p.IncomeUnit, Models: p.Models,
	}
	encoded, err := internationalCanonical(d)
	if err != nil {
		return nil, err
	}

	return decodeUSASocioeconomicData(ctx, bytes.NewReader(encoded))
}

// loadUSACountryAssets bounds each read before allocation and pins every byte.
// Cancellation is observed between and during reads, not just after decoding.
func loadUSACountryAssets(ctx context.Context, root string) (usaCountryAssets, error) {

	if ctx == nil || root == "" {
		return nil, fmt.Errorf("country context and repository root are required")
	}
	assets := usaCountryAssets{}
	for _, pin := range usaCountrySourcePins() {

		readBytes := pin.Bytes
		if pin.Path == usaIncomePins[0].path {
			readBytes = 295691
		}
		file, err := os.Open(filepath.Join(root, filepath.FromSlash(pin.Path)))
		if err != nil {
			return nil, err
		}
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() != int64(readBytes) {
			file.Close()
			return nil, fmt.Errorf("incompatible USA source size: %s", pin.Path)
		}
		encoded, err := io.ReadAll(io.LimitReader(&internationalReader{ctx: ctx, reader: file}, int64(readBytes)+1))
		closeErr := file.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if pin.Path == usaIncomePins[0].path {

			if len(encoded) != readBytes || fmt.Sprintf("%x", sha256.Sum256(encoded)) !=
				"63a4ad6f8cdf9e8626a20ff321141e5bc7acad5bf996601bd3f3c8d2dacbc9e0" {
				return nil, fmt.Errorf("altered approved USA register revision")
			}
			// The owner approval precedes the historical BASE-v2 appendix.
			// Retain the exact prior evidence for its unchanged source lock;
			// verifyUSACountryAssets still checks its complete historical SHA.
			encoded = append(encoded[:292333:292333], encoded[293677:]...)

		}
		assets[pin.Path] = encoded

	}
	err := verifyUSACountryAssets(assets)
	if err != nil {
		return nil, err
	}

	return assets, nil
}

// prepareUSACountryData verifies the assembled document against the original
// approved assets before the caller can construct a sampler or Company roster.
func prepareUSACountryData(ctx context.Context, assets usaCountryAssets, encoded []byte) (*usaCountryData, error) {

	expected, err := assembleUSACountryModel(ctx, assets)
	if err != nil {
		return nil, err
	}
	model, err := decodeUSASocioeconomicData(ctx, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	expectedBytes, err := expected.canonicalJSON()
	if err != nil {
		return nil, err
	}
	actualBytes, err := model.canonicalJSON()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(expectedBytes, actualBytes) {
		return nil, fmt.Errorf("USA model differs from the complete approved asset assembly")
	}

	// Reuse the actual Layer 1 preparers and their qualified source identities.
	ageBytes := assets["synthetic-world-usa-l1-us-2026a-CANDIDATE/assets/ages.json"]
	ages, err := decodeUSAAgeCandidate(ctx, bytes.NewReader(ageBytes))
	if err != nil {
		return nil, err
	}
	firstBytes := assets[usaCountryBaseDirectory+"assets/first-names.json"]
	first, err := decodeUSAFirstNamesV2Candidate(ctx, bytes.NewReader(firstBytes))
	if err != nil {
		return nil, err
	}
	lastBytes := assets[usaCountryBaseDirectory+"assets/surnames.json"]
	last, err := decodeUSASurname50KCandidate(ctx, bytes.NewReader(lastBytes))
	if err != nil {
		return nil, err
	}
	geographyBytes := assets["synthetic-world-usa-l1-us-2026a-GEOGRAPHY-v2-CANDIDATE/assets/geography.json"]
	geography, err := decodeUSAGeographyV2Candidate(ctx, bytes.NewReader(geographyBytes))
	if err != nil {
		return nil, err
	}
	var document usaBaseDocument
	err = decodeInternationalJSON(ctx, bytes.NewReader(assets[usaCountryBaseDirectory+"assets/base.json"]), &document)
	if err != nil {
		return nil, err
	}
	base, err := prepareUSABaseCandidate(&document, usaBaseEvidence{ages, first, last, geography})
	if err != nil {
		return nil, err
	}
	phones, err := decodeUSANPACandidate(ctx, bytes.NewReader(assets[usaCountryBaseDirectory+"assets/npas.json"]))
	if err != nil {
		return nil, err
	}

	return &usaCountryData{baseDocument: document, base: base, phones: phones, socioeconomic: model}, nil
}

// usaCountrySourcePins joins complete-current-register locks with component
// locks in canonical path order, without the fixture-only historical prefix.
func usaCountrySourcePins() []usaCountrySourcePin {

	pins := map[string]usaCountrySourcePin{}
	for _, pin := range usaCountryAdditionalPins {
		pins[pin.Path] = pin
	}
	for _, pin := range usaPersonalCategoricalPins[1:] {
		pins[pin.path] = usaCountrySourcePin{pin.path, pin.bytes, pin.sha256}
	}
	for _, pin := range usaCompanyEmployerPins {
		path := usaCountryCompanyDirectory + pin.name
		pins[path] = usaCountrySourcePin{path, pin.bytes, pin.sha256}
	}
	for _, pin := range usaIncomePins {
		pins[pin.path] = usaCountrySourcePin{pin.path, pin.bytes, pin.sha256}
	}
	result := make([]usaCountrySourcePin, 0, len(pins))
	for _, pin := range pins {
		result = append(result, pin)
	}
	slices.SortFunc(result, func(a, b usaCountrySourcePin) int { return strings.Compare(a.Path, b.Path) })

	return result
}

func verifyUSACountryAssets(assets usaCountryAssets) error {
	for _, pin := range usaCountrySourcePins() {
		encoded := assets[pin.Path]
		if len(encoded) != pin.Bytes || fmt.Sprintf("%x", sha256.Sum256(encoded)) != pin.SHA256 {
			return fmt.Errorf("missing or altered USA country source: %s", pin.Path)
		}
	}
	return nil
}

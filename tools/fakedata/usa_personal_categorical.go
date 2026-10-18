// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const (
	usaEducationDirectory    = "synthetic-world-usa-l2-acs-2024-education-crosswalks-v1-CANDIDATE/"
	usaActivityDirectory     = "synthetic-world-usa-l2-primary-activity-v1-CANDIDATE/"
	usaProfessionalDirectory = "synthetic-world-usa-l2-professional-real-fitting-v1-CANDIDATE/"
)

// Admission pins include the later owner decisions; historical CANDIDATE flags
// in the components are neither rewritten nor accepted as approval evidence.
var usaPersonalCategoricalPins = [...]struct {
	path   string
	bytes  int
	sha256 string
}{
	{"SYNTHETIC_WORLD_USA_ECONOMIC_EVIDENCE_REGISTER_v1_CANDIDATE.md", 120302,
		"7f68e8c0aab766e599871a70b1bddca4ae177674718ff1255ff8f0cd07a860dd"},
	{usaEducationDirectory + "manifest.json", 5717,
		"fecbd8e488519fc611209de93272fc92b70de718f4a5a2ad2545cb3d5fe20c05"},
	{usaActivityDirectory + "manifest.json", 3212,
		"e36d0f4cac3882177cccddce29a8fb5a27a29b4a79f99ac4542eece90e428e1b"},
	{usaProfessionalDirectory + "manifest.json", 6840,
		"3f5c3a41b293da40db6bdfae30f5d89dce33740cdef6808a093af56ce8ec8ec4"},
	{usaEducationDirectory + "assets/education-compiler-component.json", 3391,
		"6a03d8e2661050e3576f8a80ab153ce4ec50f1bfbb6880736a6725310d9816ae"},
	{usaActivityDirectory + "assets/primary-activity-compiler-component.json", 31152,
		"b25375facbba4c9d4316aa53d556d57bd1584159e1541fbbd980a9e28934f97b"},
	{usaProfessionalDirectory + "components/employeeJobFamily.json", 6597,
		"64f1abc466d69ea8277de76965498fd32c3c33094f5afceb8a147d98d9ef4ba7"},
	{usaProfessionalDirectory + "components/employeeJobLevel.json", 17386,
		"f51d0edf56e1e8ebeb9b00597c916ac8b526b3bbfef1ebc7694bb3d8317efce8"},
	{usaProfessionalDirectory + "components/selfEmployedJobFamily.json", 6505,
		"4fdf7512af93c3c1019081bd072a2ccb4c25a9fcd24e1d1824395b9f7e0606b9"},
	{usaProfessionalDirectory + "components/selfEmployedIndustry.json", 1656,
		"bab93541f6c40a15de7371c3aecfccf328df4d111d53946c986fb8e96c6055fa"},
}

var usaPersonalCategoricalNodes = [...]int{
	socioeconomicEducation, socioeconomicPrimaryActivity, socioeconomicEmployeeJobFamily,
	socioeconomicEmployeeJobLevel, socioeconomicSelfEmployedJobFamily, socioeconomicSelfEmployedIndustry,
}

type usaPersonalCategoricalAssets [10][]byte

// usaPersonalCategoricalCandidate holds six tables, three independent age maps
// and an admission guard, without Income or a public World representation.
type usaPersonalCategoricalCandidate struct {
	ages         [3][55]uint8
	tables       [6]socioeconomicWeights
	fingerprints [6][sha256.Size]byte
	identity     [sha256.Size]byte
}

// compileUSAPersonalCategorical compiles bounded national categorical input;
// only the pinned admission entry below can assign an admission identity.
func compileUSAPersonalCategorical(input [6]internationalCategorical) (*usaPersonalCategoricalCandidate, error) {

	axes := [6][]string{
		{"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := [...]string{"education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	rows := [...]int{54, 504, 63, 420, 63, 10}
	widths := [...]int{7, 6, 10, 4, 10, 14}
	model := &usaPersonalCategoricalCandidate{}
	for i, values := range input {

		if !slices.Equal(values.Axes, axes[i]) || values.Outcome != outcomes[i] ||
			len(values.Weights) != rows[i]*widths[i] {
			return nil, fmt.Errorf("invalid USA personal categorical axes, outcome or dimensions at node %d", i)
		}
		out := &model.tables[i]
		out.width = widths[i]
		out.cumulative = make([]uint64, len(values.Weights))
		for row := range rows[i] {

			start := row * out.width
			weights := values.Weights[start : start+out.width]
			for outcome, weight := range weights {
				if (weight == 0) != (i == 0 && outcome == 4) {
					return nil, fmt.Errorf("invalid USA personal categorical support at node %d row %d", i, row)
				}
			}
			// Each row has at most 14 U32 values: its U64 sum cannot overflow.
			err := compileSocioeconomicWeights(weights, out.cumulative[start:start+out.width])
			if err != nil {
				return nil, err
			}

		}

	}

	// These are separately owned maps, even where their contents are equal.
	bounds := usaInternationalPartitionBounds()
	for i := range model.ages {
		bucket := 0
		for offset := range 55 {
			if bucket+1 < len(bounds[i]) && uint32(offset+21) >= bounds[i][bucket+1] {
				bucket++
			}
			model.ages[i][offset] = uint8(bucket)
		}
	}

	// Reuse the frozen subtree encoders, without constructing a complete model.
	projection := socioeconomicNumericModel{
		education: model.tables[0], primaryActivity: model.tables[1], employeeJobFamily: model.tables[2],
		employeeJobLevel: model.tables[3], selfEmployedJobFamily: model.tables[4], selfEmployedIndustry: model.tables[5],
	}
	for i, node := range usaPersonalCategoricalNodes {

		asset, err := projection.appendAsset(nil, node)
		if err != nil {
			return nil, err
		}
		component, err := BytesComponent("asset", append(asset, '\n'))
		if err != nil {
			return nil, err
		}
		components := []Component{component}
		if age := socioeconomicNodeAges[node]; age >= 0 {
			component, err = BytesComponent("age-map", append(appendSocioeconomicAge(nil, model.ages[age]), '\n'))
			if err != nil {
				return nil, err
			}
			components = append(components, component)
		}
		model.fingerprints[i], err = canonicalDigest("fakedata/"+usaSocioeconomicPaths[node]+"/model", components)
		if err != nil {
			return nil, err
		}

	}

	return model, nil
}

// prepareUSAPersonalCategoricalCandidate pins all bytes before decoding or
// allocating tables. Its identity is an admission receipt, never a stream salt.
func prepareUSAPersonalCategoricalCandidate(assets usaPersonalCategoricalAssets) (*usaPersonalCategoricalCandidate, error) {

	components := make([]Component, len(assets))
	for i, pin := range usaPersonalCategoricalPins {
		if len(assets[i]) != pin.bytes {
			return nil, fmt.Errorf("USA personal categorical asset %d has incompatible byte count", i)
		}
		digest := sha256.Sum256(assets[i])
		if fmt.Sprintf("%x", digest) != pin.sha256 {
			return nil, fmt.Errorf("USA personal categorical asset %d has incompatible digest", i)
		}
		component, err := BytesComponent(fmt.Sprintf("asset-%d", i), digest[:])
		if err != nil {
			return nil, err
		}
		components[i] = component
	}
	for _, title := range []string{
		"USA ACS 2024 EDUCATION NODE", "USA CPS 2026 PRIMARY ACTIVITY 504×6",
		"USA EMPLOYEE JOBFAMILY", "USA EMPLOYEE JOBLEVEL",
		"USA SELFEMPLOYED JOBFAMILY", "USA SELFEMPLOYED INDUSTRY",
	} {
		if strings.Count(string(assets[0]), "## OWNER-APPROVED "+title+" — DEMO PLAUSIBLE V1\n") != 1 {
			return nil, fmt.Errorf("missing or duplicate USA personal categorical owner decision")
		}
	}

	// Check manifest membership as well as the separately pinned manifest bytes.
	for i, manifest := range [...]int{1, 2, 3, 3, 3, 3} {
		var document struct {
			Files map[string]struct {
				Size   int    `json:"size"`
				SHA256 string `json:"sha256"`
			} `json:"files"`
		}
		err := json.Unmarshal(assets[manifest], &document)
		if err != nil {
			return nil, err
		}
		pin := usaPersonalCategoricalPins[i+4]
		root := strings.TrimSuffix(usaPersonalCategoricalPins[manifest].path, "manifest.json")
		entry, ok := document.Files[strings.TrimPrefix(pin.path, root)]
		if !ok || entry.Size != pin.bytes || entry.SHA256 != pin.sha256 {
			return nil, fmt.Errorf("USA personal categorical manifest does not bind component %d", i)
		}
	}

	var input [6]internationalCategorical
	err := json.Unmarshal(assets[4], &input[0])
	if err != nil {
		return nil, err
	}
	var activity struct {
		AgeLowerBounds  []uint32          `json:"ageLowerBounds"`
		Axes            []string          `json:"axes"`
		AxisSizes       []uint32          `json:"axisSizes"`
		Country         string            `json:"country"`
		Outcomes        []PrimaryActivity `json:"outcomes"`
		PrimaryActivity [][]uint32        `json:"primaryActivity"`
		TerritorySystem string            `json:"territorySystem"`
	}
	err = json.Unmarshal(assets[5], &activity)
	if err != nil {
		return nil, err
	}
	if activity.Country != "US" || activity.TerritorySystem != "sw-us-2026a-census-division-v1" ||
		!slices.Equal(activity.AxisSizes, []uint32{8, 7, 9}) || len(activity.PrimaryActivity) != 504 ||
		!slices.Equal(activity.Outcomes, socioeconomicActivities[:]) ||
		!slices.Equal(activity.AgeLowerBounds, usaInternationalPartitionBounds()[1]) {
		return nil, fmt.Errorf("invalid USA PrimaryActivity national binding or dimensions")
	}
	input[1] = internationalCategorical{Axes: activity.Axes, Outcome: "activity"}
	for _, row := range activity.PrimaryActivity {
		if len(row) != 6 {
			return nil, fmt.Errorf("invalid USA PrimaryActivity outcome count")
		}
		input[1].Weights = append(input[1].Weights, row...)
	}

	names := [...]string{"employeeJobFamily", "employeeJobLevel", "selfEmployedJobFamily", "selfEmployedIndustry"}
	sizes := [4][]uint32{{7, 9}, {6, 7, 10}, {7, 9}, {10}}
	for i, name := range names {
		var professional struct {
			AgeLowerBounds []uint32 `json:"ageLowerBounds"`
			Axes           []string `json:"axes"`
			AxisSizes      []uint32 `json:"axisSizes"`
			Outcome        string   `json:"outcome"`
			Weights        []uint32 `json:"weights"`
		}
		err = json.Unmarshal(assets[i+6], &professional)
		if err != nil {
			return nil, err
		}
		if professional.Outcome != name || !slices.Equal(professional.AxisSizes, sizes[i]) ||
			i == 1 && !slices.Equal(professional.AgeLowerBounds, usaInternationalPartitionBounds()[2]) ||
			i != 1 && len(professional.AgeLowerBounds) != 0 {
			return nil, fmt.Errorf("invalid USA professional component binding at node %d", i)
		}
		outcome := "jobFamily"
		if i == 1 {
			outcome = "jobLevel"
		}
		if i == 3 {
			outcome = "industry"
		}
		input[i+2] = internationalCategorical{
			Axes: professional.Axes, Outcome: outcome, Weights: professional.Weights,
		}
	}
	model, err := compileUSAPersonalCategorical(input)
	if err != nil {
		return nil, err
	}
	model.identity, err = canonicalDigest("fakedata/socioeconomic/us-v1/personal-categorical/admission", components)
	if err != nil {
		return nil, err
	}

	return model, nil
}

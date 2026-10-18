// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"math/bits"
	"slices"
	"strings"
)

const usaIncomeReviewDirectory = "synthetic-world-usa-l2-income-independent-review-and-go-smoke-v1-CANDIDATE/"

// These pins bind the exact reviewed numerical asset to the owner decisions,
// preregistration and limited acceptance. They never enter generation streams.
var usaIncomePins = [...]struct {
	path   string
	bytes  int
	sha256 string
}{
	{"SYNTHETIC_WORLD_USA_ECONOMIC_EVIDENCE_REGISTER_v1_CANDIDATE.md", 294347,
		"4c8c78ff7c3009af815d49d3bae019ae8cf9d21c5e451203ffa9ced6e57a98d6"},
	{"synthetic-world-usa-l2-income-stage0-fast-gates-v1-CANDIDATE/stage2/laws-and-maps.json", 120111,
		"2364916e9b3afe401434f7ad53f6b58dd7d26020f14c4a2ad374b9cb0ee7c3c3"},
	{"synthetic-world-usa-l2-income-stage0-fast-gates-v1-CANDIDATE/manifest.json", 6921,
		"18ee902eddaed9e1300991aed9aa4c91cdcb193f2d431fb14457dee196e91cbf"},
	{"synthetic-world-usa-l2-income-stage0-fast-gates-v1-CANDIDATE/owner-bound-config.json", 2162,
		"8c61c2939554154525c5245f925484490cac95fcc473bf9048af347718be4e39"},
	{"synthetic-world-usa-l2-income-seven-decisions-and-prereg-v1-CANDIDATE/proposal/central-values.json", 80558,
		"b1bc8275bdeb5a35116cad715da0f61c58b7bfa2da84c466a379b9407839cc42"},
	{"synthetic-world-usa-l2-income-seven-decisions-and-prereg-v1-CANDIDATE/preregistration-record.json", 122806,
		"c126dda08654f78db3df46efe5c2f1a5735474d104c2480c6adea967fbb93691"},
	{"synthetic-world-usa-l2-income-stage0-fast-gates-v1-CANDIDATE/receipts/owner-adoption.json", 624,
		"1dc709cddae2ebbd76358359e7db7725fe84c62da2045d4360e88a5f688d4a42"},
	{usaIncomeReviewDirectory + "receipts/independent-review.json", 2366,
		"c7dfa29aec5b0ba406503f8101bcce23bbed13bf17e920feaabded8b4b0a0c8c"},
	{usaIncomeReviewDirectory + "receipts/internal-acceptance.json", 887,
		"d519752583bffc0e6937fe493ae3f0b71cd57d44c6369152d03b2c000f710e1f"},
	{usaIncomeReviewDirectory + "receipts/internal-acceptance-append.md", 2556,
		"96c753afae730868fd98f29b77dbc47542a4445acda672590487d015bc2ab9d0"},
	{"SYNTHETIC_WORLD_V1_LAYER2_CONTRACT5_v1_FROZEN.md", 107206,
		"8a75054f793ef47e7071e3eb9fe04c0b32abfdac9223da859f95a333a1526e67"},
	{"SYNTHETIC_WORLD_V1_LAYER2_D2A_v1_FROZEN.md", 150174,
		"33ca4c80599dddb0564ea4d27262e430b1931e0368c8d1bee6ca6c10d8078be4"},
	{"SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_v1.md", 45824,
		"8abf0347612f75c6f033b6114e9dbe28332994ee0f6f1cce6285790ea03a796f"},
	{"SYNTHETIC_WORLD_USA_NUMERIC_PRODUCTION_SPEC_v1_CANDIDATE.md", 85375,
		"0520a6a0cc44a03a546386b8b266ddfe4ba2cdc15a9f3bf543da6d666ca6b29b"},
}

var usaIncomeBranchNames = [...]string{"Employee", "SelfEmployed", "Unemployed", "Student", "Retired", "Inactive"}

type usaIncomeAssets [14][]byte

// usaIncomeCandidate is an internal six-branch sampler view whose identity
// guards admission; it never contributes aggregate identity to a stream.
type usaIncomeCandidate struct {
	numeric      socioeconomicNumericModel
	fingerprints [6][sha256.Size]byte
	identity     [sha256.Size]byte
}

// prepareFingerprints uses the existing node-owned encoding and exact USA path
// registry. No aggregate/receipt/authority digest becomes stochastic material.
func (m *usaIncomeCandidate) prepareFingerprints() error {
	for branch := range 6 {
		node := socioeconomicIncomeEmployee + branch
		asset, err := m.numeric.appendAsset(nil, node)
		if err != nil {
			return err
		}
		component, err := BytesComponent("asset", append(asset, '\n'))
		if err != nil {
			return err
		}
		components := []Component{component}
		if age := socioeconomicNodeAges[node]; age >= 0 {
			component, err = BytesComponent("age-map", append(appendSocioeconomicAge(nil, m.numeric.ages[age]), '\n'))
			if err != nil {
				return err
			}
			components = append(components, component)
		}
		m.fingerprints[branch], err = canonicalDigest("fakedata/"+usaSocioeconomicPaths[node]+"/model", components)
		if err != nil {
			return err
		}
	}
	return nil
}

// usaIncomeDocument mirrors the exact private stage2 transport, with bounded
// closed decoding before compilation. Null denotes only an inapplicable slot.
type usaIncomeDocument struct {
	Branches struct {
		Employee     usaIncomeDocumentBranch `json:"Employee"`
		Inactive     usaIncomeDocumentBranch `json:"Inactive"`
		Retired      usaIncomeDocumentBranch `json:"Retired"`
		SelfEmployed usaIncomeDocumentBranch `json:"SelfEmployed"`
		Student      usaIncomeDocumentBranch `json:"Student"`
		Unemployed   usaIncomeDocumentBranch `json:"Unemployed"`
	} `json:"branches"`
	OwnerApprovedLaws                   bool   `json:"ownerApprovedLaws"`
	PubliclyAdmissibleSocioeconomicJSON bool   `json:"publiclyAdmissibleSocioeconomicJSON"`
	Schema                              string `json:"schema"`
	Stage                               uint32 `json:"stage"`
}

func (d *usaIncomeDocument) ordered() [6]*usaIncomeDocumentBranch {
	return [6]*usaIncomeDocumentBranch{
		&d.Branches.Employee, &d.Branches.SelfEmployed, &d.Branches.Unemployed,
		&d.Branches.Student, &d.Branches.Retired, &d.Branches.Inactive,
	}
}

type usaIncomeDocumentBranch struct {
	CanonicalIdentitySHA256 string `json:"canonicalIdentitySHA256"`
	Laws                    []struct {
		ID       uint32   `json:"id"`
		Supports []uint32 `json:"supports"`
		Total    uint64   `json:"total"`
		Weights  []uint32 `json:"weights"`
	} `json:"laws"`
	Map struct {
		ApplicableSlots   uint32    `json:"applicableSlots"`
		Axes              []string  `json:"axes"`
		AxisSizes         []uint32  `json:"axisSizes"`
		InapplicableSlots uint32    `json:"inapplicableSlots"`
		LawIDs            []*uint32 `json:"lawIDs"`
		NullProjection    string    `json:"nullProjection"`
		Order             string    `json:"order"`
		StructuralSlots   uint32    `json:"structuralSlots"`
	} `json:"map"`
	Statistics struct {
		Q10  string `json:"10"`
		Q50  string `json:"50"`
		Q90  string `json:"90"`
		Q99  string `json:"99"`
		Mean string `json:"M"`
		Zero string `json:"Z"`
	} `json:"statistics"`
}

// compileUSAIncome compiles the complete six-branch family using the existing
// finite-law compiler and subtree encoders, without inventing a complete model.
func compileUSAIncome(document *usaIncomeDocument) (*usaIncomeCandidate, error) {

	if document == nil || document.Schema != "usa-income-stage2-numeric-components-candidate-v1" ||
		document.Stage != 2 || document.OwnerApprovedLaws || document.PubliclyAdmissibleSocioeconomicJSON {
		return nil, fmt.Errorf("reviewed internal USA stage2 Income candidate is required")
	}
	axes := [6][]string{
		{"Division", "JobFamily", "JobLevel", "Industry", "SizeBand"},
		{"Age-of-own-consumer", "Division", "JobFamily", "Industry"},
		{"Age-of-own-consumer", "Education", "Division"}, {"Age-of-own-consumer", "Education", "Division"},
		{"Age-of-own-consumer", "Education", "Division"}, {"Age-of-own-consumer", "Education", "Division"},
	}
	sizes := [6][]uint32{{9, 10, 4, 14, 5}, {6, 9, 10, 14}, {6, 7, 9}, {6, 7, 9}, {6, 7, 9}, {6, 7, 9}}
	model := &usaIncomeCandidate{numeric: socioeconomicNumericModel{territories: socioeconomicUSADivisions}}
	var retained, distinct uint64
	identities := map[string]bool{}
	for branch, input := range document.ordered() {

		slots := uint32(1)
		for _, size := range sizes[branch] {
			slots *= size
		}
		inapplicable := uint32(0)
		if branch == 0 {
			inapplicable = slots / 5
		}
		if !slices.Equal(input.Map.Axes, axes[branch]) || !slices.Equal(input.Map.AxisSizes, sizes[branch]) ||
			len(input.Map.LawIDs) != int(slots) || input.Map.StructuralSlots != slots ||
			input.Map.InapplicableSlots != inapplicable || input.Map.ApplicableSlots != slots-inapplicable ||
			input.Map.Order != "first dimension outermost; last dimension innermost" ||
			input.Map.NullProjection != "sentinel65535; inapplicable, never a productive zero law" ||
			len(input.Laws) != 1 || input.Laws[0].ID != 0 {
			return nil, fmt.Errorf("incompatible USA Income branch %d axes, coverage or law count", branch)
		}
		law := input.Laws[0]
		if len(law.Supports) != 385 || len(law.Weights) != 385 || law.Supports[0] != 0 {
			return nil, fmt.Errorf("USA Income branch %d requires zero and 384 positive supports", branch)
		}
		var total uint64
		for _, weight := range law.Weights {
			var carry uint64
			total, carry = bits.Add64(total, uint64(weight), 0)
			if carry != 0 {
				return nil, fmt.Errorf("USA Income total overflows uint64")
			}
		}
		if total != law.Total || total != 4294967295 {
			return nil, fmt.Errorf("USA Income branch %d has incompatible denominator", branch)
		}
		// The exact cardinalities precede all compiled law/map allocations.
		retained += 128 + 2*uint64(slots) + 32 + 16*385
		if retained > 8<<20 {
			return nil, fmt.Errorf("USA Income retained storage exceeds bounds")
		}
		compiled, err := compileSocioeconomicLaw(socioeconomicLawJSON{values: law.Supports, weights: law.Weights})
		if err != nil {
			return nil, err
		}
		canonical, err := internationalCanonical(struct {
			Supports []uint32 `json:"supports"`
			Weights  []uint32 `json:"weights"`
		}{law.Supports, law.Weights})
		if err != nil {
			return nil, err
		}
		identity := fmt.Sprintf("%x", sha256.Sum256(canonical))
		if identity != input.CanonicalIdentitySHA256 || identities[identity] {
			return nil, fmt.Errorf("incompatible or confused USA Income branch identity")
		}
		identities[identity] = true
		distinct++
		output := &model.numeric.income[branch]
		output.laws = []socioeconomicLaw{compiled}
		output.tupleMap = make([]uint16, slots)
		for slot, id := range input.Map.LawIDs {
			if branch == 0 && slot%5 == 0 {
				if id != nil {
					return nil, fmt.Errorf("inapplicable USA Income slot requires null, never a zero law")
				}
				output.tupleMap[slot] = 65535
				continue
			}
			if id == nil || *id != 0 {
				return nil, fmt.Errorf("missing or invalid USA Income law reference")
			}
		}

	}
	if distinct != 6 {
		return nil, fmt.Errorf("six distinct USA Income laws are required")
	}
	for age := 3; age < 8; age++ {
		bounds := usaInternationalPartitionBounds()[age]
		bucket := 0
		for offset := range 55 {
			if bucket+1 < len(bounds) && uint32(offset+21) >= bounds[bucket+1] {
				bucket++
			}
			model.numeric.ages[age][offset] = uint8(bucket)
		}
	}
	err := model.prepareFingerprints()
	if err != nil {
		return nil, err
	}

	return model, nil
}

// prepareUSAIncomeCandidate admits exactly the reviewed bytes and conditional
// owner action. A caller cannot obtain admission by setting an approval flag.
func prepareUSAIncomeCandidate(ctx context.Context, assets usaIncomeAssets) (*usaIncomeCandidate, error) {

	if ctx == nil {
		return nil, fmt.Errorf("Income admission context is required")
	}

	var encoded int
	for i, pin := range usaIncomePins {
		if len(assets[i]) != pin.bytes || fmt.Sprintf("%x", sha256.Sum256(assets[i])) != pin.sha256 {
			return nil, fmt.Errorf("incompatible USA Income admission asset %d", i)
		}
		encoded += len(assets[i])
	}
	if encoded > 8<<20 || !bytes.Equal(assets[0][289776:289776+len(assets[9])], assets[9]) ||
		strings.Count(string(assets[0]), "## OWNER-ACCEPTED USA INCOME VALIDATION EVIDENCE LIMITATION "+
			"— INTERNAL DEMO TESTING ONLY V1\n") != 1 ||
		strings.Count(string(assets[0]), "## OWNER-APPROVED USA INCOME METHOD C — 45 CENTRAL VALUES, "+
			"FULL NEUTRALITY — DEMO PLAUSIBLE V1\n") != 1 {
		return nil, fmt.Errorf("missing or incompatible USA Income owner authority")
	}
	var receipt struct {
		Verdict          string `json:"verdict"`
		HardPass         int    `json:"hardPass"`
		HardFail         int    `json:"hardFail"`
		Validation       string `json:"validation"`
		PublicApproval   bool   `json:"publicApproval"`
		MaterialFindings []any  `json:"materialFindings"`
		LawAsset         struct {
			Bytes  int    `json:"bytes"`
			SHA256 string `json:"sha256"`
		} `json:"lawAsset"`
	}
	err := json.Unmarshal(assets[7], &receipt)
	if err != nil {
		return nil, err
	}
	if receipt.Verdict != "INDEPENDENTLY VERIFIED" || receipt.HardPass != 24 || receipt.HardFail != 0 ||
		receipt.Validation != "REVIEW" || receipt.PublicApproval || len(receipt.MaterialFindings) != 0 ||
		receipt.LawAsset.Bytes != usaIncomePins[1].bytes || receipt.LawAsset.SHA256 != usaIncomePins[1].sha256 {
		return nil, fmt.Errorf("independent review does not admit internal Income testing")
	}
	var document usaIncomeDocument
	err = json.UnmarshalRead(&internationalReader{ctx: ctx, reader: bytes.NewReader(assets[1])},
		&document, json.RejectUnknownMembers(true))
	if err != nil {
		return nil, err
	}
	model, err := compileUSAIncome(&document)
	if err != nil {
		return nil, err
	}
	// This binds the entire read-only admission bundle, without creating M_all_US.
	components := make([]Component, len(assets))
	for i, data := range assets {
		digest := sha256.Sum256(data)
		components[i], err = BytesComponent(fmt.Sprintf("asset-%d", i), digest[:])
		if err != nil {
			return nil, err
		}
	}
	model.identity, err = canonicalDigest("fakedata/socioeconomic/us-v1/income/internal-admission", components)
	if err != nil {
		return nil, err
	}

	return model, nil
}

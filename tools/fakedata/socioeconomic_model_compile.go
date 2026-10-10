// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"slices"
)

func compileSocioeconomicLaw(input socioeconomicLawJSON) (socioeconomicLaw, error) {

	if len(input.values) == 0 || len(input.values) != len(input.weights) {
		return socioeconomicLaw{}, fmt.Errorf("support and weights must have equal positive length")
	}
	for i, value := range input.values {
		if i > 0 && value <= input.values[i-1] {
			return socioeconomicLaw{}, fmt.Errorf("support %d is not strictly increasing", i)
		}
		if input.weights[i] == 0 {
			return socioeconomicLaw{}, fmt.Errorf("canonical support %d has zero weight", i)
		}
	}
	cumulative := make([]uint64, len(input.weights))
	err := compileSocioeconomicWeights(input.weights, cumulative)
	if err != nil {
		return socioeconomicLaw{}, err
	}

	values := make([]uint32, len(input.values))
	copy(values, input.values)

	return socioeconomicLaw{values: values, cumulative: cumulative}, nil
}

// compileSocioeconomicModel validates the complete canonical numerical model.
// Work is bounded by the decoder's D2-A cardinalities; no network, random
// streams, Company universe, or Person generation participates in compilation.
func compileSocioeconomicModel(document *socioeconomicModelJSON) (*SocioeconomicModel, error) {

	// Reserve retained storage before compilation. D2-A's numerical charges
	// include exact weights recoverable from cumulative differences. The extra
	// 64 KiB covers model headers, excess law descriptors and fixed dimensions.
	// Decoder bounds make these int sums safe. Decoded work and retained output
	// each fit 8 MiB; with no retained source buffer their overlap fits 24 MiB.
	retained := 64 << 10
	weights := [...][]uint32{
		document.companyIndustry, document.education, document.primaryActivity, document.employeeJobFamily,
		document.employeeJobLevel, document.selfEmployedJobFamily, document.selfEmployedIndustry,
	}
	widths := [...]int{14, 7, 6, 10, 4, 10, 14}
	for i, values := range weights {
		retained += 64 + 32*(len(values)/widths[i]) + 12*len(values)
	}
	// Employer's separate 378-byte factor charge belongs to Contract 2, not
	// D2-A's outside-model retained allowance.
	retained += 8*127 + 72
	for _, law := range document.employeeCount {
		retained += 32 + 16*len(law.values)
	}
	retained += 64
	for _, branch := range document.income {
		retained += 128 + 2*len(branch.tupleMap)
		for _, law := range branch.laws {
			retained += 32 + 16*len(law.values)
		}
	}
	if retained > maxSocioeconomicRetainedBytes {
		return nil, fmt.Errorf("retained model charge %d exceeds %d bytes", retained, maxSocioeconomicRetainedBytes)
	}

	model := &SocioeconomicModel{companyCount: document.companyCount, employer: document.employer}
	if model.companyCount == 0 {
		return nil, fmt.Errorf("companyCount must be positive")
	}

	// Validate each consumer's age partition and its redundant compiled view.
	for i, age := range document.ages {

		if len(age.lowerBounds) == 0 || age.lowerBounds[0] != 21 || len(age.compiled) != 55 {
			return nil, fmt.Errorf("ages.%s: missing 21–75 coverage", socioeconomicAgeNames[i])
		}
		for j := 1; j < len(age.lowerBounds); j++ {
			if age.lowerBounds[j] <= age.lowerBounds[j-1] {
				return nil, fmt.Errorf("ages.%s: lower bounds are not increasing", socioeconomicAgeNames[i])
			}
		}

		bucket := 0
		for offset := range 55 {
			if bucket+1 < len(age.lowerBounds) && uint32(offset+21) >= age.lowerBounds[bucket+1] {
				bucket++
			}
			if age.compiled[offset] != uint32(bucket) {
				return nil, fmt.Errorf("ages.%s: inconsistent bucket at age %d", socioeconomicAgeNames[i], offset+21)
			}
			model.ages[i][offset] = uint8(bucket)
		}

	}

	// Validate complete categorical grids, including upstream zero-mass cells.
	rows := [...]int{
		1, 20 * len(document.ages[0].lowerBounds), 140 * len(document.ages[1].lowerBounds),
		140, 70 * len(document.ages[2].lowerBounds), 140, 10,
	}
	names := [...]string{
		"companyIndustry", "education", "primaryActivity", "employeeJobFamily",
		"employeeJobLevel", "selfEmployedJobFamily", "selfEmployedIndustry",
	}
	outputs := [...]*socioeconomicWeights{
		&model.companyIndustry, &model.education, &model.primaryActivity, &model.employeeJobFamily,
		&model.employeeJobLevel, &model.selfEmployedJobFamily, &model.selfEmployedIndustry,
	}
	for i, values := range weights {

		if len(values) != rows[i]*widths[i] {
			return nil, fmt.Errorf("%s: expected %d weights, got %d", names[i], rows[i]*widths[i], len(values))
		}

		out := outputs[i]
		out.width = widths[i]
		out.cumulative = make([]uint64, len(values))
		for row := range rows[i] {
			start := row * widths[i]
			err := compileSocioeconomicWeights(values[start:start+widths[i]], out.cumulative[start:start+widths[i]])
			if err != nil {
				return nil, fmt.Errorf("%s row %d: %w", names[i], row, err)
			}
		}

	}

	// Validate Industry-indexed count support and the exact employer factors.
	if len(document.employeeCount) != 14 {
		return nil, fmt.Errorf("employeeCount: expected 14 laws, got %d", len(document.employeeCount))
	}
	model.employeeCount = make([]socioeconomicLaw, 14)
	hasEmployerSupport := false
	for i, law := range document.employeeCount {
		compiled, err := compileSocioeconomicLaw(law)
		if err != nil {
			return nil, fmt.Errorf("employeeCount law %d: %w", i, err)
		}
		model.employeeCount[i] = compiled
		if document.companyIndustry[i] > 0 && compiled.values[len(compiled.values)-1] > 0 {
			hasEmployerSupport = true
		}
	}
	if !hasEmployerSupport {
		return nil, fmt.Errorf("employeeCount: no positive-workforce support in a positive-weight Industry")
	}
	if model.employer.cap == 0 {
		return nil, fmt.Errorf("employer.cap must be positive")
	}
	for i, factor := range model.employer.industryFactors {
		if factor == 0 {
			return nil, fmt.Errorf("employer.industryFactors[%d] must be positive", i)
		}
	}
	for i, factor := range model.employer.sizeFactors {
		if factor == 0 {
			return nil, fmt.Errorf("employer.sizeFactors[%d] must be positive", i)
		}
	}

	// Validate branch-local laws, first-occurrence IDs and complete tuple maps.
	// Employee's zero SizeBand is the sole unreachable slice: every fifth slot.
	for branch, input := range document.income {

		name := socioeconomicBranchNames[branch]
		slots := 56000
		if branch == 1 {
			slots = 2800 * len(document.ages[3].lowerBounds)
		}
		if branch >= 2 {
			slots = 140 * len(document.ages[branch+2].lowerBounds)
		}
		if len(input.tupleMap) != slots || len(input.laws) == 0 {
			return nil, fmt.Errorf("income.%s: expected %d tuple slots and nonempty laws", name, slots)
		}
		output := &model.income[branch]
		output.laws = make([]socioeconomicLaw, len(input.laws))
		for i, law := range input.laws {
			compiled, err := compileSocioeconomicLaw(law)
			if err != nil {
				return nil, fmt.Errorf("income.%s law %d: %w", name, i, err)
			}
			for j := 0; j < i; j++ {
				previous := output.laws[j]
				if slices.Equal(previous.values, compiled.values) &&
					slices.Equal(previous.cumulative, compiled.cumulative) {
					return nil, fmt.Errorf("income.%s: duplicate canonical laws %d and %d", name, j, i)
				}
			}
			output.laws[i] = compiled
		}
		output.tupleMap = make([]uint16, slots)
		nextLaw := uint32(0)
		for slot, id := range input.tupleMap {

			if branch == 0 && slot%5 == 0 {
				if id != 65535 {
					return nil, fmt.Errorf("income.e tuple %d: zero SizeBand requires sentinel 65535", slot)
				}
				output.tupleMap[slot] = 65535
				continue
			}

			if id >= uint32(len(input.laws)) || id > nextLaw {
				return nil, fmt.Errorf("income.%s tuple %d: invalid or noncanonical law reference %d", name, slot, id)
			}
			if id == nextLaw {
				nextLaw++
			}
			output.tupleMap[slot] = uint16(id)

		}
		if nextLaw != uint32(len(input.laws)) {
			return nil, fmt.Errorf("income.%s: unreferenced law %d", name, nextLaw)
		}

	}

	err := model.prepareIdentity()
	if err != nil {
		return nil, err
	}
	model.initialized = true

	return model, nil
}

func compileSocioeconomicWeights(weights []uint32, cumulative []uint64) error {

	var total, divisor uint64
	for i, weight := range weights {
		// At most 512 uint32 weights cannot overflow uint64.
		total += uint64(weight)
		divisor = greatestCommonDivisor(divisor, uint64(weight))
		cumulative[i] = total
	}

	if total == 0 {
		return fmt.Errorf("weights have zero total")
	}
	if divisor != 1 {
		return fmt.Errorf("weights are not GCD canonical")
	}

	return nil
}

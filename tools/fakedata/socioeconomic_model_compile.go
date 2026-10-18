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
// Only this Italian entry prepares the frozen national identity.
func compileSocioeconomicModel(document *socioeconomicModelJSON) (*SocioeconomicModel, error) {

	numeric, err := compileSocioeconomicNumericModel(document, socioeconomicItalyRegions)
	if err != nil {
		return nil, err
	}
	model := &SocioeconomicModel{socioeconomicNumericModel: *numeric}
	err = model.prepareIdentity()
	if err != nil {
		return nil, err
	}
	model.initialized = true

	return model, nil
}

// compileSocioeconomicNumericModel bounds and compiles internal technical input
// without a national identity, public representation, or currency semantics.
func compileSocioeconomicNumericModel(document *socioeconomicModelJSON, territories socioeconomicTerritoryDomain) (*socioeconomicNumericModel, error) {

	if document == nil {
		return nil, fmt.Errorf("missing socioeconomic numerical input")
	}
	if territories != socioeconomicItalyRegions && territories != socioeconomicUSADivisions {
		return nil, fmt.Errorf("unsupported socioeconomic territory domain %d", territories)
	}
	model := &socioeconomicNumericModel{
		territories: territories, companyCount: document.companyCount, employer: document.employer,
	}

	// Validate each consumer's age partition and its redundant compiled view.
	for i, age := range document.ages {

		maximum := 6
		if i == 1 {
			maximum = 8
		}
		if len(age.lowerBounds) > maximum || len(age.compiled) > 55 {
			return nil, fmt.Errorf("ages.%s: cardinality exceeds bounds", socioeconomicAgeNames[i])
		}
		if len(age.lowerBounds) == 0 || age.lowerBounds[0] != 21 || len(age.compiled) != 55 {
			return nil, fmt.Errorf("ages.%s: missing 21–75 coverage", socioeconomicAgeNames[i])
		}
		for j := 1; j < len(age.lowerBounds); j++ {
			if age.lowerBounds[j] <= age.lowerBounds[j-1] {
				return nil, fmt.Errorf("ages.%s: lower bounds are not increasing", socioeconomicAgeNames[i])
			}
			if age.lowerBounds[j] > 75 {
				return nil, fmt.Errorf("ages.%s: lower bounds exceed 75", socioeconomicAgeNames[i])
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

	t := int(territories)
	rows := [...]int{
		1, t * len(document.ages[0].lowerBounds), 7 * t * len(document.ages[1].lowerBounds),
		7 * t, 7 * 10 * len(document.ages[2].lowerBounds), 7 * t, 10,
	}
	slots := [6]int{t * 10 * 4 * 14 * 5, t * 10 * 14 * len(document.ages[3].lowerBounds)}
	for branch := 2; branch < len(slots); branch++ {
		slots[branch] = 7 * t * len(document.ages[branch+2].lowerBounds)
	}

	// Reserve retained storage before compilation. D2-A's numerical charges
	// include exact weights recoverable from cumulative differences. The extra
	// 64 KiB covers model headers, excess law descriptors and fixed dimensions.
	// Collection bounds below precede arithmetic and allocation, including for
	// internal input that has not passed through the Italian decoder.
	retained := 64 << 10
	weights := [...][]uint32{
		document.companyIndustry, document.education, document.primaryActivity, document.employeeJobFamily,
		document.employeeJobLevel, document.selfEmployedJobFamily, document.selfEmployedIndustry,
	}
	widths := [...]int{14, 7, 6, 10, 4, 10, 14}
	names := [...]string{
		"companyIndustry", "education", "primaryActivity", "employeeJobFamily",
		"employeeJobLevel", "selfEmployedJobFamily", "selfEmployedIndustry",
	}
	for i, values := range weights {
		if len(values) != rows[i]*widths[i] {
			return nil, fmt.Errorf("%s: expected %d weights, got %d", names[i], rows[i]*widths[i], len(values))
		}
		retained += 64 + 32*(len(values)/widths[i]) + 12*len(values)
	}
	// Employer's separate 378-byte factor charge belongs to Contract 2, not
	// D2-A's outside-model retained allowance.
	retained += 8*127 + 72
	if len(document.employeeCount) != 14 {
		return nil, fmt.Errorf("employeeCount: expected 14 laws, got %d", len(document.employeeCount))
	}
	for _, law := range document.employeeCount {
		if len(law.values) > 128 || len(law.weights) > 128 {
			return nil, fmt.Errorf("employeeCount law exceeds 128 points")
		}
		retained += 32 + 16*len(law.values)
	}
	retained += 64
	for branch, input := range document.income {

		name := socioeconomicBranchNames[branch]
		if len(input.tupleMap) != slots[branch] || len(input.laws) == 0 {
			return nil, fmt.Errorf("income.%s: expected %d tuple slots and nonempty laws", name, slots[branch])
		}
		maximum := [...]int{512, 256, 64, 64, 64, 64}
		if len(input.laws) > maximum[branch] {
			return nil, fmt.Errorf("income.%s: law count exceeds %d", name, maximum[branch])
		}
		retained += 128 + 2*len(input.tupleMap)
		for _, law := range input.laws {
			if len(law.values) > 512 || len(law.weights) > 512 {
				return nil, fmt.Errorf("income.%s law exceeds 512 points", name)
			}
			retained += 32 + 16*len(law.values)
		}

	}
	if retained > maxSocioeconomicRetainedBytes {
		return nil, fmt.Errorf("retained model charge %d exceeds %d bytes", retained, maxSocioeconomicRetainedBytes)
	}

	if model.companyCount == 0 {
		return nil, fmt.Errorf("companyCount must be positive")
	}
	if model.companyCount > 104855 {
		return nil, fmt.Errorf("companyCount exceeds 104855")
	}

	// Validate complete categorical grids, including upstream zero-mass cells.
	outputs := [...]*socioeconomicWeights{
		&model.companyIndustry, &model.education, &model.primaryActivity, &model.employeeJobFamily,
		&model.employeeJobLevel, &model.selfEmployedJobFamily, &model.selfEmployedIndustry,
	}
	for i, values := range weights {

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
		if factor == 0 || factor > 4095 {
			return nil, fmt.Errorf("employer.industryFactors[%d] must be positive and at most 4095", i)
		}
	}
	for i, factor := range model.employer.sizeFactors {
		if factor == 0 || factor > 4095 {
			return nil, fmt.Errorf("employer.sizeFactors[%d] must be positive and at most 4095", i)
		}
	}

	// Validate branch-local laws, first-occurrence IDs and complete tuple maps.
	// Employee's zero SizeBand is the sole unreachable slice: every fifth slot.
	for branch, input := range document.income {

		name := socioeconomicBranchNames[branch]
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
		output.tupleMap = make([]uint16, slots[branch])
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

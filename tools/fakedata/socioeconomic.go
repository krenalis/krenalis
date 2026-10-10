// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"math"
)

// SocioeconomicWorld provides immutable random access to anchor-date facts.
// Its base and model must not be overwritten by callers after construction.
type SocioeconomicWorld struct {
	base              *World
	model             *SocioeconomicModel
	companyCounts     []uint32
	companyIndustries []uint8
	employers         [40]socioeconomicEmployerIndex
	id                SocioeconomicWorldID
}

// NewSocioeconomicWorld constructs a complete world using bounded local work.
// It performs no model acquisition. Invalid base, model, or realized Employer
// state returns ErrInvalidSocioeconomicWorld and no world.
func NewSocioeconomicWorld(base *World, model *SocioeconomicModel) (*SocioeconomicWorld, error) {

	if base == nil || base.data == nil || base.faceCatalog == nil || base.referenceDate.IsZero() ||
		base.snapshotID == "" || base.streams == (streamFactory{}) ||
		len(base.data.ages) == 0 || len(base.data.places) == 0 {
		return nil, ErrInvalidSocioeconomicWorld
	}
	err := base.namespace.validate()
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base namespace", ErrInvalidSocioeconomicWorld)
	}
	_, err = worldIDDigest(base.worldID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base identity", ErrInvalidSocioeconomicWorld)
	}
	if model == nil || !model.initialized {
		return nil, fmt.Errorf("%w: model has not been acquired", ErrInvalidSocioeconomicWorld)
	}

	w := &SocioeconomicWorld{
		base: base, model: model,
		companyCounts: make([]uint32, model.companyCount), companyIndustries: make([]uint8, model.companyCount),
	}
	for offset := range w.companyCounts {

		index := uint64(offset) + 1
		industry, err := w.category(index, socioeconomicCompanyIndustry, model.companyIndustry, 0)
		if err != nil {
			return nil, fmt.Errorf("%w: Company Industry: %v", ErrInvalidSocioeconomicWorld, err)
		}

		count, err := w.point(index, socioeconomicCompanyEmployeeCount, model.employeeCount[industry],
			socioeconomicParent{"industry", industry})
		if err != nil {
			return nil, fmt.Errorf("%w: Company EmployeeCount: %v", ErrInvalidSocioeconomicWorld, err)
		}
		w.companyCounts[offset] = count
		w.companyIndustries[offset] = uint8(industry)

	}

	for condition := range w.employers {

		index := &w.employers[condition]
		var total uint64
		for offset, count := range w.companyCounts {

			if count == 0 {
				continue
			}
			industry := int(w.companyIndustries[offset])
			band := socioeconomicSizeBand(count)
			weight := min(uint64(count), uint64(model.employer.cap))

			for _, factor := range [...]uint16{
				model.employer.industryFactors[(condition/4)*14+industry],
				model.employer.sizeFactors[(condition%4)*4+band-1],
			} {
				if weight != 0 && uint64(factor) > math.MaxUint64/weight {
					return nil, fmt.Errorf("%w: Employer weight overflow", ErrInvalidSocioeconomicWorld)
				}
				weight *= uint64(factor)
			}
			if weight == 0 {
				continue
			}

			if total > math.MaxUint64-weight {
				return nil, fmt.Errorf("%w: Employer total overflow", ErrInvalidSocioeconomicWorld)
			}
			total += weight
			index.companies = append(index.companies, CompanyIndex(offset+1))
			index.cumulative = append(index.cumulative, total)

		}

		if total == 0 {
			return nil, fmt.Errorf("%w: Employer condition %d has zero total", ErrInvalidSocioeconomicWorld, condition)
		}

	}

	w.id, err = socioeconomicWorldIdentity(base.worldID, model.digest)
	if err != nil {
		return nil, fmt.Errorf("prepare socioeconomic identity: %v", err)
	}

	return w, nil
}

// Company returns the facts for an operational Company.
// It returns ErrInvalidSocioeconomicWorld for an invalid wrapper,
// ErrInvalidIndex for zero, or ErrCompanyIndexOutOfRange above CompanyCount.
func (w *SocioeconomicWorld) Company(index CompanyIndex) (Company, error) {

	if w == nil || w.id == "" {
		return Company{}, ErrInvalidSocioeconomicWorld
	}
	if index == 0 {
		return Company{}, ErrInvalidIndex
	}
	if uint64(index) > w.model.companyCount {
		return Company{}, ErrCompanyIndexOutOfRange
	}

	id, err := entityIDForIndex(w.base.namespace, "company", uint64(index))
	if err != nil {
		return Company{}, err
	}
	count := w.companyCounts[index-1]
	industry := w.companyIndustries[index-1]

	return Company{
		ID: id, EmployeeCount: count, Industry: socioeconomicIndustries[industry],
		SizeBand: socioeconomicSizeBands[socioeconomicSizeBand(count)],
	}, nil
}

// Facts returns complete anchor-date facts without generating a Layer 1 Person.
// It returns ErrInvalidSocioeconomicWorld for an invalid wrapper,
// ErrInvalidIndex for zero, or ErrLayer1IndexOutOfRange when index exceeds
// Layer1MaxPersonIndex.
func (w *SocioeconomicWorld) Facts(index PersonIndex) (SocioeconomicFacts, error) {
	if w == nil || w.id == "" {
		return SocioeconomicFacts{}, ErrInvalidSocioeconomicWorld
	}
	return w.facts(index)
}

// ID returns the stable identity, or empty for an invalid wrapper.
func (w *SocioeconomicWorld) ID() SocioeconomicWorldID {
	if w == nil {
		return ""
	}
	return w.id
}

// Person combines the unchanged Layer 1 Person with complete anchor-date facts.
// It returns ErrInvalidSocioeconomicWorld for an invalid wrapper,
// ErrInvalidIndex for zero, or ErrLayer1IndexOutOfRange when index exceeds
// Layer1MaxPersonIndex.
// Layer 1 failures, including ErrNoCompatiblePhoto, are propagated unchanged.
func (w *SocioeconomicWorld) Person(index PersonIndex) (SocioeconomicPerson, error) {

	if w == nil || w.id == "" {
		return SocioeconomicPerson{}, ErrInvalidSocioeconomicWorld
	}

	facts, err := w.facts(index)
	if err != nil {
		return SocioeconomicPerson{}, err
	}
	person, err := w.base.person(index)
	if err != nil {
		return SocioeconomicPerson{}, err
	}

	return SocioeconomicPerson{Person: person, Facts: facts}, nil
}

type socioeconomicEmployerIndex struct {
	companies  []CompanyIndex
	cumulative []uint64
}

// CompanyIDForIndex returns a stable Company ID for a nonzero coordinate.
// It returns ErrInvalidIdentityNamespace or ErrInvalidIndex for invalid input.
// A representable identity does not establish operational Company membership.
func CompanyIDForIndex(namespace IdentityNamespace, index CompanyIndex) (string, error) {
	return entityIDForIndex(namespace, "company", uint64(index))
}

// CompanyIndexFromID returns the Company coordinate encoded by id in namespace.
// It returns ErrInvalidIdentityNamespace, ErrMalformedSyntheticID,
// ErrUnsupportedSyntheticIDVersion, ErrWrongEntityKind,
// ErrWrongIdentityNamespace, or ErrUnassignedSyntheticID as appropriate.
func CompanyIndexFromID(namespace IdentityNamespace, id string) (CompanyIndex, error) {
	index, err := entityIndexFromID(namespace, "company", id)
	if err != nil {
		return 0, err
	}
	return CompanyIndex(index), nil
}

func socioeconomicSizeBand(count uint32) int {
	switch {
	case count == 0:
		return 0
	case count < 10:
		return 1
	case count < 50:
		return 2
	case count < 250:
		return 3
	default:
		return 4
	}
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

const socioeconomicWorldIDDomain = "fakedata/socioeconomic-world-id/v1"

// appendAsset emits one complete node-owned logical subtree, without its LF.
func (m *socioeconomicNumericModel) appendAsset(dst []byte, node int) ([]byte, error) {
	switch node {
	case socioeconomicCompanyIndustry:
		return appendSocioeconomicWeights(dst, m.companyIndustry), nil
	case socioeconomicCompanyEmployeeCount:
		dst = append(dst, `{"laws":`...)
		dst = appendSocioeconomicLaws(dst, m.employeeCount)
		return append(dst, '}'), nil
	case socioeconomicEducation:
		return appendSocioeconomicWeights(dst, m.education), nil
	case socioeconomicPrimaryActivity:
		return appendSocioeconomicWeights(dst, m.primaryActivity), nil
	case socioeconomicEmployeeJobFamily:
		return appendSocioeconomicWeights(dst, m.employeeJobFamily), nil
	case socioeconomicEmployeeJobLevel:
		return appendSocioeconomicWeights(dst, m.employeeJobLevel), nil
	case socioeconomicEmployer:
		dst = append(dst, `{"cap":`...)
		dst = strconv.AppendUint(dst, uint64(m.employer.cap), 10)
		dst = append(dst, `,"industryFactors":`...)
		dst = appendSocioeconomicIntegers(dst, m.employer.industryFactors[:])
		dst = append(dst, `,"sizeFactors":`...)
		dst = appendSocioeconomicIntegers(dst, m.employer.sizeFactors[:])
		return append(dst, '}'), nil
	case socioeconomicSelfEmployedJobFamily:
		return appendSocioeconomicWeights(dst, m.selfEmployedJobFamily), nil
	case socioeconomicSelfEmployedIndustry:
		return appendSocioeconomicWeights(dst, m.selfEmployedIndustry), nil
	case socioeconomicIncomeEmployee, socioeconomicIncomeSelfEmployed, socioeconomicIncomeUnemployed,
		socioeconomicIncomeStudent, socioeconomicIncomeRetired, socioeconomicIncomeInactive:
		income := &m.income[node-socioeconomicIncomeEmployee]
		dst = append(dst, `{"laws":`...)
		dst = appendSocioeconomicLaws(dst, income.laws)
		dst = append(dst, `,"tupleMap":`...)
		dst = appendSocioeconomicIntegers(dst, income.tupleMap)
		return append(dst, '}'), nil
	default:
		return nil, fmt.Errorf("unknown socioeconomic node %d", node)
	}
}

// canonicalModel reconstructs C(L) from validated compiled state. Every string
// is a fixed schema literal, so no general JSON string escaping is involved.
// Cumulative differences and age transitions recover the original integers.
func (m *SocioeconomicModel) canonicalModel() ([]byte, error) {

	dst := []byte(`{"ages":{`)
	for i, age := range [...]int{0, 2, 7, 6, 3, 5, 4, 1} {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, '"')
		dst = append(dst, socioeconomicAgeNames[age]...)
		dst = append(dst, '"', ':')
		dst = appendSocioeconomicAge(dst, m.ages[age])
	}
	dst = append(dst, `},"companyCount":`...)
	dst = strconv.AppendUint(dst, m.companyCount, 10)

	for _, field := range [...]struct {
		name string
		node int
	}{
		{"companyIndustry", socioeconomicCompanyIndustry},
		{"education", socioeconomicEducation},
		{"employeeCount", socioeconomicCompanyEmployeeCount},
		{"employeeJobFamily", socioeconomicEmployeeJobFamily},
		{"employeeJobLevel", socioeconomicEmployeeJobLevel},
		{"employer", socioeconomicEmployer},
		{"income", -1},
		{"primaryActivity", socioeconomicPrimaryActivity},
		{"schema", -1},
		{"selfEmployedIndustry", socioeconomicSelfEmployedIndustry},
		{"selfEmployedJobFamily", socioeconomicSelfEmployedJobFamily},
	} {

		dst = append(dst, ',', '"')
		dst = append(dst, field.name...)
		dst = append(dst, '"', ':')

		switch field.name {
		case "schema":
			dst = append(dst, `"SW-L2-D2B2-logical-v0"`...)
		case "income":
			dst = append(dst, '{')
			for i, branch := range [...]int{0, 5, 4, 1, 3, 2} {
				if i > 0 {
					dst = append(dst, ',')
				}
				dst = append(dst, '"')
				dst = append(dst, socioeconomicBranchNames[branch]...)
				dst = append(dst, '"', ':')
				var err error
				dst, err = m.appendAsset(dst, socioeconomicIncomeEmployee+branch)
				if err != nil {
					return nil, err
				}
			}
			dst = append(dst, '}')
		default:
			var err error
			dst, err = m.appendAsset(dst, field.node)
			if err != nil {
				return nil, err
			}
		}

	}

	return append(dst, '}', '\n'), nil
}

func (m *SocioeconomicModel) prepareIdentity() error {

	canonical, err := m.canonicalModel()
	if err != nil {
		return err
	}
	m.digest = sha256.Sum256(canonical)

	for node, path := range socioeconomicPaths {

		asset, err := m.appendAsset(nil, node)
		if err != nil {
			return err
		}
		component, err := BytesComponent("asset", append(asset, '\n'))
		if err != nil {
			return err
		}
		components := []Component{component}
		if age := socioeconomicNodeAges[node]; age >= 0 {
			component, err = BytesComponent("age-map", append(appendSocioeconomicAge(nil, m.ages[age]), '\n'))
			if err != nil {
				return err
			}
			components = append(components, component)
		}

		m.fingerprints[node], err = canonicalDigest("fakedata/"+path+"/model", components)
		if err != nil {
			return err
		}

	}

	return nil
}

func appendSocioeconomicAge(dst []byte, age [55]uint8) []byte {
	dst = append(dst, `{"compiled":`...)
	dst = appendSocioeconomicIntegers(dst, age[:])
	dst = append(dst, `,"lowerBounds":[21`...)
	for i := 1; i < len(age); i++ {
		if age[i] != age[i-1] {
			dst = append(dst, ',')
			dst = strconv.AppendUint(dst, uint64(i+21), 10)
		}
	}
	return append(dst, ']', '}')
}

func appendSocioeconomicIntegers[T ~uint8 | ~uint16 | ~uint32](dst []byte, values []T) []byte {
	dst = append(dst, '[')
	for i, value := range values {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = strconv.AppendUint(dst, uint64(value), 10)
	}
	return append(dst, ']')
}

func appendSocioeconomicLaws(dst []byte, laws []socioeconomicLaw) []byte {
	dst = append(dst, '[')
	for i, law := range laws {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, `{"values":`...)
		dst = appendSocioeconomicIntegers(dst, law.values)
		dst = append(dst, `,"weights":`...)
		dst = appendSocioeconomicMasses(dst, law.cumulative, len(law.cumulative))
		dst = append(dst, '}')
	}
	return append(dst, ']')
}

func appendSocioeconomicMasses(dst []byte, cumulative []uint64, width int) []byte {
	dst = append(dst, '[')
	var previous uint64
	for i, value := range cumulative {
		if i > 0 {
			dst = append(dst, ',')
		}
		if i%width == 0 {
			previous = 0
		}
		dst = strconv.AppendUint(dst, value-previous, 10)
		previous = value
	}
	return append(dst, ']')
}

func appendSocioeconomicWeights(dst []byte, weights socioeconomicWeights) []byte {
	dst = append(dst, `{"weights":`...)
	dst = appendSocioeconomicMasses(dst, weights.cumulative, weights.width)
	return append(dst, '}')
}

func socioeconomicWorldIdentity(base WorldID, model [sha256.Size]byte) (SocioeconomicWorldID, error) {
	return socioeconomicWorldIdentityForProtocol(base, model, PopulationAnchorDate, "socioeconomic-v1")
}

// socioeconomicWorldIdentityForProtocol keeps one identity codec while
// binding the national anchor and protocol independently of stream roots.
func socioeconomicWorldIdentityForProtocol(base WorldID, model [sha256.Size]byte, anchorDate, protocolVersion string) (SocioeconomicWorldID, error) {

	baseDigest, err := worldIDDigest(base)
	if err != nil {
		return "", err
	}
	modelComponent, err := BytesComponent("model", model[:])
	if err != nil {
		return "", err
	}
	anchor, err := StringComponent("population-anchor-date", anchorDate)
	if err != nil {
		return "", err
	}
	protocol, err := StringComponent("protocol", protocolVersion)
	if err != nil {
		return "", err
	}
	world, err := BytesComponent("world-id", baseDigest[:])
	if err != nil {
		return "", err
	}

	digest, err := canonicalDigest(socioeconomicWorldIDDomain, []Component{modelComponent, anchor, protocol, world})
	if err != nil {
		return "", err
	}

	return SocioeconomicWorldID("socioeconomic-world_v1_" + hex.EncodeToString(digest[:])), nil
}

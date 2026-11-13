// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

var socioeconomicAgeNames = [...]string{
	"Education", "PrimaryActivity", "Employee.JobLevel",
	"Income[s]", "Income[u]", "Income[t]", "Income[r]", "Income[i]",
}

var socioeconomicBranchNames = [...]string{"e", "s", "u", "t", "r", "i"}

// The transport uses the closed logical-v0 schema. Dimensions are positional
// in D2-A order, not category strings or source-publication metadata.
type socioeconomicModelJSON struct {
	companyCount          uint64
	ages                  [8]socioeconomicAgeJSON
	companyIndustry       []uint32
	employeeCount         []socioeconomicLawJSON
	education             []uint32
	primaryActivity       []uint32
	employeeJobFamily     []uint32
	employeeJobLevel      []uint32
	selfEmployedJobFamily []uint32
	selfEmployedIndustry  []uint32
	employer              socioeconomicEmployerParameters
	income                [6]socioeconomicIncomeJSON
}

type socioeconomicAgeJSON struct {
	lowerBounds []uint32
	compiled    []uint32
}

type socioeconomicIncomeJSON struct {
	tupleMap []uint32
	laws     []socioeconomicLawJSON
}

type socioeconomicLawJSON struct {
	values  []uint32
	weights []uint32
}

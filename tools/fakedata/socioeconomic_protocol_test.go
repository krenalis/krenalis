// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

type socioeconomicProtocolVectors struct {
	Model                    json.RawMessage
	CanonicalModelByteLength int
	LogicalModelDigest       string
	BaseRoot                 string
	BaseWorldID              WorldID
	WorldSeed                uint64
	Namespace                IdentityNamespace
	SocioeconomicWorldID     SocioeconomicWorldID
	WorldRecordHex           string
	Nodes                    []socioeconomicNodeVector
	Coordinates              []struct {
		Kind            string
		Index           uint64
		CoordinateBEHex string
		EntityID        string
		StreamDigest    string
		Operational     bool
	}
	Boundaries []struct {
		Family     string
		Cumulative []uint64
		Draw       uint64
		Output     uint64
		Position   int
	}
	Primitive []struct {
		Bound           uint64
		Draw            uint64
		InitialStateHex string
		FinalStateHex   string
		RawCandidates   []string
	}
	Framing struct {
		Domain    string
		RecordHex string
		Digest    string
		Fields    []struct {
			Hex  string
			Kind byte
			Name string
		}
	}
	InactiveOnlyMutation struct {
		ChangedNodeFingerprints []string
		ChangedStreamDigests    []string
		LogicalModelDigest      string
		SocioeconomicWorldID    SocioeconomicWorldID
		WorldRecordHex          string
	}
}

type socioeconomicNodeVector struct {
	Node                string
	Path                string
	Kind                string
	Coordinate          uint64
	Parents             map[string]int
	ModelFingerprint    string
	ModelRecordHex      string
	DependencyDigest    string
	DependencyRecordHex string
	StreamDigest        string
	StreamRecordHex     string
	InitialStateHex     string
	FinalStateHex       string
	RawCandidates       []string
	Cumulative          []uint64
	Values              []uint64
	Total               uint64
	BoundedDraw         uint64
	SelectedPosition    int
	SelectedValue       uint64
}

// TestSocioeconomicProtocol reproduces every frozen protocol fixture with the
// actual compiled-model serializer, stream factory and bounded sampler.
// The synthetic model is used only as a protocol fixture, not as a real world.
func TestSocioeconomicProtocol(t *testing.T) {

	v := readSocioeconomicVectors(t)
	m, err := decodeSocioeconomicModel(bytes.NewReader(v.Model))
	if err != nil {
		t.Fatalf("expected compiled protocol fixture, got %v", err)
	}
	canonical, err := m.canonicalModel()
	if err != nil {
		t.Fatalf("expected canonical model, got %v", err)
	}
	if len(canonical) != v.CanonicalModelByteLength || hex.EncodeToString(m.digest[:]) != v.LogicalModelDigest {
		t.Fatalf("expected model %s (%d bytes), got %x (%d bytes)",
			v.LogicalModelDigest, v.CanonicalModelByteLength, m.digest, len(canonical))
	}
	base := mustLayer1World(t, PopulationAnchorDate, v.WorldSeed)
	if base.worldID != v.BaseWorldID || hex.EncodeToString(base.streams.root[:]) != v.BaseRoot {
		t.Fatalf("expected vector base identity/root, got %s/%x", base.worldID, base.streams.root)
	}
	w := &SocioeconomicWorld{base: base, model: m}
	assertSocioeconomicWorldVector(t, base.worldID, m.digest, v.SocioeconomicWorldID, v.WorldRecordHex)
	if len(v.Nodes) != 15 {
		t.Fatalf("expected fifteen node vectors, got %d", len(v.Nodes))
	}
	for node, vector := range v.Nodes {
		t.Run(vector.Node, func(t *testing.T) {

			if vector.Path != socioeconomicPaths[node] ||
				hex.EncodeToString(m.fingerprints[node][:]) != vector.ModelFingerprint {
				t.Fatalf("expected path %s and fingerprint %s, got %s and %x",
					vector.Path, vector.ModelFingerprint, socioeconomicPaths[node], m.fingerprints[node])
			}
			asset, err := m.appendAsset(nil, node)
			if err != nil {
				t.Fatalf("expected node asset, got %v", err)
			}
			fields := []Component{mustBytesComponent(t, "asset", append(asset, '\n'))}
			if age := socioeconomicNodeAges[node]; age >= 0 {
				ageBytes := append(appendSocioeconomicAge(nil, m.ages[age]), '\n')
				fields = append(fields, mustBytesComponent(t, "age-map", ageBytes))
			}
			assertSocioeconomicRecord(t, "fakedata/"+vector.Path+"/model", fields, vector.ModelRecordHex)

			parents, dependencies := socioeconomicVectorParents(t, m.fingerprints[node], vector.Parents)
			assertSocioeconomicRecord(t, streamDependenciesDomain, dependencies, vector.DependencyRecordHex)
			digest, err := dependencyDigest(dependencies)
			if err != nil {
				t.Fatalf("expected dependency digest, got %v", err)
			}
			if hex.EncodeToString(digest[:]) != vector.DependencyDigest {
				t.Fatalf("expected dependency digest %s, got %x", vector.DependencyDigest, digest)
			}
			streamFields := []Component{
				mustBytesComponent(t, "dependencies", digest[:]),
				mustUint64Component(t, "entity-index", vector.Coordinate),
				mustStringComponent(t, "entity-kind", vector.Kind), mustStringComponent(t, "path", vector.Path),
				mustBytesComponent(t, "root", base.streams.root[:]), mustStringComponent(t, "scope", "entity"),
			}
			assertSocioeconomicRecord(t, streamSeedDomain, streamFields, vector.StreamRecordHex)
			streamDigest, err := deriveStreamDigest(
				base.streams.root, "entity", vector.Kind, vector.Coordinate, vector.Path, dependencies)
			if err != nil {
				t.Fatalf("expected stream digest, got %v", err)
			}
			if hex.EncodeToString(streamDigest[:]) != vector.StreamDigest {
				t.Fatalf("expected stream digest %s, got %x", vector.StreamDigest, streamDigest)
			}
			rng, err := w.stream(vector.Coordinate, node, parents...)
			if err != nil {
				t.Fatalf("expected runtime node stream, got %v", err)
			}
			if fmt.Sprintf("%016x", rng.state) != vector.InitialStateHex {
				t.Fatalf("expected initial state %s, got %016x", vector.InitialStateHex, rng.state)
			}
			draw, err := rng.Uint64n(vector.Total)
			if err != nil {
				t.Fatalf("expected bounded draw, got %v", err)
			}
			if draw != vector.BoundedDraw || fmt.Sprintf("%016x", rng.state) != vector.FinalStateHex {
				t.Fatalf("expected draw/state %d/%s, got %d/%016x",
					vector.BoundedDraw, vector.FinalStateHex, draw, rng.state)
			}
			position, err := w.sample(vector.Coordinate, node, vector.Cumulative, parents...)
			if err != nil {
				t.Fatalf("expected runtime finite selection, got %v", err)
			}
			if position != vector.SelectedPosition || vector.Values[position] != vector.SelectedValue {
				t.Fatalf("expected position/value %d/%d, got %d/%d",
					vector.SelectedPosition, vector.SelectedValue, position, vector.Values[position])
			}
			assertSocioeconomicRawCandidates(t, vector.InitialStateHex, vector.RawCandidates)

		})
	}

	t.Run("coordinates", func(t *testing.T) {
		if len(v.Coordinates) != 6 {
			t.Fatalf("expected six coordinate vectors, got %d", len(v.Coordinates))
		}
		for _, vector := range v.Coordinates {
			got := hex.EncodeToString(binary.BigEndian.AppendUint64(nil, vector.Index))
			if got != vector.CoordinateBEHex {
				t.Fatalf("expected coordinate bytes %s, got %s", vector.CoordinateBEHex, got)
			}
			id, err := entityIDForIndex(v.Namespace, vector.Kind, vector.Index)
			if err != nil {
				t.Fatalf("expected entity identity, got %v", err)
			}
			index, err := entityIndexFromID(v.Namespace, vector.Kind, id)
			if err != nil {
				t.Fatalf("expected identity inverse, got %v", err)
			}
			if id != vector.EntityID || index != vector.Index {
				t.Fatalf("expected identity/index %s/%d, got %s/%d", vector.EntityID, vector.Index, id, index)
			}
			node := socioeconomicEducation
			if vector.Kind == "company" {
				node = socioeconomicCompanyIndustry
			}
			_, dependencies := socioeconomicVectorParents(t, m.fingerprints[node], v.Nodes[node].Parents)
			digest, err := deriveStreamDigest(
				base.streams.root, "entity", vector.Kind, vector.Index, v.Nodes[node].Path, dependencies)
			if err != nil {
				t.Fatalf("expected coordinate stream, got %v", err)
			}
			if hex.EncodeToString(digest[:]) != vector.StreamDigest {
				t.Fatalf("expected coordinate digest %s, got %x", vector.StreamDigest, digest)
			}
		}
	})

	t.Run("boundaries", func(t *testing.T) {
		if len(v.Boundaries) != 16 {
			t.Fatalf("expected sixteen boundaries, got %d", len(v.Boundaries))
		}
		for _, vector := range v.Boundaries {
			position := socioeconomicPosition(vector.Cumulative, vector.Draw)
			outputs := []uint64{0, 1, 2, 3, 4, 5, 6}
			switch vector.Family {
			case "count":
				outputs = []uint64{0, 9, 10}
			case "employer":
				outputs = []uint64{1, 2, 3}
			case "income":
				outputs = []uint64{0, 100, 1000}
			}
			if position != vector.Position || outputs[position] != vector.Output {
				t.Fatalf("expected %s position/output %d/%d, got %d/%d",
					vector.Family, vector.Position, vector.Output, position, outputs[position])
			}
		}
	})

	t.Run("primitive", func(t *testing.T) {
		for _, vector := range v.Primitive {
			state, err := strconv.ParseUint(vector.InitialStateHex, 16, 64)
			if err != nil {
				t.Fatalf("expected initial state, got %v", err)
			}
			rng := splitMix64{state: state}
			draw, err := rng.Uint64n(vector.Bound)
			if err != nil {
				t.Fatalf("expected bounded draw, got %v", err)
			}
			if draw != vector.Draw || fmt.Sprintf("%016x", rng.state) != vector.FinalStateHex {
				t.Fatalf("expected draw/state %d/%s, got %d/%016x", vector.Draw, vector.FinalStateHex, draw, rng.state)
			}
			assertSocioeconomicRawCandidates(t, vector.InitialStateHex, vector.RawCandidates)
		}
	})

	t.Run("framing", func(t *testing.T) {
		var fields []Component
		for _, field := range v.Framing.Fields {
			value, err := hex.DecodeString(field.Hex)
			if err != nil {
				t.Fatalf("expected component bytes, got %v", err)
			}
			fields = append(fields, Component{name: field.Name, kind: field.Kind, value: value, valid: true})
		}
		assertSocioeconomicRecord(t, v.Framing.Domain, fields, v.Framing.RecordHex)
		digest, err := canonicalDigest(v.Framing.Domain, fields)
		if err != nil {
			t.Fatalf("expected framing digest, got %v", err)
		}
		if hex.EncodeToString(digest[:]) != v.Framing.Digest {
			t.Fatalf("expected framing digest %s, got %x", v.Framing.Digest, digest)
		}
	})

	t.Run("inactive-only mutation", func(t *testing.T) {
		changed, err := decodeSocioeconomicModel(bytes.NewReader(v.Model))
		if err != nil {
			t.Fatalf("expected independent fixture, got %v", err)
		}
		changed.income[5].laws[0].values[1] = 101
		err = changed.prepareIdentity()
		if err != nil {
			t.Fatalf("expected changed identity, got %v", err)
		}
		if hex.EncodeToString(changed.digest[:]) != v.InactiveOnlyMutation.LogicalModelDigest {
			t.Fatalf("expected changed digest %s, got %x", v.InactiveOnlyMutation.LogicalModelDigest, changed.digest)
		}
		var fingerprints, streams []string
		for node, vector := range v.Nodes {
			if m.fingerprints[node] != changed.fingerprints[node] {
				fingerprints = append(fingerprints, vector.Node)
			}
			_, dependencies := socioeconomicVectorParents(t, changed.fingerprints[node], vector.Parents)
			digest, err := deriveStreamDigest(
				base.streams.root, "entity", vector.Kind, vector.Coordinate, vector.Path, dependencies)
			if err != nil {
				t.Fatalf("expected changed node stream, got %v", err)
			}
			if hex.EncodeToString(digest[:]) != vector.StreamDigest {
				streams = append(streams, vector.Node)
			}
		}
		if !reflect.DeepEqual(fingerprints, v.InactiveOnlyMutation.ChangedNodeFingerprints) ||
			!reflect.DeepEqual(streams, v.InactiveOnlyMutation.ChangedStreamDigests) {
			t.Fatalf("expected only prescribed node/stream changes, got %v/%v", fingerprints, streams)
		}
		assertSocioeconomicWorldVector(t, base.worldID, changed.digest,
			v.InactiveOnlyMutation.SocioeconomicWorldID, v.InactiveOnlyMutation.WorldRecordHex)
	})

}

// TestSocioeconomicProtocolLawLookup checks compiled row and law selection
// against the frozen outputs, independently of the direct sampler assertions.
func TestSocioeconomicProtocolLawLookup(t *testing.T) {

	v := readSocioeconomicVectors(t)
	m, err := decodeSocioeconomicModel(bytes.NewReader(v.Model))
	if err != nil {
		t.Fatalf("expected compiled protocol fixture, got %v", err)
	}
	w := &SocioeconomicWorld{base: mustLayer1World(t, PopulationAnchorDate, v.WorldSeed), model: m}

	// Protocol sections 7–8 prescribe these dimensions and outermost-first
	// parent orders. Every age map in this synthetic fixture has one bucket.
	dimensions := map[string]int{
		"age-bucket": 1, "education": 7, "region": 20, "job-family": 10,
		"job-level": 4, "industry": 14, "size-band": 5,
	}
	for _, test := range []struct {
		node  int
		axes  []string
		table socioeconomicWeights
	}{
		{socioeconomicCompanyIndustry, nil, m.companyIndustry},
		{socioeconomicCompanyEmployeeCount, []string{"industry"}, socioeconomicWeights{}},
		{socioeconomicEducation, []string{"age-bucket", "region"}, m.education},
		{socioeconomicPrimaryActivity, []string{"age-bucket", "education", "region"}, m.primaryActivity},
		{socioeconomicEmployeeJobFamily, []string{"education", "region"}, m.employeeJobFamily},
		{socioeconomicEmployeeJobLevel, []string{"age-bucket", "education", "job-family"}, m.employeeJobLevel},
		{socioeconomicEmployer, []string{"job-family", "job-level"}, socioeconomicWeights{}},
		{socioeconomicSelfEmployedJobFamily, []string{"education", "region"}, m.selfEmployedJobFamily},
		{socioeconomicSelfEmployedIndustry, []string{"job-family"}, m.selfEmployedIndustry},
		{socioeconomicIncomeEmployee,
			[]string{"region", "job-family", "job-level", "industry", "size-band"}, socioeconomicWeights{}},
		{socioeconomicIncomeSelfEmployed,
			[]string{"age-bucket", "region", "job-family", "industry"}, socioeconomicWeights{}},
		{socioeconomicIncomeUnemployed, []string{"age-bucket", "education", "region"}, socioeconomicWeights{}},
		{socioeconomicIncomeStudent, []string{"age-bucket", "education", "region"}, socioeconomicWeights{}},
		{socioeconomicIncomeRetired, []string{"age-bucket", "education", "region"}, socioeconomicWeights{}},
		{socioeconomicIncomeInactive, []string{"age-bucket", "education", "region"}, socioeconomicWeights{}},
	} {

		vector := v.Nodes[test.node]
		t.Run(vector.Node, func(t *testing.T) {

			parents, _ := socioeconomicVectorParents(t, m.fingerprints[test.node], vector.Parents)
			offset := 0
			for _, axis := range test.axes {
				offset = offset*dimensions[axis] + vector.Parents[axis]
			}

			var value uint64
			var err error
			switch {
			case test.node == socioeconomicCompanyEmployeeCount:
				var count uint32
				count, err = w.point(vector.Coordinate, test.node, m.employeeCount[offset], parents...)
				value = uint64(count)
			case test.node == socioeconomicEmployer:
				// This vector prescribes an artificial three-candidate selector,
				// separate from the forty indexes of a constructed world.
				position, sampleErr := w.sample(vector.Coordinate, test.node, vector.Cumulative, parents...)
				if sampleErr != nil {
					t.Fatalf("expected artificial Employer selection, got %v", sampleErr)
				}
				value = vector.Values[position]
			case test.node >= socioeconomicIncomeEmployee:
				var income AnnualIncomeEUR
				income, err = w.income(vector.Coordinate, test.node-socioeconomicIncomeEmployee, offset, parents...)
				value = uint64(income)
			default:
				var ordinal int
				ordinal, err = w.category(vector.Coordinate, test.node, test.table, offset, parents...)
				value = uint64(ordinal)
			}
			if err != nil {
				t.Fatalf("expected compiled row/law lookup, got %v", err)
			}
			if value != vector.SelectedValue {
				t.Fatalf("expected selected value %d, got %d", vector.SelectedValue, value)
			}

		})

	}

}

func assertSocioeconomicRawCandidates(t *testing.T, initial string, candidates []string) {
	t.Helper()
	state, err := strconv.ParseUint(initial, 16, 64)
	if err != nil {
		t.Fatalf("expected initial state, got %v", err)
	}
	rng := splitMix64{state: state}
	for _, expected := range candidates {
		if got := fmt.Sprintf("%016x", rng.Uint64()); got != expected {
			t.Fatalf("expected raw candidate %s, got %s", expected, got)
		}
	}
}

func assertSocioeconomicRecord(t *testing.T, domain string, fields []Component, expected string) {
	t.Helper()
	record, err := canonicalRecord(domain, fields)
	if err != nil {
		t.Fatalf("expected KFD1 record, got %v", err)
	}
	if got := hex.EncodeToString(record); got != expected {
		t.Fatalf("expected exact %s record %s, got %s", domain, expected, got)
	}
}

func assertSocioeconomicWorldVector(t *testing.T, base WorldID, model [32]byte, expected SocioeconomicWorldID, record string) {
	t.Helper()
	id, err := socioeconomicWorldIdentity(base, model)
	if err != nil {
		t.Fatalf("expected socioeconomic identity, got %v", err)
	}
	if id != expected {
		t.Fatalf("expected socioeconomic identity %s, got %s", expected, id)
	}
	baseDigest, err := worldIDDigest(base)
	if err != nil {
		t.Fatalf("expected base identity, got %v", err)
	}
	fields := []Component{
		mustBytesComponent(t, "model", model[:]), mustStringComponent(t, "population-anchor-date", "2026-01-01"),
		mustStringComponent(t, "protocol", "socioeconomic-v1"), mustBytesComponent(t, "world-id", baseDigest[:]),
	}
	assertSocioeconomicRecord(t, socioeconomicWorldIDDomain, fields, record)
}

func readSocioeconomicVectors(t *testing.T) socioeconomicProtocolVectors {
	t.Helper()
	data, err := os.ReadFile("../../SYNTHETIC_WORLD_V1_LAYER2_DETERMINISTIC_PROTOCOL_VECTORS_v1.json")
	if err != nil {
		t.Fatalf("expected frozen reference vectors, got %v", err)
	}
	const expected = "131b0de96484f61900f27daa3e182b9a526a1d1200f4b2819c084ceed2f2b218"
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != expected {
		t.Fatalf("expected approved reference-vector digest, got %s", got)
	}
	var vectors socioeconomicProtocolVectors
	err = json.Unmarshal(data, &vectors)
	if err != nil {
		t.Fatalf("expected parsed reference vectors, got %v", err)
	}
	return vectors
}

func socioeconomicVectorParents(t *testing.T, fingerprint [32]byte, values map[string]int) ([]socioeconomicParent, []Component) {
	t.Helper()
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	slices.Sort(names)
	var parents []socioeconomicParent
	components := []Component{mustBytesComponent(t, "model", fingerprint[:])}
	for _, name := range names {
		parents = append(parents, socioeconomicParent{name, values[name]})
		components = append(components, mustUint64Component(t, name, uint64(values[name])))
	}
	return parents, components
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"runtime"
	"sync/atomic"
	"testing"
)

// BenchmarkBase32Decode measures fixed-width uint64 decoding.
func BenchmarkBase32Decode(b *testing.B) {

	b.ReportAllocs()
	var decoded uint64
	var err error
	for b.Loop() {
		decoded, err = decodeUint64Base32("14d2pf2dbsqqg")
		if err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(decoded)

}

// BenchmarkBase32Encode measures fixed-width uint64 encoding.
func BenchmarkBase32Encode(b *testing.B) {

	b.ReportAllocs()
	value := uint64(0x123456789abcdef0)
	var encoded string
	for b.Loop() {
		encoded = encodeUint64Base32(value)
		value += 0x9e3779b97f4a7c15
	}
	runtime.KeepAlive(encoded)

}

// BenchmarkCanonicalRecord measures canonical KFD1 encoding.
func BenchmarkCanonicalRecord(b *testing.B) {

	components := benchmarkComponents(b)
	b.ReportAllocs()
	var record []byte
	var err error
	for b.Loop() {
		record, err = canonicalRecord("benchmark/v1", components)
		if err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(record)

}

// BenchmarkEntityStream measures independent entity stream derivation.
func BenchmarkEntityStream(b *testing.B) {

	factory := benchmarkStreamFactory(b)
	b.ReportAllocs()
	var rng splitMix64
	var err error
	index := uint64(1)
	for b.Loop() {
		rng, err = factory.entityStream("person", index, "identity/name")
		if err != nil {
			b.Fatal(err)
		}
		index += 1_000_003
	}
	runtime.KeepAlive(rng)

}

// BenchmarkIndependentIndexesParallel measures concurrent derivation and one draw from independent entity streams.
func BenchmarkIndependentIndexesParallel(b *testing.B) {

	factory := benchmarkStreamFactory(b)
	var next atomic.Uint64
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {

		var value uint64
		for pb.Next() {
			index := next.Add(1)
			rng, err := factory.entityStream("person", index, "identity/name")
			if err != nil {
				b.Error(err)
				return
			}
			value = rng.Uint64()
		}
		runtime.KeepAlive(value)

	})

}

// BenchmarkIndependentIndexesSerial measures serial derivation and one draw from independent entity streams.
func BenchmarkIndependentIndexesSerial(b *testing.B) {

	factory := benchmarkStreamFactory(b)
	b.ReportAllocs()
	index := uint64(1)
	var value uint64
	for b.Loop() {
		rng, err := factory.entityStream("person", index, "identity/name")
		if err != nil {
			b.Fatal(err)
		}
		value = rng.Uint64()
		index += 1_000_003
	}
	runtime.KeepAlive(value)

}

// BenchmarkPersonIDForward measures synthetic person ID generation for low and high indices.
func BenchmarkPersonIDForward(b *testing.B) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	indices := [...]PersonIndex{1, 42, 1537291, ^PersonIndex(0)}
	b.ReportAllocs()
	var id string
	var err error
	i := 0
	for b.Loop() {
		id, err = PersonIDForIndex(namespace, indices[i&3])
		if err != nil {
			b.Fatal(err)
		}
		i++
	}
	runtime.KeepAlive(id)

}

// BenchmarkPersonIDInverse measures synthetic person ID parsing for IDs derived from low and high indices.
func BenchmarkPersonIDInverse(b *testing.B) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	ids := [...]string{
		"person_v1_krenalis-demo_00000001_1z4yz7emgwe48",
		"person_v1_krenalis-demo_00000001_4r0n8d46pgtwb",
		"person_v1_krenalis-demo_00000001_c5m6wxatk5862",
		"person_v1_krenalis-demo_00000001_7dzg0z9r9ekm6",
	}
	b.ReportAllocs()
	var index PersonIndex
	var err error
	i := 0
	for b.Loop() {
		index, err = PersonIndexFromID(namespace, ids[i&3])
		if err != nil {
			b.Fatal(err)
		}
		i++
	}
	runtime.KeepAlive(index)

}

// BenchmarkRangeValidation measures validation of small ranges and ranges near the upper boundary.
func BenchmarkRangeValidation(b *testing.B) {

	b.ReportAllocs()
	starts := [...]PersonIndex{1, 42, 1 << 63, ^PersonIndex(0) - 1024}
	counts := [...]uint64{0, 100, 1024, 1025}
	var personRange PersonRange
	var err error
	i := 0
	for b.Loop() {
		personRange, err = NewPersonRange(starts[i&3], counts[i&3])
		if err != nil {
			b.Fatal(err)
		}
		i++
	}
	runtime.KeepAlive(personRange)

}

// BenchmarkSHA256Derivation measures a complete canonical SHA-256 derivation.
func BenchmarkSHA256Derivation(b *testing.B) {

	components := benchmarkComponents(b)
	b.ReportAllocs()
	var digest [32]byte
	var err error
	for b.Loop() {
		digest, err = canonicalDigest("benchmark/v1", components)
		if err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(digest)

}

// BenchmarkSplitMix64 measures raw SplitMix64 generation.
func BenchmarkSplitMix64(b *testing.B) {

	b.ReportAllocs()
	rng := splitMix64{}
	var value uint64
	for b.Loop() {
		value = rng.Uint64()
	}
	runtime.KeepAlive(value)

}

// BenchmarkStreamRoot measures stream-factory root derivation.
func BenchmarkStreamRoot(b *testing.B) {

	components := benchmarkComponents(b)
	b.ReportAllocs()
	var factory streamFactory
	var err error
	for b.Loop() {
		factory, err = newStreamFactory(components...)
		if err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(factory)

}

// BenchmarkUint64n measures normative bounded sampling.
func BenchmarkUint64n(b *testing.B) {

	b.ReportAllocs()
	rng := splitMix64{}
	var value uint64
	var err error
	for b.Loop() {
		value, err = rng.Uint64n(10_000)
		if err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(value)

}

// BenchmarkWeightedIndex measures ordered weighted sampling.
func BenchmarkWeightedIndex(b *testing.B) {

	b.ReportAllocs()
	rng := splitMix64{}
	weights := []uint64{0, 3, 1, 6, 2, 9}
	var index int
	var err error
	for b.Loop() {
		index, err = rng.WeightedIndex(weights)
		if err != nil {
			b.Fatal(err)
		}
	}
	runtime.KeepAlive(index)

}

func benchmarkComponents(b *testing.B) []Component {

	b.Helper()
	modelVersion, err := StringComponent("world-model-version", "world-v1")
	if err != nil {
		b.Fatal(err)
	}
	seed, err := Uint64Component("world-seed", 726381)
	if err != nil {
		b.Fatal(err)
	}

	return []Component{modelVersion, seed}
}

func benchmarkStreamFactory(b *testing.B) streamFactory {

	b.Helper()
	factory, err := newStreamFactory(benchmarkComponents(b)...)
	if err != nil {
		b.Fatal(err)
	}

	return factory
}

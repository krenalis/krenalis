// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"encoding/hex"
	"errors"
	"reflect"
	"sync"
	"testing"
)

// TestDependencyStreamGolden verifies dependency-specific digest and stream vectors.
func TestDependencyStreamGolden(t *testing.T) {

	faceVersion := mustStringComponent(t, "face-catalog-version", "faces-v1")
	faceDigest := make([]byte, 32)
	for i := range faceDigest {
		faceDigest[i] = byte(i)
	}
	faceSHA256 := mustBytesComponent(t, "face-catalog-sha256", faceDigest)
	dependencies := []Component{faceVersion, faceSHA256}

	digest, err := dependencyDigest(dependencies)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	got := hex.EncodeToString(digest[:])
	want := "67f5176ac71016327871d2e05be9ace4921db95e09067ffddeec18a9a2eb7827"
	if got != want {
		t.Fatalf("expected dependency digest %s, got %s", want, got)
	}

	factory := goldenStreamFactory(t)
	streamDigest, err := deriveStreamDigest(factory.root, "entity", "person", 42, "photo/select", dependencies)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	got = hex.EncodeToString(streamDigest[:])
	want = "3927dbb94df6a2b6f14205f2d02fe360df1188b807a446af133c8e283a61e4c0"
	if got != want {
		t.Fatalf("expected stream digest %s, got %s", want, got)
	}
	rng, err := factory.entityStream("person", 42, "photo/select", dependencies...)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if rng.state != 0x3927dbb94df6a2b6 {
		t.Fatalf("expected initial state 3927dbb94df6a2b6, got %016x", rng.state)
	}

}

// TestStreamConcurrentEquivalence verifies request-order independence and equivalence between
// serial and concurrent derivation.
func TestStreamConcurrentEquivalence(t *testing.T) {

	factory := goldenStreamFactory(t)
	const count = 512
	serial := make([][4]uint64, count)
	for i := range serial {
		rng, err := factory.entityStream("person", uint64(i+1), "identity/name")
		if err != nil {
			t.Fatalf("expected nil error at index %d, got %v", i+1, err)
		}
		for j := range serial[i] {
			serial[i][j] = rng.Uint64()
		}
	}

	concurrent := make([][4]uint64, count)
	var wg sync.WaitGroup
	for i := count - 1; i >= 0; i-- {
		wg.Add(1)
		go func(position int) {
			defer wg.Done()
			rng, err := factory.entityStream("person", uint64(position+1), "identity/name")
			if err != nil {
				t.Errorf("expected nil error at index %d, got %v", position+1, err)
				return
			}
			for j := range concurrent[position] {
				concurrent[position][j] = rng.Uint64()
			}
		}(i)
	}
	wg.Wait()

	if !reflect.DeepEqual(serial, concurrent) {
		t.Fatal("expected serial and reverse-order concurrent derivations to match")
	}

}

// TestStreamDerivationGolden verifies every parent, child, sibling, world, and entity stream vector.
func TestStreamDerivationGolden(t *testing.T) {

	factory := goldenStreamFactory(t)
	tests := []struct {
		scope        string
		kind         string
		index        uint64
		path         string
		digest       string
		initialState uint64
	}{
		{
			"entity",
			"person",
			42,
			"identity",
			"314d1fa0a367dbca9080f01e2edcf49187a6d301d21c4d2e5e3b426e916759f1",
			0x314d1fa0a367dbca,
		},
		{
			"entity",
			"person",
			42,
			"identity/name",
			"89a10f64d4a955cafdf001e69120780c7794560db1e6b6f57808d10642a25a16",
			0x89a10f64d4a955ca,
		},
		{
			"entity",
			"person",
			42,
			"identity/email",
			"d138758916487db4d9550b1538aa3fd7a0711431047f2b00d82515d3f7049d70",
			0xd138758916487db4,
		},
		{
			"world",
			"",
			0,
			"companies/size",
			"13ab6ef2cac6579aaf9f4e13b06eb54c9b5af818ff696c5a659f41ed8ce907d9",
			0x13ab6ef2cac6579a,
		},
		{
			"entity",
			"person",
			42,
			"photo/select",
			"d631c77121eaa1ed24ca822923a1f20d7ba141873265356da66a370077528af7",
			0xd631c77121eaa1ed,
		},
	}

	for _, test := range tests {
		digest, err := deriveStreamDigest(factory.root, test.scope, test.kind, test.index, test.path, nil)
		if err != nil {
			t.Fatalf("expected nil error for %s/%s, got %v", test.scope, test.path, err)
		}
		if got := hex.EncodeToString(digest[:]); got != test.digest {
			t.Fatalf("expected %s/%s digest %s, got %s", test.scope, test.path, test.digest, got)
		}
		rng, err := factory.stream(test.scope, test.kind, test.index, test.path)
		if err != nil {
			t.Fatalf("expected nil stream error for %s/%s, got %v", test.scope, test.path, err)
		}
		if rng.state != test.initialState {
			t.Fatalf("expected %s/%s initial state %016x, got %016x", test.scope, test.path, test.initialState, rng.state)
		}
	}
	worldRNG, err := factory.worldStream("companies/size")
	if err != nil {
		t.Fatalf("expected nil world stream error, got %v", err)
	}
	if worldRNG.state != 0x13ab6ef2cac6579a {
		t.Fatalf("expected world stream state 13ab6ef2cac6579a, got %016x", worldRNG.state)
	}

}

// TestStreamDoesNotImplicitlyUseComposedIDs verifies that composed IDs enter a root only as explicit components.
func TestStreamDoesNotImplicitlyUseComposedIDs(t *testing.T) {

	factory := goldenStreamFactory(t)
	baseline := streamSequence(t, factory, "identity", nil, 3)
	worldComponent := mustStringComponent(t, "world-id", string(goldenWorldID))
	withWorldID, err := newStreamFactory(
		mustStringComponent(t, "world-model-version", "world-v1"),
		mustUint64Component(t, "world-seed", 726381),
		worldComponent,
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reflect.DeepEqual(baseline, streamSequence(t, withWorldID, "identity", nil, 3)) {
		t.Fatal("expected explicitly supplied WorldID component to change the root")
	}
	snapshotID, err := ComposeSnapshotID(goldenWorldID)
	if err != nil {
		t.Fatalf("expected nil snapshot error, got %v", err)
	}
	withSnapshotID, err := newStreamFactory(
		mustStringComponent(t, "world-model-version", "world-v1"),
		mustUint64Component(t, "world-seed", 726381),
		mustStringComponent(t, "snapshot-id", string(snapshotID)),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reflect.DeepEqual(baseline, streamSequence(t, withSnapshotID, "identity", nil, 3)) {
		t.Fatal("expected explicitly supplied SnapshotID component to change the root")
	}
	if got := streamSequence(t, factory, "identity", nil, 3); !reflect.DeepEqual(got, baseline) {
		t.Fatalf("expected factory without explicit WorldID to remain unchanged, got %x and %x", baseline, got)
	}

}

// TestStreamNonInterference verifies root, path, dependencies, request order, and the behavior of copied RNGs.
func TestStreamNonInterference(t *testing.T) {

	factory := goldenStreamFactory(t)
	identityBefore := streamSequence(t, factory, "identity", nil, 8)
	photoDependency := mustStringComponent(t, "face-catalog-version", "faces-v2")
	photo := streamSequence(t, factory, "photo/select", nil, 8)
	identityAfter := streamSequence(t, factory, "identity", nil, 8)
	if !reflect.DeepEqual(identityBefore, identityAfter) {
		t.Fatalf("expected identity stream to survive unrelated request, got %x and %x", identityBefore, identityAfter)
	}
	identityDependency := streamSequence(t, factory, "identity", []Component{photoDependency}, 8)
	if reflect.DeepEqual(identityBefore, identityDependency) {
		t.Fatal("expected an explicit dependency to change its stream")
	}
	if reflect.DeepEqual(identityBefore, photo) {
		t.Fatal("expected sibling paths to have independent sequences")
	}

	changedRoot, err := newStreamFactory(
		mustStringComponent(t, "world-model-version", "world-v1"),
		mustUint64Component(t, "world-seed", 726382),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reflect.DeepEqual(identityBefore, streamSequence(t, changedRoot, "identity", nil, 8)) {
		t.Fatal("expected a root change to change the derived stream")
	}

	sameNameRoot, err := newStreamFactory(mustStringComponent(t, "version", "root"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	dependency := mustStringComponent(t, "version", "dependency")
	if got := streamSequence(t, sameNameRoot, "identity", []Component{dependency}, 1); len(got) != 1 {
		t.Fatalf("expected root/dependency namespaces to allow the same component name, got %v", got)
	}

	rng, err := factory.entityStream("person", 42, "identity/name")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	copyRNG := rng
	first := rng.Uint64()
	if got := copyRNG.Uint64(); got != first {
		t.Fatalf("expected copied RNG first output %016x, got %016x", first, got)
	}
	_ = rng.Uint64()
	if got := copyRNG.Uint64(); got == rng.Uint64() {
		t.Fatalf("expected independently advanced copied RNGs, both produced %016x", got)
	}

}

// TestStreamRootGolden verifies the authoritative root and empty-dependency digests.
func TestStreamRootGolden(t *testing.T) {

	modelVersion := mustStringComponent(t, "world-model-version", "world-v1")
	seed := mustUint64Component(t, "world-seed", 726381)
	factory, err := newStreamFactory(modelVersion, seed)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	got := hex.EncodeToString(factory.root[:])
	want := "30248f6f4dfbdb86e37a6b68aba18e4d79c99a933af7211358577db734fdfdda"
	if got != want {
		t.Fatalf("expected root %s, got %s", want, got)
	}
	if _, err := newStreamFactory(); err != nil {
		t.Fatalf("expected empty root component set to be valid, got %v", err)
	}

	zeroFactory, err := newStreamFactory(modelVersion, mustUint64Component(t, "world-seed", 0))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	got = hex.EncodeToString(zeroFactory.root[:])
	want = "c57683746540e57d257b5013932962044930a263cfab4877e82d0640f841a39a"
	if got != want {
		t.Fatalf("expected zero-seed root %s, got %s", want, got)
	}

	emptyDependencies, err := dependencyDigest(nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	got = hex.EncodeToString(emptyDependencies[:])
	want = "1e5c1aa6f425463a6f91ad0cde21751118e17caf05c60919cae02bb364f735fb"
	if got != want {
		t.Fatalf("expected empty dependency digest %s, got %s", want, got)
	}

}

// TestStreamValidation verifies scope, coordinate, path, and component failures.
func TestStreamValidation(t *testing.T) {

	factory := goldenStreamFactory(t)
	tests := []struct {
		name  string
		scope string
		kind  string
		index uint64
		path  string
		want  error
	}{
		{"invalid scope", "global", "", 0, "identity", ErrInvalidStreamScope},
		{"world kind", "world", "person", 0, "identity", ErrInvalidStreamScope},
		{"world index", "world", "", 1, "identity", ErrInvalidStreamScope},
		{"entity zero", "entity", "person", 0, "identity", ErrInvalidIndex},
		{"entity empty kind", "entity", "", 1, "identity", ErrInvalidStreamScope},
		{"entity invalid kind", "entity", "Person", 1, "identity", ErrInvalidStreamScope},
		{"empty path", "entity", "person", 1, "", ErrInvalidStreamPath},
		{"leading slash", "entity", "person", 1, "/identity", ErrInvalidStreamPath},
		{"trailing slash", "entity", "person", 1, "identity/", ErrInvalidStreamPath},
		{"double slash", "entity", "person", 1, "identity//name", ErrInvalidStreamPath},
		{"uppercase", "entity", "person", 1, "Identity", ErrInvalidStreamPath},
		{"long segment", "entity", "person", 1, "abcdefghijklmnopqrstuvwxyz1234567", ErrInvalidStreamPath},
		{"nine segments", "entity", "person", 1, "a/b/c/d/e/f/g/h/i", ErrInvalidStreamPath},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			_, err := factory.stream(test.scope, test.kind, test.index, test.path)
			if err != nil {
				if !errors.Is(err, test.want) {
					t.Fatalf("expected %v, got %v", test.want, err)
				}
				return
			}
			t.Fatalf("expected %v, got nil", test.want)

		})

	}

	_, err := factory.entityStream("person", 1, "identity", Component{})
	if err != nil {
		if !errors.Is(err, ErrInvalidDeterministicComponent) {
			t.Fatalf("expected invalid component, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid component, got nil")
	}
	_, err = newStreamFactory(Component{})
	if err != nil {
		if !errors.Is(err, ErrInvalidDeterministicComponent) {
			t.Fatalf("expected invalid root component, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid root component, got nil")
	}

}

func goldenStreamFactory(t *testing.T) streamFactory {

	t.Helper()
	factory, err := newStreamFactory(
		mustStringComponent(t, "world-model-version", "world-v1"),
		mustUint64Component(t, "world-seed", 726381),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	return factory
}

func streamSequence(t *testing.T, factory streamFactory, path string, dependencies []Component, count int) []uint64 {

	t.Helper()
	rng, err := factory.entityStream("person", 42, path, dependencies...)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	sequence := make([]uint64, count)
	for i := range sequence {
		sequence[i] = rng.Uint64()
	}

	return sequence
}

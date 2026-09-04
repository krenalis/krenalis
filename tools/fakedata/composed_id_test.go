// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"errors"
	"testing"
)

const goldenWorldID = WorldID("world_v1_e0ee5788fe9d83aaaadbab5146e888bf43c88848938485937d1b988b3dfc3d42")

// TestComposedIDComponentValidation verifies reserved names, duplicates, zero components, and change sensitivity.
func TestComposedIDComponentValidation(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	reservedNames := []string{
		"identity-name",
		"identity-generation",
		"synthetic-id-protocol",
		"deterministic-protocol",
	}
	for _, name := range reservedNames {
		reserved := mustStringComponent(t, name, "reserved")
		_, err := ComposeWorldID(namespace, reserved)
		if err != nil {
			if !errors.Is(err, ErrDuplicateComponent) {
				t.Fatalf("expected duplicate component for reserved name %q, got %v", name, err)
			}
			continue
		}
		t.Fatalf("expected duplicate component for reserved name %q, got nil", name)
	}

	duplicate := mustStringComponent(t, "market", "italy")
	_, err := ComposeWorldID(namespace, duplicate, duplicate)
	if err != nil {
		if !errors.Is(err, ErrDuplicateComponent) {
			t.Fatalf("expected duplicate component, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected duplicate component, got nil")
	}
	_, err = ComposeWorldID(namespace, Component{})
	if err != nil {
		if !errors.Is(err, ErrInvalidDeterministicComponent) {
			t.Fatalf("expected invalid component, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid component, got nil")
	}
	_, err = ComposeWorldID(IdentityNamespace{}, duplicate)
	if err != nil {
		if !errors.Is(err, ErrInvalidIdentityNamespace) {
			t.Fatalf("expected invalid namespace, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid namespace, got nil")
	}

	reservedWorld := mustBytesComponent(t, "world-id", bytes.Repeat([]byte{1}, 32))
	_, err = ComposeSnapshotID(goldenWorldID, reservedWorld)
	if err != nil {
		if !errors.Is(err, ErrDuplicateComponent) {
			t.Fatalf("expected duplicate component, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected duplicate component, got nil")
	}
	_, err = ComposeSnapshotID(goldenWorldID, Component{})
	if err != nil {
		if !errors.Is(err, ErrInvalidDeterministicComponent) {
			t.Fatalf("expected invalid component, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid component, got nil")
	}

	one, err := ComposeWorldID(namespace, mustStringComponent(t, "market", "italy"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	two, err := ComposeWorldID(namespace, mustStringComponent(t, "market", "france"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if one == two {
		t.Fatal("expected modified world component to change WorldID")
	}

	snapshotOne, err := ComposeSnapshotID(one, mustStringComponent(t, "reference-date", "2026-01-01"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	snapshotTwo, err := ComposeSnapshotID(one, mustStringComponent(t, "reference-date", "2026-01-02"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if snapshotOne == snapshotTwo {
		t.Fatal("expected modified snapshot component to change SnapshotID")
	}

}

// TestComposedIDsAllowNoSuppliedComponents verifies that omitting supplied components is valid.
func TestComposedIDsAllowNoSuppliedComponents(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	worldID, err := ComposeWorldID(namespace)
	if err != nil {
		t.Fatalf("expected nil world error, got %v", err)
	}
	if _, err := worldIDDigest(worldID); err != nil {
		t.Fatalf("expected syntactically valid WorldID, got %v", err)
	}
	snapshotID, err := ComposeSnapshotID(worldID)
	if err != nil {
		t.Fatalf("expected nil snapshot error, got %v", err)
	}
	if _, err := snapshotIDDigest(snapshotID); err != nil {
		t.Fatalf("expected syntactically valid SnapshotID, got %v", err)
	}

}

// TestComposedIDsDoNotAffectEntityIDs verifies that composed IDs do not affect entity IDs.
func TestComposedIDsDoNotAffectEntityIDs(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	before, err := PersonIDForIndex(namespace, 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	worldID, err := ComposeWorldID(namespace,
		mustStringComponent(t, "market", "italy"),
		mustStringComponent(t, "world-model-version", "world-v99"),
		mustUint64Component(t, "world-seed", 999),
	)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	_, err = ComposeSnapshotID(worldID, mustStringComponent(t, "reference-date", "2030-12-31"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	after, err := PersonIDForIndex(namespace, 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if before != after {
		t.Fatalf("expected stable entity ID %s, got %s", before, after)
	}

}

// TestComposedIDValidation verifies WorldID and SnapshotID syntax validation.
func TestComposedIDValidation(t *testing.T) {

	invalidWorldIDs := []WorldID{
		"",
		"world_v1_",
		"world_v2_e0ee5788fe9d83aaaadbab5146e888bf43c88848938485937d1b988b3dfc3d42",
		"world_v1_E0ee5788fe9d83aaaadbab5146e888bf43c88848938485937d1b988b3dfc3d42",
		"world_v1_e0ee5788fe9d83aaaadbab5146e888bf43c88848938485937d1b988b3dfc3d4g",
		"world_v1_e0ee5788fe9d83aaaadbab5146e888bf43c88848938485937d1b988b3dfc3d420",
		" world_v1_e0ee5788fe9d83aaaadbab5146e888bf43c88848938485937d1b988b3dfc3d42",
	}
	for _, worldID := range invalidWorldIDs {
		_, err := ComposeSnapshotID(worldID)
		if err != nil {
			if !errors.Is(err, ErrInvalidComposedID) {
				t.Fatalf("expected invalid composed ID for %q, got %v", worldID, err)
			}
			continue
		}
		t.Fatalf("expected invalid composed ID for %q, got nil", worldID)
	}

	validSnapshot := SnapshotID("snapshot_v1_3aae60767efa04fed9faea664a13cc35be0c3dcb9d3bb28fa48fe92b9c7252cb")
	if _, err := snapshotIDDigest(validSnapshot); err != nil {
		t.Fatalf("expected valid snapshot ID, got %v", err)
	}
	invalidSnapshots := []SnapshotID{
		"",
		"snapshot_v1_3aae60767efa04fed9faea664a13cc35be0c3dcb9d3bb28fa48fe92b9c7252c",
		"snapshot_v2_3aae60767efa04fed9faea664a13cc35be0c3dcb9d3bb28fa48fe92b9c7252cb",
		"snapshot_v1_3AAe60767efa04fed9faea664a13cc35be0c3dcb9d3bb28fa48fe92b9c7252cb",
	}
	for _, snapshotID := range invalidSnapshots {
		_, err := snapshotIDDigest(snapshotID)
		if err != nil {
			if !errors.Is(err, ErrInvalidComposedID) {
				t.Fatalf("expected invalid composed ID for %q, got %v", snapshotID, err)
			}
			continue
		}
		t.Fatalf("expected invalid composed ID for %q, got nil", snapshotID)
	}

}

// TestComposeSnapshotIDGolden verifies the authoritative snapshot vector and that the raw WorldID digest is used.
func TestComposeSnapshotIDGolden(t *testing.T) {

	referenceDate := mustStringComponent(t, "reference-date", "2026-01-01")
	faceVersion := mustStringComponent(t, "face-catalog-version", "faces-v1")
	faceDigest := make([]byte, 32)
	for i := range faceDigest {
		faceDigest[i] = byte(i)
	}
	faceSHA256 := mustBytesComponent(t, "face-catalog-sha256", faceDigest)

	snapshotID, err := ComposeSnapshotID(goldenWorldID, referenceDate, faceVersion, faceSHA256)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	want := SnapshotID("snapshot_v1_3aae60767efa04fed9faea664a13cc35be0c3dcb9d3bb28fa48fe92b9c7252cb")
	if snapshotID != want {
		t.Fatalf("expected %s, got %s", want, snapshotID)
	}

	reordered, err := ComposeSnapshotID(goldenWorldID, faceSHA256, referenceDate, faceVersion)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reordered != want {
		t.Fatalf("expected reordered ID %s, got %s", want, reordered)
	}

}

// TestComposeWorldIDGolden verifies the authoritative world vectors and input-order independence.
func TestComposeWorldIDGolden(t *testing.T) {

	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	market := mustStringComponent(t, "market", "italy")
	personVersion := mustStringComponent(t, "person-spec-version", "person-v1")
	modelVersion := mustStringComponent(t, "world-model-version", "world-v1")
	seed := mustUint64Component(t, "world-seed", 726381)

	worldID, err := ComposeWorldID(namespace, market, personVersion, modelVersion, seed)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if worldID != goldenWorldID {
		t.Fatalf("expected %s, got %s", goldenWorldID, worldID)
	}

	reordered, err := ComposeWorldID(namespace, seed, modelVersion, market, personVersion)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if reordered != goldenWorldID {
		t.Fatalf("expected reordered ID %s, got %s", goldenWorldID, reordered)
	}

	zeroSeed := mustUint64Component(t, "world-seed", 0)
	worldID, err = ComposeWorldID(namespace, market, personVersion, modelVersion, zeroSeed)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	wantZero := WorldID("world_v1_016030ff93490055ecaeb6330131542d2fac84ad5ff894528994a54840cf6818")
	if worldID != wantZero {
		t.Fatalf("expected %s, got %s", wantZero, worldID)
	}

}

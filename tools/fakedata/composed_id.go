// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	snapshotIDDomain = "fakedata/snapshot-id/v1"
	snapshotIDPrefix = "snapshot_v1_"
	worldIDDomain    = "fakedata/world-id/v1"
	worldIDPrefix    = "world_v1_"
)

// WorldID identifies a non-temporal synthetic-world configuration.
type WorldID string

// SnapshotID identifies one observable snapshot of a synthetic world.
type SnapshotID string

// ComposeSnapshotID returns the deterministic fingerprint of worldID and the supplied snapshot components.
// It returns ErrInvalidComposedID, ErrInvalidDeterministicComponent, ErrDuplicateComponent, or
// ErrComponentCountOverflow as appropriate.
func ComposeSnapshotID(worldID WorldID, components ...Component) (SnapshotID, error) {

	err := validateComponentCount(uint64(len(components)) + 1)
	if err != nil {
		return "", err
	}
	worldDigest, err := worldIDDigest(worldID)
	if err != nil {
		return "", err
	}
	worldComponent, err := BytesComponent("world-id", worldDigest[:])
	if err != nil {
		return "", err
	}

	all := make([]Component, 0, len(components)+1)
	all = append(all, worldComponent)
	all = append(all, components...)
	digest, err := canonicalDigest(snapshotIDDomain, all)
	if err != nil {
		return "", err
	}

	return SnapshotID(snapshotIDPrefix + hex.EncodeToString(digest[:])), nil
}

// ComposeWorldID returns the deterministic fingerprint of namespace and the supplied world components.
// It returns ErrInvalidIdentityNamespace, ErrInvalidDeterministicComponent, ErrDuplicateComponent, or
// ErrComponentCountOverflow as appropriate.
func ComposeWorldID(namespace IdentityNamespace, components ...Component) (WorldID, error) {

	err := namespace.validate()
	if err != nil {
		return "", err
	}
	err = validateComponentCount(uint64(len(components)) + 4)
	if err != nil {
		return "", err
	}

	identityName, err := StringComponent("identity-name", namespace.Name)
	if err != nil {
		return "", err
	}
	identityGeneration, err := Uint64Component("identity-generation", uint64(namespace.Generation))
	if err != nil {
		return "", err
	}
	syntheticProtocol, err := StringComponent("synthetic-id-protocol", syntheticIDProtocol)
	if err != nil {
		return "", err
	}
	deterministicVersion, err := StringComponent("deterministic-protocol", deterministicProtocol)
	if err != nil {
		return "", err
	}

	all := make([]Component, 0, len(components)+4)
	all = append(all, identityName, identityGeneration, syntheticProtocol, deterministicVersion)
	all = append(all, components...)
	digest, err := canonicalDigest(worldIDDomain, all)
	if err != nil {
		return "", err
	}

	return WorldID(worldIDPrefix + hex.EncodeToString(digest[:])), nil
}

func composedIDDigest(value, prefix string) ([sha256.Size]byte, error) {

	if len(value) != len(prefix)+sha256.Size*2 || value[:len(prefix)] != prefix || !lowerHex(value[len(prefix):]) {
		return [sha256.Size]byte{}, ErrInvalidComposedID
	}

	var digest [sha256.Size]byte
	_, err := hex.Decode(digest[:], []byte(value[len(prefix):]))
	if err != nil {
		return [sha256.Size]byte{}, ErrInvalidComposedID
	}

	return digest, nil
}

func snapshotIDDigest(snapshotID SnapshotID) ([sha256.Size]byte, error) {
	return composedIDDigest(string(snapshotID), snapshotIDPrefix)
}

func worldIDDigest(worldID WorldID) ([sha256.Size]byte, error) {
	return composedIDDigest(string(worldID), worldIDPrefix)
}

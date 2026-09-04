// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

const (
	streamDependenciesDomain = "fakedata/stream-dependencies/v1"
	streamRootDomain         = "fakedata/stream-root/v1"
	streamScopeEntity        = "entity"
	streamScopeWorld         = "world"
	streamSeedDomain         = "fakedata/stream-seed/v1"
)

type streamFactory struct {
	root [sha256.Size]byte
}

// newStreamFactory returns an immutable stream factory derived from components.
func newStreamFactory(components ...Component) (streamFactory, error) {

	digest, err := canonicalDigest(streamRootDomain, components)
	if err != nil {
		return streamFactory{}, err
	}

	return streamFactory{root: digest}, nil
}

// entityStream derives a fresh entity-scoped RNG.
func (f streamFactory) entityStream(kind string, index uint64, path string, dependencies ...Component) (splitMix64, error) {
	return f.stream(streamScopeEntity, kind, index, path, dependencies...)
}

// stream derives a fresh RNG for the given scope, entity kind/index, path, and dependency set.
func (f streamFactory) stream(scope, entityKind string, entityIndex uint64, path string, dependencies ...Component) (splitMix64, error) {

	digest, err := deriveStreamDigest(f.root, scope, entityKind, entityIndex, path, dependencies)
	if err != nil {
		return splitMix64{}, err
	}

	return splitMix64{state: binary.BigEndian.Uint64(digest[:8])}, nil
}

// worldStream derives a fresh world-scoped RNG.
func (f streamFactory) worldStream(path string, dependencies ...Component) (splitMix64, error) {
	return f.stream(streamScopeWorld, "", 0, path, dependencies...)
}

// dependencyDigest returns the KFD1 digest of a stream's dependency components.
func dependencyDigest(components []Component) ([sha256.Size]byte, error) {
	return canonicalDigest(streamDependenciesDomain, components)
}

// deriveStreamDigest returns the digest for one fully qualified deterministic stream.
func deriveStreamDigest(root [sha256.Size]byte, scope, entityKind string, entityIndex uint64, path string, dependencies []Component) ([sha256.Size]byte, error) {

	switch scope {
	case streamScopeWorld:
		if entityKind != "" || entityIndex != 0 {
			return [sha256.Size]byte{}, ErrInvalidStreamScope
		}
	case streamScopeEntity:
		if entityIndex == 0 {
			return [sha256.Size]byte{}, ErrInvalidIndex
		}
		if !validLowerIdentifier(entityKind, 16) {
			return [sha256.Size]byte{}, ErrInvalidStreamScope
		}
	default:
		return [sha256.Size]byte{}, ErrInvalidStreamScope
	}
	if !validStreamPath(path) {
		return [sha256.Size]byte{}, ErrInvalidStreamPath
	}

	dependenciesDigest, err := dependencyDigest(dependencies)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	dependencyComponent, err := BytesComponent("dependencies", dependenciesDigest[:])
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	entityIndexComponent, err := Uint64Component("entity-index", entityIndex)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	entityKindComponent, err := StringComponent("entity-kind", entityKind)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	pathComponent, err := StringComponent("path", path)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	rootComponent, err := BytesComponent("root", root[:])
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	scopeComponent, err := StringComponent("scope", scope)
	if err != nil {
		return [sha256.Size]byte{}, err
	}

	return canonicalDigest(streamSeedDomain, []Component{
		dependencyComponent,
		entityIndexComponent,
		entityKindComponent,
		pathComponent,
		rootComponent,
		scopeComponent,
	})
}

func validStreamPath(path string) bool {

	segments := strings.Split(path, "/")
	if len(segments) == 0 || len(segments) > 8 {
		return false
	}
	for _, segment := range segments {
		if !validLowerIdentifier(segment, 32) {
			return false
		}
	}

	return true
}

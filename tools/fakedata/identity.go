// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	base32Alphabet        = "0123456789abcdefghjkmnpqrstvwxyz"
	deterministicProtocol = "det-v1"
	personEntityKind      = "person"
	syntheticIDKeyDomain  = "fakedata/synthetic-id-key/v1"
	syntheticIDProtocol   = "sid-v1"
	syntheticIDVersion    = "v1"
)

// PersonIndex is a one-based coordinate for a synthetic person.
type PersonIndex uint64

// IdentityNamespace defines the scope in which synthetic identities remain stable.
type IdentityNamespace struct {
	Name       string
	Generation uint32
}

// NewIdentityNamespace validates and returns an identity namespace.
// It returns ErrInvalidIdentityNamespace for an invalid name or generation.
func NewIdentityNamespace(name string, generation uint32) (IdentityNamespace, error) {

	namespace := IdentityNamespace{Name: name, Generation: generation}
	err := namespace.validate()
	if err != nil {
		return IdentityNamespace{}, err
	}

	return namespace, nil
}

// validate checks the complete identity namespace grammar.
func (namespace IdentityNamespace) validate() error {

	if !validLowerIdentifier(namespace.Name, 32) || namespace.Generation == 0 {
		return ErrInvalidIdentityNamespace
	}

	return nil
}

type parsedEntityID struct {
	kind       string
	namespace  string
	generation uint32
	token      uint64
}

func canonicalDigit(c byte) (byte, bool) {

	position := strings.IndexByte(base32Alphabet, c)
	if position < 0 {
		return 0, false
	}

	return byte(position), true
}

// decodeUint64Base32 decodes the canonical fixed-width base32 representation of a uint64.
func decodeUint64Base32(token string) (uint64, error) {

	if len(token) != 13 {
		return 0, ErrMalformedSyntheticID
	}

	first, ok := canonicalDigit(token[0])
	if !ok || first > 15 {
		return 0, ErrMalformedSyntheticID
	}

	value := uint64(first)
	for i := 1; i < 13; i++ {
		digit, ok := canonicalDigit(token[i])
		if !ok {
			return 0, ErrMalformedSyntheticID
		}
		value = (value << 5) | uint64(digit)
	}

	return value, nil
}

// encodeUint64Base32 returns the canonical fixed-width base32 representation of value.
func encodeUint64Base32(value uint64) string {

	var out [13]byte
	for pos := 12; pos >= 0; pos-- {
		out[pos] = base32Alphabet[value&31]
		value >>= 5
	}

	return string(out[:])
}

// entityIDForIndex returns the synthetic ID assigned to one entity coordinate.
func entityIDForIndex(namespace IdentityNamespace, kind string, index uint64) (string, error) {

	err := namespace.validate()
	if err != nil {
		return "", err
	}
	if !validLowerIdentifier(kind, 16) {
		return "", ErrWrongEntityKind
	}
	if index == 0 {
		return "", ErrInvalidIndex
	}

	_, key, err := identityKey(namespace, kind)
	if err != nil {
		return "", err
	}
	token := encodeUint64Base32(mix64((index - 1) ^ key))

	return kind + "_" + syntheticIDVersion + "_" + namespace.Name + "_" +
		fmt.Sprintf("%08x", namespace.Generation) + "_" + token, nil
}

// entityIndexFromID returns the entity coordinate encoded by id.
func entityIndexFromID(namespace IdentityNamespace, expectedKind, id string) (uint64, error) {

	err := namespace.validate()
	if err != nil {
		return 0, err
	}
	if !validLowerIdentifier(expectedKind, 16) {
		return 0, ErrWrongEntityKind
	}

	parsed, err := parseEntityID(id)
	if err != nil {
		return 0, err
	}
	if parsed.kind != expectedKind {
		return 0, ErrWrongEntityKind
	}
	if parsed.namespace != namespace.Name || parsed.generation != namespace.Generation {
		return 0, ErrWrongIdentityNamespace
	}

	_, key, err := identityKey(namespace, expectedKind)
	if err != nil {
		return 0, err
	}
	x := unmix64(parsed.token) ^ key
	if x == math.MaxUint64 {
		return 0, ErrUnassignedSyntheticID
	}

	return x + 1, nil
}

// identityKey returns the complete identity-key digest and its first eight bytes interpreted as a big-endian uint64.
func identityKey(namespace IdentityNamespace, kind string) ([sha256.Size]byte, uint64, error) {

	err := namespace.validate()
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}
	if !validLowerIdentifier(kind, 16) {
		return [sha256.Size]byte{}, 0, ErrWrongEntityKind
	}

	entityKind, err := StringComponent("entity-kind", kind)
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}
	namespaceGeneration, err := Uint64Component("namespace-generation", uint64(namespace.Generation))
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}
	namespaceName, err := StringComponent("namespace-name", namespace.Name)
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}

	digest, err := canonicalDigest(syntheticIDKeyDomain, []Component{entityKind, namespaceGeneration, namespaceName})
	if err != nil {
		return [sha256.Size]byte{}, 0, err
	}

	return digest, binary.BigEndian.Uint64(digest[:8]), nil
}

func lowerHex(value string) bool {

	for i := 0; i < len(value); i++ {
		c := value[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}

// mix64 is the bijective mixer used by the synthetic identity protocol.
func mix64(z uint64) uint64 {

	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb

	return z ^ (z >> 31)
}

// parseEntityID validates and splits one synthetic entity ID.
func parseEntityID(id string) (parsedEntityID, error) {

	// Split the ID into its five fields.
	kind, remainder, ok := strings.Cut(id, "_")
	if !ok {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}
	version, remainder, ok := strings.Cut(remainder, "_")
	if !ok {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}
	namespace, remainder, ok := strings.Cut(remainder, "_")
	if !ok {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}
	generationHex, tokenString, ok := strings.Cut(remainder, "_")

	// Validate field structure, kind, and namespace.
	if !ok || strings.Contains(tokenString, "_") ||
		!validLowerIdentifier(kind, 16) || !validLowerIdentifier(namespace, 32) {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}

	// Validate the version.
	if version != syntheticIDVersion {
		if !validSyntheticIDVersion(version) {
			return parsedEntityID{}, ErrMalformedSyntheticID
		}
		return parsedEntityID{}, ErrUnsupportedSyntheticIDVersion
	}

	// Parse the generation and token.
	if len(generationHex) != 8 || !lowerHex(generationHex) {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}
	generation, err := strconv.ParseUint(generationHex, 16, 32)
	if err != nil {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}
	if generation == 0 {
		return parsedEntityID{}, ErrMalformedSyntheticID
	}
	token, err := decodeUint64Base32(tokenString)
	if err != nil {
		return parsedEntityID{}, err
	}

	return parsedEntityID{kind: kind, namespace: namespace, generation: uint32(generation), token: token}, nil
}

// PersonIDForIndex returns the stable synthetic person ID assigned to index in namespace.
// It returns ErrInvalidIdentityNamespace or ErrInvalidIndex for invalid arguments.
func PersonIDForIndex(namespace IdentityNamespace, index PersonIndex) (string, error) {
	return entityIDForIndex(namespace, personEntityKind, uint64(index))
}

// PersonIndexFromID returns the one-based person index encoded by id in namespace.
//
// It returns ErrInvalidIdentityNamespace, ErrMalformedSyntheticID, ErrUnsupportedSyntheticIDVersion,
// ErrWrongEntityKind, ErrWrongIdentityNamespace, or ErrUnassignedSyntheticID as appropriate.
func PersonIndexFromID(namespace IdentityNamespace, id string) (PersonIndex, error) {

	index, err := entityIndexFromID(namespace, personEntityKind, id)
	if err != nil {
		return 0, err
	}

	return PersonIndex(index), nil
}

// undoXorShiftRight reverses x ^= x >> shift.
func undoXorShiftRight(x uint64, shift uint) uint64 {

	for s := shift; s < 64; s <<= 1 {
		x ^= x >> s
	}

	return x
}

// unmix64 reverses the synthetic identity mixer.
func unmix64(z uint64) uint64 {

	z = undoXorShiftRight(z, 31)
	z *= 0x319642b2d24d8ec3
	z = undoXorShiftRight(z, 27)
	z *= 0x96de1b173f119089

	return undoXorShiftRight(z, 30)
}

func validSyntheticIDVersion(version string) bool {

	if len(version) < 2 || version[0] != 'v' {
		return false
	}
	for i := 1; i < len(version); i++ {
		if version[i] < '0' || version[i] > '9' {
			return false
		}
	}

	return true
}

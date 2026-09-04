// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"unicode/utf8"
)

const (
	componentKindString byte = 0x01
	componentKindUint64 byte = 0x02
	componentKindBytes  byte = 0x03
)

// Component represents an opaque named value in a canonical deterministic record. Its zero value is invalid.
type Component struct {
	name  string
	kind  byte
	value []byte
	valid bool
}

// BytesComponent returns a byte-valued deterministic component.
// It returns ErrInvalidDeterministicComponent for an invalid name.
func BytesComponent(name string, value []byte) (Component, error) {

	if !validLowerIdentifier(name, 64) {
		return Component{}, fmt.Errorf("%w: invalid component name", ErrInvalidDeterministicComponent)
	}

	return Component{name: name, kind: componentKindBytes, value: append([]byte(nil), value...), valid: true}, nil
}

// StringComponent returns a string-valued deterministic component.
// It returns ErrInvalidDeterministicComponent for an invalid name or value.
func StringComponent(name, value string) (Component, error) {

	if !validLowerIdentifier(name, 64) {
		return Component{}, fmt.Errorf("%w: invalid component name", ErrInvalidDeterministicComponent)
	}
	if !utf8.ValidString(value) {
		return Component{}, fmt.Errorf("%w: string value is not valid UTF-8", ErrInvalidDeterministicComponent)
	}

	return Component{name: name, kind: componentKindString, value: []byte(value), valid: true}, nil
}

// Uint64Component returns an unsigned-integer-valued deterministic component.
// It returns ErrInvalidDeterministicComponent for an invalid name.
func Uint64Component(name string, value uint64) (Component, error) {

	if !validLowerIdentifier(name, 64) {
		return Component{}, fmt.Errorf("%w: invalid component name", ErrInvalidDeterministicComponent)
	}

	encoded := make([]byte, 8)
	binary.BigEndian.PutUint64(encoded, value)

	return Component{name: name, kind: componentKindUint64, value: encoded, valid: true}, nil
}

// validate checks that the component has exactly the representation produced by its constructor.
func (c Component) validate() error {

	if !c.valid || !validLowerIdentifier(c.name, 64) {
		return ErrInvalidDeterministicComponent
	}

	switch c.kind {
	case componentKindString:
		if !utf8.Valid(c.value) {
			return ErrInvalidDeterministicComponent
		}
	case componentKindUint64:
		if len(c.value) != 8 {
			return ErrInvalidDeterministicComponent
		}
	case componentKindBytes:
	default:
		return ErrInvalidDeterministicComponent
	}

	return nil
}

func appendUint32(dst []byte, value uint32) []byte {
	return binary.BigEndian.AppendUint32(dst, value)
}

func appendUint64(dst []byte, value uint64) []byte {
	return binary.BigEndian.AppendUint64(dst, value)
}

// canonicalDigest returns the SHA-256 digest of a KFD1 record.
func canonicalDigest(domain string, components []Component) ([sha256.Size]byte, error) {

	record, err := canonicalRecord(domain, components)
	if err != nil {
		return [sha256.Size]byte{}, err
	}

	return sha256.Sum256(record), nil
}

// canonicalRecord returns the canonical KFD1 encoding of a deterministic record.
func canonicalRecord(domain string, components []Component) ([]byte, error) {

	if !validDomain(domain) {
		return nil, fmt.Errorf("%w: invalid domain", ErrInvalidDeterministicComponent)
	}
	err := validateComponentCount(uint64(len(components)))
	if err != nil {
		return nil, err
	}

	ordered := append([]Component(nil), components...)
	for _, component := range ordered {
		err = component.validate()
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].name < ordered[j].name
	})
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].name == ordered[i].name {
			return nil, ErrDuplicateComponent
		}
	}

	size := 4 + 4 + len(domain) + 4
	for _, component := range ordered {
		fieldSize := 4 + len(component.name) + 1 + 8
		if fieldSize > math.MaxInt-len(component.value) || size > math.MaxInt-fieldSize-len(component.value) {
			return nil, fmt.Errorf("%w: encoded record is too large", ErrInvalidDeterministicComponent)
		}
		size += fieldSize + len(component.value)
	}

	record := make([]byte, 0, size)
	record = append(record, "KFD1"...)
	record = appendUint32(record, uint32(len(domain)))
	record = append(record, domain...)
	record = appendUint32(record, uint32(len(ordered)))
	for _, component := range ordered {
		record = appendUint32(record, uint32(len(component.name)))
		record = append(record, component.name...)
		record = append(record, component.kind)
		record = appendUint64(record, uint64(len(component.value)))
		record = append(record, component.value...)
	}

	return record, nil
}

// validateComponentCount reports whether a component count can be encoded by KFD1.
func validateComponentCount(count uint64) error {

	if count > math.MaxUint32 {
		return ErrComponentCountOverflow
	}

	return nil
}

// validDomain reports whether domain follows the KFD1 domain grammar.
func validDomain(domain string) bool {

	if len(domain) == 0 || len(domain) > 128 || domain[0] < 'a' || domain[0] > 'z' {
		return false
	}
	for i := 1; i < len(domain); i++ {
		c := domain[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '.' && c != '/' && c != '-' {
			return false
		}
	}

	return true
}

// validLowerIdentifier reports whether value is a lowercase ASCII identifier of at most maxBytes bytes.
func validLowerIdentifier(value string, maxBytes int) bool {

	if len(value) == 0 || len(value) > maxBytes || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
			return false
		}
	}

	return true
}

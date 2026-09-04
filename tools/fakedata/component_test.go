// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"testing"
)

// TestCanonicalRecordErrors verifies every KFD1 validation category.
func TestCanonicalRecordErrors(t *testing.T) {

	valid := mustStringComponent(t, "value", "ok")
	invalidUTF8 := Component{name: "value", kind: componentKindString, value: []byte{0xff}, valid: true}
	invalidUint64 := Component{name: "value", kind: componentKindUint64, value: []byte{1}, valid: true}
	invalidKind := Component{name: "value", kind: 0xff, valid: true}

	tests := []struct {
		name       string
		domain     string
		components []Component
		want       error
	}{
		{"empty domain", "", []Component{valid}, ErrInvalidDeterministicComponent},
		{"uppercase domain", "Test/v1", []Component{valid}, ErrInvalidDeterministicComponent},
		{"underscore domain", "test_v1", []Component{valid}, ErrInvalidDeterministicComponent},
		{"long domain", "a" + string(bytes.Repeat([]byte{'b'}, 128)), []Component{valid}, ErrInvalidDeterministicComponent},
		{"zero component", "test/v1", []Component{{}}, ErrInvalidDeterministicComponent},
		{"invalid UTF-8", "test/v1", []Component{invalidUTF8}, ErrInvalidDeterministicComponent},
		{"invalid uint64", "test/v1", []Component{invalidUint64}, ErrInvalidDeterministicComponent},
		{"invalid kind", "test/v1", []Component{invalidKind}, ErrInvalidDeterministicComponent},
		{"duplicate", "test/v1", []Component{valid, valid}, ErrDuplicateComponent},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			_, err := canonicalRecord(test.domain, test.components)
			if err != nil {
				if !errors.Is(err, test.want) {
					t.Fatalf("expected %v, got %v", test.want, err)
				}
				return
			}

			t.Fatalf("expected %v, got nil", test.want)

		})

	}

	err := validateComponentCount(uint64(math.MaxUint32) + 1)
	if err != nil {
		if !errors.Is(err, ErrComponentCountOverflow) {
			t.Fatalf("expected %v, got %v", ErrComponentCountOverflow, err)
		}
	}
	if err == nil {
		t.Fatalf("expected %v, got nil", ErrComponentCountOverflow)
	}

}

// TestCanonicalRecordGolden verifies the authoritative KFD1 byte and digest vectors.
func TestCanonicalRecordGolden(t *testing.T) {

	a := mustStringComponent(t, "a", "")
	n := mustUint64Component(t, "n", 42)
	z := mustBytesComponent(t, "z", []byte{0x00, 0xff})

	tests := []struct {
		name       string
		components []Component
		recordHex  string
		digestHex  string
	}{
		{
			name:       "components",
			components: []Component{z, n, a},
			recordHex: "4b46443100000007746573742f7631000000030000000161010000000000000000000000016e020000000000000008" +
				"000000000000002a000000017a03000000000000000200ff",
			digestHex: "90ea4404f554a4c3fc90c77df4a84404bec12892f3a266e6b22bca78b1928976",
		},
		{
			name:       "empty",
			components: nil,
			recordHex:  "4b46443100000007746573742f763100000000",
			digestHex:  "7e330eddde6d018e692dc94b69009d51a703b553ccccca6d2d1d1e02858be404",
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			record, err := canonicalRecord("test/v1", test.components)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if got := hex.EncodeToString(record); got != test.recordHex {
				t.Fatalf("expected record %s, got %s", test.recordHex, got)
			}
			digest := sha256.Sum256(record)
			if got := hex.EncodeToString(digest[:]); got != test.digestHex {
				t.Fatalf("expected digest %s, got %s", test.digestHex, got)
			}

		})

	}

}

// TestCanonicalRecordProperties verifies ordering, type separation, and empty-value semantics.
func TestCanonicalRecordProperties(t *testing.T) {

	a := mustStringComponent(t, "a", "value")
	b := mustUint64Component(t, "b", 7)
	forward, err := canonicalRecord("test/v1", []Component{a, b})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	reverse, err := canonicalRecord("test/v1", []Component{b, a})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !bytes.Equal(forward, reverse) {
		t.Fatalf("expected order-independent records, got %x and %x", forward, reverse)
	}

	stringRecord := mustCanonicalRecord(t, "test/v1", mustStringComponent(t, "value", "\x00\x00\x00\x00\x00\x00\x00\x01"))
	uint64Record := mustCanonicalRecord(t, "test/v1", mustUint64Component(t, "value", 1))
	bytesRecord := mustCanonicalRecord(t, "test/v1",
		mustBytesComponent(t, "value", []byte("\x00\x00\x00\x00\x00\x00\x00\x01")))
	kindsOverlap := bytes.Equal(stringRecord, uint64Record) || bytes.Equal(stringRecord, bytesRecord) ||
		bytes.Equal(uint64Record, bytesRecord)
	if kindsOverlap {
		t.Fatal("expected component kinds to have distinct encodings")
	}

	emptyValue := mustCanonicalRecord(t, "test/v1", mustStringComponent(t, "value", ""))
	absent := mustCanonicalRecord(t, "test/v1")
	if bytes.Equal(emptyValue, absent) {
		t.Fatal("expected an empty value to differ from an absent component")
	}

}

// TestComponentConstructors verifies names, UTF-8, nil bytes, and mutation isolation.
func TestComponentConstructors(t *testing.T) {

	invalidNames := []string{"", "A", "a_b", "0a", "a" + string(bytes.Repeat([]byte{'b'}, 64)), "é"}
	constructors := []struct {
		name      string
		construct func(string) (Component, error)
	}{
		{"bytes", func(name string) (Component, error) { return BytesComponent(name, nil) }},
		{"string", func(name string) (Component, error) { return StringComponent(name, "") }},
		{"uint64", func(name string) (Component, error) { return Uint64Component(name, 0) }},
	}
	for _, constructor := range constructors {

		for _, name := range invalidNames {

			_, err := constructor.construct(name)
			if err != nil {
				if !errors.Is(err, ErrInvalidDeterministicComponent) {
					t.Fatalf("expected invalid component error from %s constructor for %q, got %v",
						constructor.name, name, err)
				}
				continue
			}

			t.Fatalf("expected invalid component error from %s constructor for %q, got nil", constructor.name, name)

		}

	}

	_, err := StringComponent("value", string([]byte{0xff}))
	if err != nil {
		if !errors.Is(err, ErrInvalidDeterministicComponent) {
			t.Fatalf("expected invalid component error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid component error, got nil")
	}

	input := []byte{1, 2, 3}
	component := mustBytesComponent(t, "value", input)
	input[0] = 9
	record := mustCanonicalRecord(t, "test/v1", component)
	if record[len(record)-3] != 1 {
		t.Fatalf("expected copied byte 1, got %d", record[len(record)-3])
	}

	nilRecord := mustCanonicalRecord(t, "test/v1", mustBytesComponent(t, "value", nil))
	emptyRecord := mustCanonicalRecord(t, "test/v1", mustBytesComponent(t, "value", []byte{}))
	if !bytes.Equal(nilRecord, emptyRecord) {
		t.Fatalf("expected nil and empty bytes to match, got %x and %x", nilRecord, emptyRecord)
	}

}

func mustBytesComponent(t *testing.T, name string, value []byte) Component {

	t.Helper()
	component, err := BytesComponent(name, value)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	return component
}

func mustCanonicalRecord(t *testing.T, domain string, components ...Component) []byte {

	t.Helper()
	record, err := canonicalRecord(domain, components)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	return record
}

func mustStringComponent(t *testing.T, name, value string) Component {

	t.Helper()
	component, err := StringComponent(name, value)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	return component
}

func mustUint64Component(t *testing.T, name string, value uint64) Component {

	t.Helper()
	component, err := Uint64Component(name, value)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	return component
}

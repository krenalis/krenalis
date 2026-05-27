// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package base58

import "testing"

func TestGenerate(t *testing.T) {
	id := Generate(12)
	if len(id) != 12 {
		t.Fatalf("expected length 12, got %d", len(id))
	}
	if !IsValid(id) {
		t.Fatalf("expected valid base58 value, got %q", id)
	}
}

func TestGenerateNegative(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = Generate(-1)
}

func TestGenerateZero(t *testing.T) {
	id := Generate(0)
	if id != "" {
		t.Fatalf("expected empty value, got %q", id)
	}
}

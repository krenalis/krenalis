// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package nats defines the configuration keys and identifiers used to configure
// NATS within the core package. These names are also imported by
// core/internal/stream/nats to ensure consistent usage across packages.
package natsopts

import (
	"crypto/ed25519"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nkeys"
)

type Options struct {

	// Connection options.
	Servers  []string
	NKey     ed25519.PrivateKey
	Token    string
	User     string
	Password string

	// Stream options.
	Replicas    int // 0-5
	Storage     StorageType
	Compression StoreCompression
}

type StorageType = jetstream.StorageType

const (
	FileStorage   = jetstream.FileStorage
	MemoryStorage = jetstream.MemoryStorage
)

type StoreCompression = jetstream.StoreCompression

const (
	NoCompression = jetstream.NoCompression
	S2Compression = jetstream.S2Compression
)

type PrefixByte = nkeys.PrefixByte

const PrefixByteUser = nkeys.PrefixByteUser

func DecodeSeed(src []byte) (PrefixByte, []byte, error) {
	return nkeys.DecodeSeed(src)
}

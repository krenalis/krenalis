// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package synthetic provides the two static Source Simulation CSV snapshots.
package synthetic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/fakedata"
)

const maxExactInteger uint64 = 1<<53 - 1

// Config fixes the reproducible Source Simulation inputs for one Core instance.
// Seed and ratio components are limited to the exact integer range of the UI.
type Config struct {
	Namespace                   string               `json:"namespace"`
	Generation                  uint32               `json:"generation"`
	Seed                        uint64               `json:"seed"`
	ReferenceDate               string               `json:"referenceDate"`
	PersonCount                 fakedata.PersonIndex `json:"personCount"`
	PhotoOrigin                 string               `json:"photoOrigin"`
	SourceAID                   string               `json:"sourceAID"`
	SourceAVersion              string               `json:"sourceAVersion"`
	SourceACoverageNumerator    uint64               `json:"sourceACoverageNumerator"`
	SourceACoverageDenominator  uint64               `json:"sourceACoverageDenominator"`
	SourceADuplicateNumerator   uint64               `json:"sourceADuplicateNumerator"`
	SourceADuplicateDenominator uint64               `json:"sourceADuplicateDenominator"`
	SourceBID                   string               `json:"sourceBID"`
	SourceBVersion              string               `json:"sourceBVersion"`
	SourceBCoverageNumerator    uint64               `json:"sourceBCoverageNumerator"`
	SourceBCoverageDenominator  uint64               `json:"sourceBCoverageDenominator"`
	SourceBDuplicateNumerator   uint64               `json:"sourceBDuplicateNumerator"`
	SourceBDuplicateDenominator uint64               `json:"sourceBDuplicateDenominator"`
}

func (c Config) sourceA() fakedata.SourceInstanceConfig {
	return fakedata.SourceInstanceConfig{ID: c.SourceAID, Version: c.SourceAVersion,
		CoverageNumerator: c.SourceACoverageNumerator, CoverageDenominator: c.SourceACoverageDenominator,
		DuplicateNumerator: c.SourceADuplicateNumerator, DuplicateDenominator: c.SourceADuplicateDenominator}
}

func (c Config) sourceB() fakedata.SourceInstanceConfig {
	return fakedata.SourceInstanceConfig{ID: c.SourceBID, Version: c.SourceBVersion,
		CoverageNumerator: c.SourceBCoverageNumerator, CoverageDenominator: c.SourceBCoverageDenominator,
		DuplicateNumerator: c.SourceBDuplicateNumerator, DuplicateDenominator: c.SourceBDuplicateDenominator}
}

// Scenario owns immutable inputs shared by independent CSV readers.
type Scenario struct {
	sourceA  *fakedata.SourceInstance
	sourceB  *fakedata.SourceInstance
	catalog  *fakedata.FaceCatalog
	origin   string
	count    fakedata.PersonIndex
	modified time.Time
}

// New prepares a snapshot once, using the same verified catalog as /photos/.
func New(config Config, catalog *fakedata.FaceCatalog) (*Scenario, error) {
	if catalog == nil {
		return nil, errors.New("Synthetic requires KRENALIS_SYNTHETIC_PHOTOS_DIR")
	}
	if config.Seed > maxExactInteger ||
		config.SourceACoverageNumerator > maxExactInteger || config.SourceACoverageDenominator > maxExactInteger ||
		config.SourceADuplicateNumerator > maxExactInteger || config.SourceADuplicateDenominator > maxExactInteger ||
		config.SourceBCoverageNumerator > maxExactInteger || config.SourceBCoverageDenominator > maxExactInteger ||
		config.SourceBDuplicateNumerator > maxExactInteger || config.SourceBDuplicateDenominator > maxExactInteger {
		return nil, fmt.Errorf("Synthetic seed and source ratios must be at most %d", maxExactInteger)
	}
	if config.PersonCount > fakedata.Layer1MaxPersonIndex || config.SourceAID == config.SourceBID {
		return nil, errors.New("invalid Synthetic population or sources")
	}
	modified, err := time.Parse(time.DateOnly, config.ReferenceDate)
	if err != nil || modified.Format(time.DateOnly) != config.ReferenceDate ||
		modified.After(time.Now().UTC().Add(5*time.Minute)) {
		return nil, errors.New("invalid Synthetic reference date")
	}
	namespace, err := fakedata.NewIdentityNamespace(config.Namespace, config.Generation)
	if err != nil {
		return nil, err
	}
	_, err = fakedata.NewSourceCSVWriter(io.Discard, catalog, config.PhotoOrigin)
	if err != nil {
		return nil, err
	}
	base, err := fakedata.NewWorld(fakedata.WorldConfig{IdentityNamespace: namespace, WorldSeed: config.Seed,
		ReferenceDate: config.ReferenceDate, FaceCatalog: catalog})
	if err != nil {
		return nil, err
	}
	world, err := fakedata.NewSourceWorld(base, []fakedata.CountryShare{
		{Code: "IT", Version: fakedata.MarketDataVersion, Weight: 1},
	})
	if err != nil {
		return nil, err
	}
	a, err := fakedata.NewSourceInstance(world, config.sourceA())
	if err != nil {
		return nil, err
	}
	b, err := fakedata.NewSourceInstance(world, config.sourceB())
	if err != nil {
		return nil, err
	}
	return &Scenario{sourceA: a, sourceB: b, catalog: catalog, origin: config.PhotoOrigin,
		count: config.PersonCount, modified: modified}, nil
}

// Reader opens a bounded-memory stream for a supported snapshot.
func (s *Scenario) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	path, err := Path(name)
	if err != nil {
		return nil, time.Time{}, err
	}
	if ctx == nil || s == nil {
		return nil, time.Time{}, errors.New("invalid Synthetic scenario or context")
	}
	source := s.sourceA
	if path == "customers-b.csv" {
		source = s.sourceB
	}
	r := &csvReader{ctx: ctx, source: source, count: s.count, next: 1}
	r.writer, err = fakedata.NewSourceCSVWriter(&r.buffer, s.catalog, s.origin)
	if err != nil {
		return nil, time.Time{}, err
	}
	err = r.writer.Flush(ctx)
	if err != nil {
		return nil, time.Time{}, err
	}
	return r, s.modified, nil
}

type csvReader struct {
	ctx    context.Context
	source *fakedata.SourceInstance
	writer *fakedata.SourceCSVWriter
	buffer bytes.Buffer
	next   fakedata.PersonIndex
	count  fakedata.PersonIndex
	closed bool
}

func (r *csvReader) Close() error {
	r.closed = true
	r.buffer.Reset()
	return nil
}

func (r *csvReader) Read(p []byte) (int, error) {
	if r.closed {
		return 0, errors.New("Synthetic reader is closed")
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) == 0 {
		return 0, nil
	}
	for r.buffer.Len() == 0 {
		if err := r.ctx.Err(); err != nil {
			return 0, err
		}
		if r.next > r.count {
			return 0, io.EOF
		}
		records, err := r.source.Records(r.next)
		if err != nil {
			return 0, err
		}
		r.next++
		for _, record := range records {
			err = r.writer.Write(r.ctx, record)
			if err != nil {
				return 0, err
			}
		}
		err = r.writer.Flush(r.ctx)
		if err != nil {
			return 0, err
		}
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.buffer.Read(p)
}

// Path normalizes only the two supported CSV names and their leading-slash aliases.
func Path(name string) (string, error) {
	switch name {
	case "customers-a.csv", "/customers-a.csv":
		return "customers-a.csv", nil
	case "customers-b.csv", "/customers-b.csv":
		return "customers-b.csv", nil
	default:
		return "", connectors.InvalidPathErrorf("unsupported Synthetic path %q", name)
	}
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/krenalis/krenalis/tools/fakedata/internal/usadiagnostic"
)

const (
	usaSourceUSDModelBytes  = 206928
	usaSourceUSDModelSHA256 = "d48443e062ba30f684c7bea088a8f184e8e5a971e417f08c596985358e71811f"
)

// usaSourceUSDProvider is a lazy repository-only configuration, with no cache.
// Failed preparation publishes no state; the same provider can retry safely.
type usaSourceUSDProvider struct {
	options usadiagnostic.Options
}

func newUSASourceUSDProvider(options usadiagnostic.Options) (*usaSourceUSDProvider, error) {
	if options.Country != "US" || options.Currency != "USD" || options.Profile != usaSourceUSDProfile ||
		options.ModelDirectory == "" || options.Root == "" || options.Publish ||
		(options.Order != "ascending" && options.Order != "descending") {
		return nil, fmt.Errorf("explicit internal US/USD profile, model directory and traversal are required")
	}
	return &usaSourceUSDProvider{options: options}, nil
}

func (p *usaSourceUSDProvider) prepare(ctx context.Context) (*usaCountryWorld, map[string]any, error) {

	if p == nil || ctx == nil {
		return nil, nil, fmt.Errorf("USA source provider and context are required")
	}
	started := time.Now()
	var initial runtime.MemStats
	runtime.ReadMemStats(&initial)
	resources := map[string]any{"goVersion": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH}
	assets, err := loadUSACountryAssets(ctx, p.options.Root)
	if err != nil {
		return nil, nil, err
	}
	resources["aggregateAssetLoadSeconds"] = time.Since(started).Seconds()

	// Pin the real M1 document before decoding. No model or currency fallback.
	file, err := os.Open(filepath.Join(p.options.ModelDirectory, "assets/socioeconomic.json"))
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !info.Mode().IsRegular() || info.Size() != usaSourceUSDModelBytes {
		return nil, nil, fmt.Errorf("incompatible USA source model size")
	}
	encoded, err := io.ReadAll(io.LimitReader(&internationalReader{ctx: ctx, reader: file}, usaSourceUSDModelBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if len(encoded) != usaSourceUSDModelBytes || fmt.Sprintf("%x", sha256.Sum256(encoded)) != usaSourceUSDModelSHA256 {
		return nil, nil, fmt.Errorf("altered USA source model asset")
	}
	assembled := time.Now()
	data, err := prepareUSACountryData(ctx, assets, encoded)
	if err != nil {
		return nil, nil, err
	}
	resources["completeModelPreparationSeconds"] = time.Since(assembled).Seconds()
	resources["coldLoadSeconds"] = time.Since(started).Seconds()
	assets, encoded = nil, nil
	runtime.GC()
	var prepared runtime.MemStats
	runtime.ReadMemStats(&prepared)
	resources["preparedHeapAllocBytes"] = prepared.HeapAlloc

	constructed := time.Now()
	world, err := newUSACountryWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		return nil, nil, err
	}
	resources["rosterConstructionSeconds"] = time.Since(constructed).Seconds()
	runtime.GC()
	var retained runtime.MemStats
	runtime.ReadMemStats(&retained)
	resources["worldHeapAllocBytes"] = retained.HeapAlloc
	resources["preparationTotalAllocBytes"] = retained.TotalAlloc - initial.TotalAlloc
	resources["preparationMallocs"] = retained.Mallocs - initial.Mallocs

	return world, resources, nil
}

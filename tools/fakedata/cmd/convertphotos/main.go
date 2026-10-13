// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Command convertphotos qualifies a real photo source and the existing Italian
// JSON documents, then writes only new photo and release assets locally.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/krenalis/krenalis/tools/fakedata"
)

func main() {
	local := flag.String("local", "", "complete local fakefacegen snapshot to verify")
	selectionSeed := flag.String("selection-seed", "", "64 lowercase hex digits, nonzero (required with -local)")
	catalog := flag.String("catalog", "", "complete old external v1 catalog; alternative to -local")
	base := flag.String("base", "", "already converted base.json (read only)")
	model := flag.String("socioeconomic", "", "already converted socioeconomic.json (read only)")
	out := flag.String("out", "", "new local staging directory (must not exist)")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	err := run(ctx, *local, *catalog, *selectionSeed, *base, *model, *out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, local, catalogPath, selectionSeed, basePath, modelPath, output string) error {

	if (local == "") == (catalogPath == "") || basePath == "" || modelPath == "" || output == "" {
		return fmt.Errorf("provide one of -local/-catalog, plus -base, -socioeconomic and -out")
	}
	if local != "" && selectionSeed == "" {
		return fmt.Errorf("-selection-seed is required with -local")
	}
	baseFile, err := os.Open(basePath)
	if err != nil {
		return err
	}
	defer baseFile.Close()
	base, err := fakedata.DecodeItalianBaseData(ctx, baseFile)
	if err != nil {
		return err
	}
	modelFile, err := os.Open(modelPath)
	if err != nil {
		return err
	}
	defer modelFile.Close()
	model, err := fakedata.DecodeItalianSocioeconomicData(ctx, modelFile)
	if err != nil {
		return err
	}

	var photos *fakedata.PhotoCatalogData
	if local != "" {
		photos, err = fakedata.ConvertLocalPhotoCatalogData(ctx, local, selectionSeed)
	} else {
		file, openErr := os.Open(catalogPath)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		photos, err = fakedata.ConvertExternalPhotoCatalogData(ctx, file)
	}
	if err != nil {
		return err
	}
	photoJSON, err := photos.CanonicalJSON()
	if err != nil {
		return err
	}
	// Qualify the emitted bytes and the complete combination before writing.
	photos, err = fakedata.DecodePhotoCatalogData(ctx, bytes.NewReader(photoJSON))
	if err != nil {
		return err
	}
	_, err = fakedata.PrepareItalianWorldData(base, model, photos)
	if err != nil {
		return err
	}
	baseDigest, err := base.LogicalDigest()
	if err != nil {
		return err
	}
	modelDigest, err := model.LogicalDigest()
	if err != nil {
		return err
	}
	photoDigest, err := photos.LogicalDigest()
	if err != nil {
		return err
	}
	fingerprint, err := photos.SelectionFingerprint()
	if err != nil {
		return err
	}
	input := fmt.Sprintf(`{
		"schema":"sw-country-release-v1", "release":"it-2026a-json1", "country":"IT",
		"base":{"path":"synthetic-world/countries/IT/it-2026a-json1/base.json","logicalSha256":"%x"},
		"socioeconomic":{"path":"synthetic-world/countries/IT/it-2026a-json1/socioeconomic.json","logicalSha256":"%x"},
		"photos":{"path":"synthetic-world/photos/v1-json1/catalog.json","logicalSha256":"%x"}
	}`, baseDigest, modelDigest, photoDigest)
	release, err := fakedata.DecodeCountryReleaseData(ctx, bytes.NewBufferString(input))
	if err != nil {
		return err
	}
	releaseJSON, err := release.CanonicalJSON()
	if err != nil {
		return err
	}
	releaseDigest, err := release.LogicalDigest()
	if err != nil {
		return err
	}

	err = os.Mkdir(output, 0755)
	if err != nil {
		return err
	}
	for _, asset := range []struct {
		name   string
		data   []byte
		digest [32]byte
	}{{"photos.json", photoJSON, photoDigest}, {"release.json", releaseJSON, releaseDigest}} {
		path := filepath.Join(output, asset.name)
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		_, err = file.Write(asset.data)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		fmt.Printf("%s\nbytes=%d\nsha256=%x\nlogicalSha256=%x\n",
			path, len(asset.data), sha256.Sum256(asset.data), asset.digest)
	}
	fmt.Printf("F_photo=%x\nD_base=%x\nD_socioeconomic=%x\n", fingerprint, baseDigest, modelDigest)

	return nil
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Command convertitalian converts the frozen Italian sources into local v1
// staging assets. It refuses an existing output directory and never publishes.
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
	csvPath := flag.String("csv", "tools/fakedata/data/it-2026a", "directory containing the six original CSV files")
	modelPath := flag.String("model", "D2B2-logical-assets-v0.json", "original canonical socioeconomic asset")
	output := flag.String("out", "", "new local staging directory (must not exist)")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	err := run(ctx, *csvPath, *modelPath, *output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, csvPath, modelPath, output string) error {

	if output == "" {
		return fmt.Errorf("-out must name a new local staging directory")
	}
	base, err := fakedata.ConvertItalianBaseData(ctx, os.DirFS(csvPath))
	if err != nil {
		return err
	}
	file, err := os.Open(modelPath)
	if err != nil {
		return err
	}
	model, err := fakedata.ConvertItalianSocioeconomicData(ctx, file)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	baseJSON, err := base.CanonicalJSON()
	if err != nil {
		return err
	}
	modelJSON, err := model.CanonicalJSON()
	if err != nil {
		return err
	}

	// Qualify the actual emitted bytes through both new typed decoders.
	decodedBase, err := fakedata.DecodeItalianBaseData(ctx, bytes.NewReader(baseJSON))
	if err != nil {
		return err
	}
	decodedModel, err := fakedata.DecodeItalianSocioeconomicData(ctx, bytes.NewReader(modelJSON))
	if err != nil {
		return err
	}
	baseDigest, err := decodedBase.LogicalDigest()
	if err != nil {
		return err
	}
	modelDigest, err := decodedModel.LogicalDigest()
	if err != nil {
		return err
	}
	fingerprints, err := decodedBase.Fingerprints()
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
	}{
		{"base.json", baseJSON, baseDigest},
		{"socioeconomic.json", modelJSON, modelDigest},
	} {
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
	for i, name := range []string{"ages", "first-names", "last-names", "places", "postal-codes", "street-names"} {
		fmt.Printf("%s fingerprint=%x\n", name, fingerprints[i])
	}

	return nil
}

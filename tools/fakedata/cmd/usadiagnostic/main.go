// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// usadiagnostic assembles and exercises USA aggregate assets offline.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	_ "github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/fakedata/internal/usadiagnostic"
)

func main() {

	var options usadiagnostic.Options
	flag.StringVar(&options.Operation, "operation", "run", "assemble, run or source-usd")
	flag.StringVar(&options.Root, "repo", ".", "repository root with byte-locked aggregate assets")
	flag.StringVar(&options.Directory, "directory", "", "internal staging directory")
	flag.StringVar(&options.ModelDirectory, "model-directory",
		"synthetic-world-usa-m1-internal-country-diagnostic-v2-CANDIDATE", "existing complete M1 model directory")
	flag.StringVar(&options.Country, "country", "US", "diagnostic country; only US is admitted")
	flag.StringVar(&options.Currency, "currency", "USD", "diagnostic currency; only USD is admitted")
	flag.StringVar(&options.Profile, "profile", "us-2026a-socioeconomic-v1", "diagnostic socioeconomic profile")
	flag.StringVar(&options.Order, "order", "ascending", "source-usd traversal: ascending or descending")
	flag.IntVar(&options.People, "people", 20000, "ascending prefix size, 20000 through 50000")
	flag.BoolVar(&options.Publish, "publish", false, "unsupported; requests are rejected")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	err := usadiagnostic.Run(ctx, options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}

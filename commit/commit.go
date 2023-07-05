//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

func main() {

	var short bool
	var verbose bool
	flag.BoolVar(&short, "short", false, "pass the '-short' flag to 'go test'")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.Parse()

	start := time.Now()

	// Find modules in this repository.
	var modules []string
	err := filepath.Walk(".", func(path string, _ fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Base(path) == "go.mod" {
			modules = append(modules, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(modules)

	// Check if the command has been executed correctly basing on modules which
	// certainly should be found.
	for _, mod := range []string{".", "chichi-cli"} {
		if !slices.Contains(modules, mod) {
			log.Fatalf("module %q not found, maybe you ran this script incorrectly"+
				" or this script is out-of-date", mod)
		}
	}

	// Get the cwd.
	repo, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Running 'go mod tidy' in every module and regenerating 'go.sum' files")
	for _, module := range modules {
		removeGoSum(repo, module, verbose)
		cmd("go", []string{"mod", "tidy"}, repo, module, verbose)
	}

	fmt.Println("Running 'go fmt' and 'go vet' in every module")
	for _, module := range modules {
		cmd("go", []string{"fmt", "./..."}, repo, module, verbose)
		cmd("go", []string{"vet", "./..."}, repo, module, verbose)
	}

	// Call command(s) on every module.
	for _, module := range modules {
		args := []string{"test", "./..."}
		if short {
			args = append(args, "-short")
		}
		if verbose {
			args = append(args, "-v")
		}
		cmd("go", args, repo, module, verbose)
	}

	// Call command(s) on the workspace.
	cmd("go", []string{"work", "sync"}, repo, ".", true)

	fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
}

func cmd(name string, arg []string, repo, moduleDir string, echo bool) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = filepath.Join(repo, moduleDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if echo {
		logCmd(moduleDir, strings.Join(append([]string{name}, arg...), " "))
	}
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func removeGoSum(repo, module string, verbose bool) {
	if verbose {
		logCmd(module, "rm go.sum")
	}
	err := os.Remove(filepath.Join(repo, module, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}
}

func logCmd(dir, cmd string) {
	fmt.Printf("%-30s%s\n", dir, cmd)
}

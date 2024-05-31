//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const expectedDenoVersion = "1.44.0"

func main() {

	var short bool
	var verbose bool
	var testPackages bool
	flag.BoolVar(&short, "short", false, "pass the '-short' flag to 'go test'")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&testPackages, "pkg", false, "run tests on every single package"+
		" instead of every module (used in conjunction with option '-v', may"+
		" help spotting problems in tests)")
	flag.Parse()

	start := time.Now()

	// Check if the locally installed Deno version is correct.
	checkDenoVersion()

	// Find modules and packages in this repository.
	var modules []string
	var packages []string
	err := filepath.Walk(".", func(path string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() && fi.Name() == "vendor" {
			return filepath.SkipDir
		}
		// Module found.
		if filepath.Base(path) == "go.mod" {
			modules = append(modules, filepath.Dir(path))
		}
		// Package found.
		if filepath.Ext(path) == ".go" {
			dir := filepath.Dir(path)
			if !slices.Contains(packages, dir) {
				packages = append(packages, dir)
			}
		}
		return nil
	})
	if err != nil {
		fatal("cannot read modules and packages in the repository: %s", err)
	}
	slices.Sort(modules)
	slices.Sort(packages)

	// Check if the command has been executed correctly basing on modules which
	// certainly should be found.
	for _, mod := range []string{".", "chichi-cli"} {
		if !slices.Contains(modules, mod) {
			fatal("module %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", mod)
		}
	}

	// Check if the command has been executed correctly basing on packages which
	// certainly should be found.
	for _, pkg := range []string{".", "apis", "chichi-cli"} {
		if !slices.Contains(packages, pkg) {
			fatal("package %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", pkg)
		}
	}

	// Get the cwd.
	repo, err := os.Getwd()
	if err != nil {
		fatal("cannot read the cwd: %s", err)
	}

	fmt.Println("Tidying modules")
	for _, module := range modules {
		removeGoSum(repo, module, verbose)
		cmd("go", []string{"mod", "tidy"}, repo, module, verbose)
	}

	fmt.Println("Formatting modules")
	for _, module := range modules {
		cmd("go", []string{"fmt", "./..."}, repo, module, verbose)
	}

	fmt.Println("Running 'go vet' on every module")
	for _, module := range modules {
		cmd("go", []string{"vet", "./..."}, repo, module, verbose)
	}

	// Test single packages or modules.
	fmt.Println("Running Go tests")
	args := []string{"test", "-count", "1"}
	if short {
		args = append(args, "-short")
	}
	if verbose {
		args = append(args, "-v")
	}
	if testPackages {
		for _, pkg := range packages {
			cmd("go", args, repo, pkg, verbose)
		}
	} else {
		args = append(args, "./...")
		for _, module := range modules {
			cmd("go", args, repo, module, verbose)
		}
	}

	// Sync and vendor the workspace.
	cmd("go", []string{"work", "sync"}, repo, ".", true)
	cmd("go", []string{"work", "vendor"}, repo, ".", true)

	// Run 'npm install' in the 'assets' directory.
	cmd("npm", []string{"install"}, repo, "assets", true)

	// Format the files in the 'assets' directory.
	cmd("npm", []string{"run", "prettier"}, repo, "assets", true)

	// Minify the JavaScript snippet in the 'assets' directory.
	cmd("npm", []string{"run", "minify-snippet"}, repo, "assets", true)

	// Typecheck the Typescript code in the 'assets' directory.
	cmd("npm", []string{"run", "typecheck"}, repo, "assets", true)

	// Make the vendor of assets' 'node_modules' directory.
	cmd("npm", []string{"run", "make-vendor"}, repo, "assets", true)

	// Format, test and build the files in the 'javascript-sdk' directory.
	cmd("deno", []string{"fmt"}, repo, "javascript-sdk", true)
	cmd("deno", []string{"task", "build"}, repo, "javascript-sdk", true)

	fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
}

func checkDenoVersion() {
	fmt.Println("Checking the Deno version")
	cmd := exec.Command("deno", "--version")
	var stdout bytes.Buffer
	cmd.Stderr = os.Stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		fatal("cannot execute the command 'deno --version': %s", err)
	}
	firstLine := strings.Split(stdout.String(), "\n")[0]
	parts := strings.Split(firstLine, " ")
	if len(parts) < 2 {
		fatal("unexpected output from 'deno --version': %q", stdout.String())
	}
	version := parts[1]
	if version != expectedDenoVersion {
		msg := "the version of Deno you have locally does not match the one required by the script 'commit/commit.go':\n" +
			fmt.Sprintf("the expected Deno version was %s, but the installed is %s.\n", expectedDenoVersion, version) +
			"This can happen for two reasons: either the version of Deno you have locally is incorrect (or not up-to-date),\n" +
			"in which case you should upgrade it by running the command:\n" +
			fmt.Sprintf("\n\tdeno upgrade --version %s\n\n", expectedDenoVersion) +
			"Alternatively, if the version you have locally is correct and up-to-date,\n" +
			"you need to update the version specified in the 'commit/commit.go' script."
		fatal(msg)
	}
	fmt.Printf("Locally installed Deno version is correct: %s\n", version)
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
		fatal("command %q failed (%s)", name, err)
	}
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Fatal error: "+msg+"\n", args...)
	os.Exit(1)
}

func removeGoSum(repo, module string, verbose bool) {
	if verbose {
		logCmd(module, "rm go.sum")
	}
	err := os.Remove(filepath.Join(repo, module, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fatal("cannot remove 'go.sum': %s", err)
	}
}

func logCmd(dir, cmd string) {
	fmt.Printf("%-39s %s\n", dir, cmd)
}

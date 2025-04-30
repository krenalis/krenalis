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

// expectedDenoVersion is the expected Deno version.
//
// Keep in sync with the version within ".github/workflows/main.yml".
const expectedDenoVersion = "2.3.1"

func main() {

	cliOptions := parseCli()

	if cliOptions.justTestAdmin {
		runGoTestAdmin(cliOptions.explicit)
		os.Exit(0)
	}

	start := time.Now()

	// Check if the locally installed Deno version is correct.
	checkDenoVersion(cliOptions.explicit)

	// Find modules and packages in the current working directory, then ensure
	// that the script has been launched with the correct working directory.
	modules, packages := findModulesPackages(".")
	for _, mod := range []string{"."} { // just some random modules in the repository (currently we have just one).
		if !slices.Contains(modules, mod) {
			fatal("module %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", mod)
		}
	}
	for _, pkg := range []string{".", "core", "connectors", "warehouses"} { // just some random top-level packages in the repository.
		if !slices.Contains(packages, pkg) {
			fatal("package %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", pkg)
		}
	}

	// Get the current working directory.
	repo, err := os.Getwd()
	if err != nil {
		fatal("cannot read the current working directory: %s", err)
	}

	// Tidy modules.
	if cliOptions.explicit {
		fmt.Println("Tidy modules")
	}
	for _, module := range modules {
		removeGoSum(repo, module, cliOptions.explicit)
		NewCmd("go", "mod", "tidy").InDir(repo, module).Run(cliOptions.explicit)
	}

	// Format modules.
	if cliOptions.explicit {
		fmt.Println("Format modules")
	}
	for _, module := range modules {
		NewCmd("go", "fmt", "./...").InDir(repo, module).Run(cliOptions.explicit)
	}

	// Running 'go vet' on every module.
	if cliOptions.explicit {
		fmt.Println("Running 'go vet' on every module")
	}
	for _, module := range modules {
		NewCmd("go", "vet", "./...").InDir(repo, module).Run(cliOptions.explicit)
	}

	// Run Go tests.
	if runGoTests := !cliOptions.noGoTest; runGoTests {
		if cliOptions.explicit {
			fmt.Println("Run Go tests")
		}
		args := []string{"test", "-count", "1", "-failfast"}
		if cliOptions.short {
			args = append(args, "-short")
		}
		if cliOptions.explicit {
			args = append(args, "-v")
			// It is important to specify a timeout, because otherwise `go test` has
			// a default timeout of 10 minutes (see 'go help testflag'), which is
			// not sufficient to run all the tests inside "/test" in cases when, for
			// example, they are executed in a GitHub Action.
			args = append(args, "--timeout=30m")
		} else {
			// Just to avoid the command running indefinitely without even printing
			// output. 18 minutes should be more than enough time to run the tests
			// locally.
			args = append(args, "-timeout=18m")
		}
		if cliOptions.explicit {
			for _, pkg := range packages {
				NewCmd("go", args...).InDir(repo, pkg).Run(cliOptions.explicit)
			}
		} else {
			args = append(args, "./...")
			for _, module := range modules {
				NewCmd("go", args...).InDir(repo, module).Run(cliOptions.explicit)
			}
		}
	}

	// Update the Go vendor.
	NewCmd("go", "mod", "vendor").InDir(repo).Run(cliOptions.explicit)

	// Run checks and do operations on the admin assets.
	if cliOptions.explicit {
		fmt.Println("Run checks and do operations on the admin assets")
	}
	NewCmd("npm", "install").InDir(repo, "assets").Run(cliOptions.explicit)
	NewCmd("npm", "run", "prettier").InDir(repo, "assets").Run(cliOptions.explicit)
	NewCmd("npm", "run", "minify-snippet").InDir(repo, "assets").Run(cliOptions.explicit)
	NewCmd("npm", "run", "typecheck").InDir(repo, "assets").Run(cliOptions.explicit)
	NewCmd("npm", "run", "make-vendor").InDir(repo, "assets").Run(cliOptions.explicit)

	// Run checks and do operations on the JavaScript SDK.
	if cliOptions.explicit {
		fmt.Println("Run checks and do operations on the JavaScript SDK")
	}
	NewCmd("npm", "install").InDir(repo, "javascript-sdk").Run(cliOptions.explicit)
	NewCmd("deno", "fmt").InDir(repo, "javascript-sdk").Run(cliOptions.explicit)
	NewCmd("deno", "task", "build").InDir(repo, "javascript-sdk").Run(cliOptions.explicit)

	// Run "go generate" within cmd/spec.
	NewCmd("go", "generate", "./...").InDir(repo, "cmd", "spec").Run(cliOptions.explicit)

	if cliOptions.explicit {
		fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
	}
}

type cliOptions struct {
	explicit      bool
	justTestAdmin bool
	noGoTest      bool
	short         bool
}

func parseCli() cliOptions {

	var explicit bool
	var justTestAdmin bool
	var noGoTest bool
	var printHelp bool
	var short bool

	const reducedTestSetWarning = "WARNING: this option reduces the set of tests performed," +
		" so it cannot be used to validate the repository before a commit"

	flag.BoolVar(&explicit, "x", false, "explicit mode, which runs the tests for"+
		" each package separately and prints verbose output; may take a little longer;"+
		" the tests set is unaltered by this option")
	flag.BoolVar(&justTestAdmin, "just-test-admin", false, "just run the go tests on the admin. "+
		reducedTestSetWarning)
	flag.BoolVar(&noGoTest, "no-go-test", false, "do not run 'go test' at all."+
		" Useful when you just want to run vendor generation commands, various asset related commands, etc... "+
		reducedTestSetWarning)
	flag.BoolVar(&short, "short", false, "pass the '-short' flag to 'go test', reducing the tests set. "+reducedTestSetWarning)
	flag.BoolVar(&printHelp, "help", false, "print help message and exit")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of the 'commit' command:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) > 0 {
		fmt.Fprintf(flag.CommandLine.Output(), "Unexpected command line parameters: %s\n", strings.Join(flag.Args(), " "))
		flag.Usage()
		os.Exit(1)
	}

	if printHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Mutually exclusive flags.
	mutualExclusive := func(flag1, flag2 bool, name1, name2 string) {
		if flag1 && flag2 {
			fmt.Fprintf(flag.CommandLine.Output(), "CLI error: flag '%s' cannot be used in conjunction with flag '%s'\n", name1, name2)
			flag.Usage()
			os.Exit(1)
		}
	}
	mutualExclusive(justTestAdmin, short, "-just-test-admin", "-short")
	mutualExclusive(justTestAdmin, noGoTest, "-just-test-admin", "-no-go-test")
	mutualExclusive(noGoTest, short, "-no-go-test", "-short")

	return cliOptions{explicit: explicit, justTestAdmin: justTestAdmin, noGoTest: noGoTest, short: short}
}

func checkDenoVersion(explicit bool) {
	if explicit {
		fmt.Println("Checking the Deno version")
	}
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
		fatal(fmt.Sprintf("it is not possible to run the tests because they require Deno %s, but the installed version is Deno %s.\n"+
			"To proceed with the tests, please update the Deno version:\n"+
			"\n\tdeno upgrade --version %s\n\n"+
			"If your intention is to update the tests to use Deno %s instead, please modify the specified version in the 'commit/commit.go' script.\n",
			expectedDenoVersion, version, expectedDenoVersion, version))
	}
	if explicit {
		fmt.Printf("Locally installed Deno version is correct: %s\n", version)
	}
}

// findModulesPackages finds the Go modules and packages within the given dir.
// Both the modules and packages are sorted in alphabetical order.
// This function skips directories named "vendor".
func findModulesPackages(dir string) (modules, packages []string) {
	err := filepath.Walk(dir, func(path string, fi fs.FileInfo, err error) error {
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
	return modules, packages
}

func logCmd(dir, cmd string) {
	const (
		Reset = "\033[0m"
		Bold  = "\033[1m"
	)
	fmt.Printf("%s%s$ %s%s\n", Bold, dir, cmd, Reset)
}

type Cmd struct {
	Name          string
	Args          []string
	Dir           string
	AdditionalEnv []string
}

func NewCmd(name string, args ...string) *Cmd {
	return &Cmd{Name: name, Args: args}
}

func (cmd *Cmd) WithEnv(name, value string) *Cmd {
	cmd.AdditionalEnv = append(cmd.AdditionalEnv, name+"="+value)
	return cmd
}

func (cmd *Cmd) InDir(elem ...string) *Cmd {
	cmd.Dir = filepath.Join(elem...)
	return cmd
}

func (cmd *Cmd) Run(explicit bool) {
	goCmd := exec.Command(cmd.Name, cmd.Args...)
	var output bytes.Buffer
	if explicit {
		goCmd.Stdout = os.Stdout
		goCmd.Stderr = os.Stderr
	} else {
		goCmd.Stdout = &output
		goCmd.Stderr = &output
	}
	goCmd.Env = append(os.Environ(), cmd.AdditionalEnv...)
	goCmd.Dir = cmd.Dir
	if explicit {
		logCmd(cmd.Dir, strings.Join(append([]string{cmd.Name}, cmd.Args...), " "))
	}
	err := goCmd.Run()
	if err != nil {
		if explicit {
			// Stdout and Stderr have already been printed.
		} else {
			fmt.Print(output.String())
		}
		fatal("command %q failed (%s)", cmd.Name, err)
	}
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Fatal error: "+msg+"\n", args...)
	os.Exit(1)
}

func removeGoSum(repo, module string, explicit bool) {
	if explicit {
		logCmd(filepath.Join(repo, module), "rm go.sum")
	}
	err := os.Remove(filepath.Join(repo, module, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fatal("cannot remove 'go.sum': %s", err)
	}
}

// runGoTestAdmin runs admin tests via go test.
func runGoTestAdmin(explicit bool) {
	start := time.Now()
	args := []string{"test", "-run", "^TestAdmin$", "github.com/meergo/meergo/test", "-count", "1"}
	if explicit {
		args = append(args, "-v")
	}
	NewCmd("go", args...).Run(explicit)
	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		fatal("admin test took too little time (< 2 seconds). There is probably a problem" +
			" with its execution, try running it with the '-x' option or check" +
			" the implementation of the commit command")
	}
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
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

// goTestFlags are the flags passed to every 'go test' invocation.
var goTestFlags = []string{
	"-count",
	"1",
	"-failfast",
	// It is important to specify a timeout, because otherwise 'go test'
	// has a default timeout of 10 minutes (see 'go help testflag'),
	// which may not be sufficient to run all the tests inside "/test".
	"--timeout=2h",
}

func main() {

	cliOptions := parseCli()

	// The KRENALIS_SKIP_SNOWFLAKE_TESTS environment variable is set internally
	// by this script when -no-snowflake-tests is used. It must not be set
	// externally, as that would bypass the intended flow and could cause
	// confusion.
	if _, alreadySet := os.LookupEnv("KRENALIS_SKIP_SNOWFLAKE_TESTS"); alreadySet {
		fatal("the KRENALIS_SKIP_SNOWFLAKE_TESTS environment variable is already set in the environment." +
			" This variable is managed internally by the 'commit' script via the -no-snowflake-tests flag" +
			" and must not be set manually.")
	}

	start := time.Now()

	// Find the packages of the module in the current working directory, then
	// ensure that the script has been launched with the correct working
	// directory. Other modules nested in the repository, such as ignored local
	// ones, are not considered.
	_, err := os.Stat("go.mod")
	if err != nil {
		fatal("module was expected in the current working directory but not found, maybe because you ran this script incorrectly or this script is out-of-date: %s", err)
	}
	packages := findPackages(".")
	for _, pkg := range []string{"core", "connectors", "warehouses"} { // just some random top-level packages in the repository.
		if !slices.Contains(packages, pkg) {
			fatal("package %q was expected in the repository but not found, maybe because you ran this script incorrectly or this script is out-of-date", pkg)
		}
	}

	// Get the current working directory.
	repo, err := os.Getwd()
	if err != nil {
		fatal("cannot read the current working directory: %s", err)
	}

	if cliOptions.justTestAdmin {
		runGoTestAdmin(repo)
		fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
		os.Exit(0)
	}

	// Tidy the module.
	fmt.Println("Tidy the module")
	removeGoSum(repo)
	NewCmd("go", "mod", "tidy").InDir(repo).Run()

	// Go-fix the module.
	fmt.Println("Go-fixing the module")
	// Call the command 3 times, because some fixes are applied in subsequent
	// calls. 3 is a reasonable number to most likely perform all the necessary
	// fixes.
	for range 3 {
		NewCmd("go", "fix", "./...").InDir(repo).Run()
	}

	// Format the module.
	fmt.Println("Format the module")
	NewCmd("go", "fmt", "./...").InDir(repo).Run()

	// Running 'go vet' on the module.
	fmt.Println("Running 'go vet' on the module")
	NewCmd("go", "vet", "./...").InDir(repo).Run()

	// Update the Go vendor.
	NewCmd("go", "mod", "vendor").InDir(repo).Run()

	// Run checks and do operations on the Admin.
	// Keep the commands that regenerate the Admin files in sync with runGoTestAdmin.
	fmt.Println("Run checks and do operations on the Admin")
	NewCmd("npm", "ci").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "prettier").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "minify-snippet").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "typecheck").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "makevendor").InDir(repo, "admin").Run()

	// Validate the Docker Compose files.
	fmt.Println("Validate Docker Compose files")
	NewCmd("docker", "compose", "config", "--quiet").WithEnv("KRENALIS_KMS", "key:test-master-key").InDir(repo).Run()                           // compose.yaml with overriding (default).
	NewCmd("docker", "compose", "-f", "compose.yaml", "config", "--quiet").WithEnv("KRENALIS_KMS", "key:test-master-key").InDir(repo).Run()     // compose.yaml without overriding.
	NewCmd("docker", "compose", "-f", "compose.dev.yaml", "config", "--quiet").WithEnv("KRENALIS_KMS", "key:test-master-key").InDir(repo).Run() // compose.dev.yaml.

	// Run Go tests.
	if runGoTests := !cliOptions.noGoTest; runGoTests {
		fmt.Println("Run Go tests")
		args := append([]string{"test"}, goTestFlags...)
		if cliOptions.short {
			args = append(args, "-short")
		}
		if cliOptions.concurrent {
			// Test all the packages with a single 'go test' command, without -v
			// to minimize noise in the output.
			var pkgPaths []string
			for _, pkg := range packages {
				if cliOptions.noConnectorTests && strings.HasPrefix(pkg, "connectors"+string(os.PathSeparator)) {
					continue // skip this package.
				}
				pkgPaths = append(pkgPaths, "./"+filepath.ToSlash(pkg))
			}
			cmd := NewCmd("go", append(args, pkgPaths...)...).InDir(repo)
			if cliOptions.noSnowflakeTests {
				cmd = cmd.WithEnv("KRENALIS_SKIP_SNOWFLAKE_TESTS", "true")
			}
			cmd.Run()
		} else {
			// Test the packages one at a time; since they run sequentially, the
			// output printed by -v is convenient and informative.
			args = append(args, "-v")
			for _, pkg := range packages {
				if cliOptions.noConnectorTests && strings.HasPrefix(pkg, "connectors"+string(os.PathSeparator)) {
					continue // skip this package.
				}
				cmd := NewCmd("go", args...).InDir(repo, pkg)
				if cliOptions.noSnowflakeTests {
					cmd = cmd.WithEnv("KRENALIS_SKIP_SNOWFLAKE_TESTS", "true")
				}
				cmd.Run()
			}
		}
	}

	fmt.Printf("\nDone! (took ~%v)\n", time.Since(start).Round(time.Second))
}

type cliOptions struct {
	concurrent       bool
	justTestAdmin    bool
	noConnectorTests bool
	noSnowflakeTests bool
	noGoTest         bool
	short            bool
}

func parseCli() cliOptions {

	var concurrent bool
	var justTestAdmin bool
	var noGoTest bool
	var printHelp bool
	var short bool
	var noConnectorTests bool
	var noSnowflakeTests bool

	const reducedTestSetWarning = "WARNING: this option reduces the set of tests performed," +
		" so some parts of the software and/or changes made may not be validated " +
		"when running the script with this option"

	flag.BoolVar(&concurrent, "concurrent", false, "run the Go test packages in parallel,"+
		" with a single 'go test' command. Only the packages that fail print their output. Experimental")
	flag.BoolVar(&justTestAdmin, "just-test-admin", false, "just regenerate the Admin files and run the Go tests"+
		" on the Admin console. "+reducedTestSetWarning)
	flag.BoolVar(&noConnectorTests, "no-connector-tests", false, "do not run 'go test' within the 'connectors' directory. "+reducedTestSetWarning)
	flag.BoolVar(&noSnowflakeTests, "no-snowflake-tests", false, "skip Go tests that need a Snowflake environment. "+reducedTestSetWarning)
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

	// Flags incompatible with '--just-test-admin'.
	mutualExclusive(justTestAdmin, concurrent, "-just-test-admin", "-concurrent")
	mutualExclusive(justTestAdmin, noConnectorTests, "-just-test-admin", "-no-connector-tests")
	mutualExclusive(justTestAdmin, noGoTest, "-just-test-admin", "-no-go-test")
	mutualExclusive(justTestAdmin, short, "-just-test-admin", "-short")
	mutualExclusive(justTestAdmin, noSnowflakeTests, "-just-test-admin", "-no-snowflake-tests")
	// Flags incompatible with '--no-go-test'.
	mutualExclusive(noGoTest, concurrent, "-no-go-test", "-concurrent")
	mutualExclusive(noGoTest, short, "-no-go-test", "-short")
	mutualExclusive(noGoTest, noConnectorTests, "-no-go-test", "-no-connector-tests")
	mutualExclusive(noGoTest, noSnowflakeTests, "-no-go-test", "-no-snowflake-tests")

	return cliOptions{
		concurrent:       concurrent,
		justTestAdmin:    justTestAdmin,
		noConnectorTests: noConnectorTests,
		noSnowflakeTests: noSnowflakeTests,
		noGoTest:         noGoTest,
		short:            short,
	}
}

// findPackages finds the Go packages of the module rooted at the given dir,
// sorted in alphabetical order. This function skips directories named "vendor"
// and the directories of other modules, which contain a "go.mod" file.
func findPackages(dir string) []string {
	var packages []string
	err := filepath.Walk(dir, func(path string, fi fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() && fi.Name() == "vendor" {
			return filepath.SkipDir
		}
		// Other module found.
		if fi.IsDir() && path != dir {
			_, err := os.Stat(filepath.Join(path, "go.mod"))
			if err == nil {
				return filepath.SkipDir
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
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
		fatal("cannot read packages in the repository: %s", err)
	}
	slices.Sort(packages)
	return packages
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

func (cmd *Cmd) Run() {
	goCmd := exec.Command(cmd.Name, cmd.Args...)
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	goCmd.Env = append(os.Environ(), cmd.AdditionalEnv...)
	goCmd.Dir = cmd.Dir
	logCmd(cmd.Dir, strings.Join(append([]string{cmd.Name}, cmd.Args...), " "))
	err := goCmd.Run()
	if err != nil {
		// Stdout and Stderr have already been printed.
		fatal("command %q failed (%s)", cmd.Name, err)
	}
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Fatal error: "+msg+"\n", args...)
	os.Exit(1)
}

func removeGoSum(repo string) {
	logCmd(repo, "rm go.sum")
	err := os.Remove(filepath.Join(repo, "go.sum"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fatal("cannot remove 'go.sum': %s", err)
	}
}

// runGoTestAdmin runs Admin tests via go test, after regenerating the Admin
// files embedded in the Krenalis instance that the tests build.
func runGoTestAdmin(repo string) {
	// Keep in sync with the corresponding commands in the main function.
	NewCmd("npm", "ci").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "minify-snippet").InDir(repo, "admin").Run()
	NewCmd("npm", "run", "makevendor").InDir(repo, "admin").Run()
	start := time.Now()
	args := append([]string{"test", "-v", "-run", "^TestAdmin$", "github.com/krenalis/krenalis/test"}, goTestFlags...)
	NewCmd("go", args...).Run()
	elapsed := time.Since(start)
	if elapsed < 2*time.Second {
		fatal("admin test took too little time (< 2 seconds). There is probably a problem" +
			" with the execution of such tests")
	}
}

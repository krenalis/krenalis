//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/meergo/meergo/json"

	"github.com/evanw/esbuild/pkg/api"
)

// Path to the Shoelace icons within the "node_modules" directory.
const shoelaceIconsPath = "@shoelace-style/shoelace/dist/assets/icons"

// Path to the file containing the list of Shoelace icons used in the
// admin.
const shoelaceIconsListPath = "assets/src/shoelace-icons.txt"

func main() {
	err := makeVendor()
	if err != nil {
		log.Fatal(err)
	}
}

func makeVendor() error {

	root, err := moveToModuleRoot()
	if err != nil {
		return err
	}

	nodeModulesDir := filepath.Join(root, "assets", "node_modules") + string(os.PathSeparator)

	// Create the out directory used by esbuild.
	outDir, err := os.MkdirTemp("", "meergo-admin-make-vendor-*")
	if err != nil {
		panic(err)
	}
	defer func() {
		err := os.RemoveAll(outDir)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: could not remove the temporary directory %q: %s", outDir, err)
		}
	}()

	var resolve = newResolveFile()

	plugin := api.Plugin{
		Name: "resolver",
		Setup: func(build api.PluginBuild) {
			build.OnResolve(api.OnResolveOptions{Filter: `^.*$`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if args.Kind == api.ResolveEntryPoint {
						return api.OnResolveResult{}, nil
					}
					if strings.HasPrefix(args.Path, "data:") {
						return api.OnResolveResult{}, nil
					}
					var key string
					if dir, ok := strings.CutPrefix(args.ResolveDir, nodeModulesDir); ok {
						key = pathKey(dir, args.Path)
					} else if isPackagePath(args.Path) {
						key = path.Clean(args.Path)
					}
					if key != "" {
						if resolve.Contains(key) {
							return api.OnResolveResult{}, nil
						}
					}
					if key == "" {
						return api.OnResolveResult{}, nil
					}
					resolve.AddImport(key, "", false)
					result := build.Resolve(args.Path, api.ResolveOptions{
						Importer:   args.Importer,
						Namespace:  args.Namespace,
						Kind:       args.Kind,
						ResolveDir: args.ResolveDir,
					})
					if len(result.Errors) > 0 {
						return api.OnResolveResult{Errors: result.Errors}, nil
					}
					value, ok := strings.CutPrefix(result.Path, nodeModulesDir)
					if !ok {
						return api.OnResolveResult{}, fmt.Errorf("esbuild has resolved %s to a path that is not in 'node_packages' directory: %s", key, result.Path)
					}
					value = filepath.ToSlash(value)
					resolve.AddImport(key, value, result.SideEffects)
					res := api.OnResolveResult{
						Path:        result.Path,
						External:    result.External,
						Namespace:   result.Namespace,
						SideEffects: api.SideEffectsFalse,
					}
					if result.SideEffects {
						res.SideEffects = api.SideEffectsTrue
					}
					return res, nil
				})
		},
	}

	// Run esbuild for the admin.
	entryPoint := filepath.Join(root, "assets", "src", "index.jsx")
	err = build(outDir, entryPoint, plugin)
	if err != nil {
		return err
	}

	// Run esbuild for Monaco workers.
	tsWorker := filepath.Join(root, "assets", "node_modules", "monaco-editor", "esm", "vs", "language", "typescript", "ts.worker.js")
	err = build(outDir, tsWorker, plugin)
	if err != nil {
		return err
	}
	editorWorker := filepath.Join(root, "assets", "node_modules", "monaco-editor", "esm", "vs", "editor", "editor.worker.js")
	err = build(outDir, editorWorker, plugin)
	if err != nil {
		return err
	}

	// Copy the resolved files from the "node_modules" directory to "node_modules_vendor".
	paths := resolve.ResolvedPaths()
	err = os.RemoveAll("./assets/node_modules_vendor")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cannot remove the current 'assets/node_modules_vendor' directory: %s", err)
	}
	packages := map[string]struct{}{}
	for _, name := range paths {
		src := filepath.Join("assets/node_modules", name)
		dst := filepath.Join("assets/node_modules_vendor", name)
		err := os.MkdirAll(filepath.Dir(dst), 0755)
		if err != nil {
			return err
		}
		err = copyFile(dst, src)
		if err != nil {
			return err
		}
		i := 0
		if strings.HasPrefix(name, "@") {
			i = strings.Index(name, "/") + 1
		}
		for {
			if p := strings.Index(name[i:], "/"); p != -1 {
				i += p
				packages[name[:i]] = struct{}{}
			}
			if p := strings.Index(name[i:], "/node_modules/"); p != -1 {
				i += p + len("/node_modules/")
				continue
			}
			break
		}
	}

	// Copy the Monaco workers.
	err = copyFile(filepath.Clean("assets/node_modules_vendor/monaco-editor/esm/vs/language/typescript/ts.worker.js"), tsWorker)
	if err != nil {
		return err
	}
	err = copyFile(filepath.Clean("assets/node_modules_vendor/monaco-editor/esm/vs/editor/editor.worker.js"), editorWorker)
	if err != nil {
		return err
	}

	// Copy the Shoelace icons.
	shoelaceIcons, err := usedShoelaceIconFiles(shoelaceIconsListPath)
	if err != nil {
		return fmt.Errorf("cannot find Shoelace icons: %s", err)
	}
	shoelaceIconsSrc := filepath.Join("assets/node_modules", shoelaceIconsPath)
	shoelaceIconsDst := filepath.Join("assets/node_modules_vendor", shoelaceIconsPath)
	err = os.MkdirAll(shoelaceIconsDst, 0755)
	if err != nil {
		return fmt.Errorf("cannot create directory %q: %s", shoelaceIconsDst, err)
	}
	for _, icon := range shoelaceIcons {
		err = copyFile(filepath.Join(shoelaceIconsDst, icon), filepath.Join(shoelaceIconsSrc, icon))
		if err != nil {
			return fmt.Errorf("cannot copy Shoelace icon file %q: %s", icon, err)
		}
	}

	// Copy 'package.json' and 'LICENSE' files.
	for dir := range packages {
		src := filepath.Join("assets/node_modules", dir, "package.json")
		dst := filepath.Join("assets/node_modules_vendor", dir, "package.json")
		err = copyPackageFile(dst, src)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		for _, license := range []string{"LICENSE", "license", "License"} {
			src = filepath.Join("assets/node_modules", dir, license)
			dst = filepath.Join("assets/node_modules_vendor", dir, strings.ToUpper(license))
			err = copyFile(dst, src)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}

	// Save the 'resolve.json' file.
	b, _ := resolve.MarshalJSON()
	err = os.WriteFile("assets/node_modules_vendor/resolve.json", b, 0644)
	if err != nil {
		return err
	}

	return nil
}

type resolveFile struct {
	imports     map[string]string
	sideEffects map[string]bool
	mu          sync.Mutex
}

func newResolveFile() *resolveFile {
	return &resolveFile{
		imports:     map[string]string{},
		sideEffects: map[string]bool{},
	}
}

func (f *resolveFile) AddImport(path, value string, sideEffects bool) {
	f.mu.Lock()
	f.imports[path] = value
	f.sideEffects[path] = sideEffects
	f.mu.Unlock()
}

func (f *resolveFile) Contains(path string) bool {
	f.mu.Lock()
	_, ok := f.imports[path]
	f.mu.Unlock()
	return ok
}

func (f *resolveFile) ResolvedPaths() []string {
	return slices.Sorted(maps.Values(f.imports))
}

func (f *resolveFile) MarshalJSON() ([]byte, error) {
	f.mu.Lock()
	var b json.Buffer
	b.WriteString("{")
	paths := slices.Sorted(maps.Keys(f.imports))
	for i, name := range paths {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("\n\t")
		_ = b.Encode(name)
		b.WriteString(": ")
		value := f.imports[name]
		if f.sideEffects[name] {
			value += ";true"
		} else {
			value += ";false"
		}
		_ = b.Encode(value)
	}
	b.WriteString("\n}")
	f.mu.Unlock()
	return b.Bytes(), nil
}

// build builds the assets at the provided entry point and writes them into the
// outDir directory.
func build(outDir, entryPoint string, plugin api.Plugin) error {
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{entryPoint},
		Format:            api.FormatESModule,
		JSX:               api.JSXAutomatic,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            outDir,
		Plugins:           []api.Plugin{plugin},
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Loader: map[string]api.Loader{
			".ttf": api.LoaderFile,
		},
		Write: true,
	})
	if result.Errors != nil {
		msg := "cannot generate admin assets when making vendor:"
		for _, err := range result.Errors {
			if len(result.Errors) == 1 {
				msg += " "
			} else {
				msg += "\n  - "
			}
			msg += err.Text
			if loc := err.Location; loc != nil {
				msg += fmt.Sprintf(" at %s %d:%d", loc.File, loc.Line, loc.Column)
			}
		}
		return errors.New(msg)
	}
	if result.Warnings != nil {
		msg := "cannot generate admin assets when making vendor:"
		for _, err := range result.Warnings {
			if len(result.Warnings) == 1 {
				msg += " "
			} else {
				msg += "\n  - "
			}
			msg += err.Text
			if loc := err.Location; loc != nil {
				msg += fmt.Sprintf(" at %s %d:%d", loc.File, loc.Line, loc.Column)
			}
		}
		return errors.New(msg)
	}
	return nil
}

func pathKey(dir, name string) string {
	dir = filepath.ToSlash(dir)
	if isPackagePath(name) {
		i := strings.LastIndex(dir, "/node_modules/")
		if i != -1 {
			dir = dir[:i]
		}
	}
	return path.Join(dir, name)
}

func isPackagePath(name string) bool {
	return name != "." && name != ".." && !strings.HasPrefix(name, "./") && !strings.HasPrefix(name, "../") && !strings.HasPrefix(name, "/")
}

func copyFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

type PackageFile struct {
	Name        any `json:"name,omitempty"`
	Version     any `json:"version,omitempty"`
	Author      any `json:"author,omitempty"`
	License     any `json:"license,omitempty"`
	Type        any `json:"type,omitempty"`
	Main        any `json:"main,omitempty"`
	Browser     any `json:"browser,omitempty"`
	Module      any `json:"module,omitempty"`
	Imports     any `json:"imports,omitempty"`
	Exports     any `json:"exports,omitempty"`
	SideEffects any `json:"sideEffects,omitempty"`
}

func copyPackageFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	file := PackageFile{}
	err = json.Unmarshal(data, &file)
	if err != nil {
		return err
	}
	var b json.Buffer
	err = b.EncodeIndent(file, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b.Bytes(), 0644)
}

func moveToModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		st, err := os.Stat("go.mod")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if !errors.Is(err, os.ErrNotExist) && st.Mode().IsRegular() {
			return cwd, nil
		}
		err = os.Chdir("..")
		if err != nil {
			return "", err
		}
		parent, err := os.Getwd()
		if err != nil {
			return "", err
		}
		if parent == cwd {
			return "", errors.New("go.mod file not found in current directory or any parent directory")
		}
		cwd = parent
	}
}

// usedShoelaceIconFiles returns the Shoelace icon files listed in the
// file at the specified path.
func usedShoelaceIconFiles(path string) ([]string, error) {
	icons := []string{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		iconFile := scanner.Text()
		icons = append(icons, iconFile)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return icons, nil
}

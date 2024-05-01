//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

// This program builds and compresses the assets, storing them in a directory
// named 'chichi-assets' within the current working directory. If a directory
// with the same name already exists, it will be deleted.
package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/evanw/esbuild/pkg/api"
)

//go:embed package.json public/index.html all:src tsconfig.json all:vendor vendor/resolve.json
var assetsFS embed.FS

func main() {
	err := buildAssets()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(" ✔ The asset files have been generated.")
}

// buildAssets builds and the assets.
func buildAssets() error {

	// Create the directory used to build the assets.
	buildDir, err := os.MkdirTemp("", "chichi-assets-build")
	if err != nil {
		return fmt.Errorf("cannot create a temporary directory: %s", err)
	}
	defer func() {
		err = os.Chdir("..")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: cannot chdir to directory %q: %s\n", filepath.Join(buildDir, ".."), err)
		}
		if err = os.RemoveAll(buildDir); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "warning: cannot remove temporary directory %q: %s\n", buildDir, err)
		}
	}()

	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %s", err)
	}

	err = os.Chdir(buildDir)
	if err != nil {
		return fmt.Errorf("cannot chdir to build directory: %s", err)
	}
	buildDir, err = os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory: %s", err)
	}
	buildDir += string(os.PathSeparator)

	uiDir := buildDir + "ui" + string(os.PathSeparator)
	outDir := buildDir + "out" + string(os.PathSeparator)
	dstDir := buildDir + "dst" + string(os.PathSeparator)

	for _, dir := range []string{outDir, dstDir} {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return fmt.Errorf("cannot create directory %q: %s", dir, err)
		}
	}

	// Copy the UI's assets into the uiDir directory.
	err = copyFS(uiDir, assetsFS)
	if err != nil {
		return fmt.Errorf("cannot copy assets: %s", err)
	}

	// Read the 'resolve.json' file.
	resolve, err := readResolveFile()
	if err != nil {
		return fmt.Errorf("cannot read the resolve file: %s", err)
	}

	// Bundle the UI's assets.
	err = build(outDir, uiDir, resolve)
	if err != nil {
		return err
	}

	// Verify that all assets are been generated.
	for _, file := range []string{"index.js", "index.js.map", "index.css", "index.css.map"} {
		st, err := os.Stat(outDir + file)
		if err != nil {
			return fmt.Errorf("cannot stat file %q: %s", outDir+file, err)
		}
		if st.Size() == 0 {
			return fmt.Errorf("bundled file %q is empty", file)
		}
	}

	// Copy the "index.html" file.
	data, _ := assetsFS.ReadFile("public/index.html")
	err = os.WriteFile(outDir+"index.html", data, 0666)
	if err != nil {
		return err
	}

	// Compress the UI's assets.
	var in, out *os.File
	var bw *brotli.Writer
	defer func() {
		if in != nil {
			_ = in.Close()
		}
		if bw != nil {
			_ = bw.Close()
		}
		if out != nil {
			_ = out.Close()
		}
	}()
	for _, name := range []string{"index.html", "index.js", "index.js.map", "index.css", "index.css.map"} {

		srcPath := outDir + name
		dstPath := dstDir + name + ".br"

		// Compress the file.
		in, err = os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("cannot open file %q: %s", srcPath, err)
		}

		out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("cannot create file %q: %s", srcPath+".br", err)
		}

		bw := brotli.NewWriter(out)

		_, err = io.Copy(bw, in)
		if err != nil {
			return fmt.Errorf("cannot compress file %q as %q: %s", srcPath, dstPath, err)
		}

		_ = in.Close()
		in = nil

		err = bw.Close()
		if err != nil {
			return fmt.Errorf("cannot compress file %q as %q: %s", srcPath, dstPath, err)
		}
		bw = nil

		err = out.Close()
		if err != nil {
			return fmt.Errorf("cannot close file %q: %s", dstPath, err)
		}
		out = nil

		// Verify the compressed file.
		in, err = os.Open(dstPath)
		if err != nil {
			return fmt.Errorf("cannot open file %q: %s", dstPath+".br", err)
		}
		br := brotli.NewReader(in)
		_, err = io.Copy(io.Discard, br)
		if err != nil {
			return fmt.Errorf("cannot read file %q: %s", dstPath, err)
		}
		_ = in.Close()
		in = nil

	}

	// Copy the assets to the destination.
	err = os.RemoveAll(filepath.Join(root, "chichi-assets"))
	if err != nil {
		return err
	}
	err = copyFS(filepath.Join(root, "chichi-assets"), os.DirFS(dstDir))
	if err != nil {
		return err
	}

	return nil
}

// build builds the assets located in the assetsDir directory and saves them
// into the outDir directory. If resolve is not nil, it will be used to resolve
// the paths.
func build(outDir, assetsDir string, resolve map[string]string) error {

	var err error
	assetsDir, err = filepath.Abs(assetsDir)
	if err != nil {
		return err
	}

	entryPoint := filepath.Join(assetsDir, "src", "index.tsx")
	vendorDir := filepath.Join(assetsDir, "vendor") + string(os.PathSeparator)

	var plugins []api.Plugin
	if resolve != nil {
		plugin := api.Plugin{
			Name: "resolve_from_vendor",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `.*`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						if args.Kind == api.ResolveEntryPoint {
							return api.OnResolveResult{}, nil
						}
						var key string
						if dir, ok := strings.CutPrefix(args.ResolveDir, vendorDir); ok {
							key = pathKey(dir, args.Path)
						} else if isPackagePath(args.Path) {
							key = path.Clean(args.Path)
						} else {
							return api.OnResolveResult{}, nil
						}
						value, ok := resolve[key]
						if !ok {
							return api.OnResolveResult{}, fmt.Errorf("vendor does not contain the key %q (imported as %q from %q)", key, args.Path, args.ResolveDir)
						}
						var sideEffect api.SideEffects
						if value, ok = strings.CutSuffix(value, ";true"); ok {
							sideEffect = api.SideEffectsTrue
						} else {
							value = strings.TrimSuffix(value, ";false")
							sideEffect = api.SideEffectsFalse
						}
						value = filepath.ToSlash(value)
						res := api.OnResolveResult{
							Path:        filepath.Join(vendorDir, value),
							Namespace:   "file",
							SideEffects: sideEffect,
						}
						return res, nil
					})
			},
		}
		plugins = []api.Plugin{plugin}
	}

	// Bundle the assets.
	result := api.Build(api.BuildOptions{
		Bundle:            true,
		EntryPoints:       []string{entryPoint},
		Format:            api.FormatESModule,
		JSX:               api.JSXAutomatic,
		LegalComments:     api.LegalCommentsNone,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		MinifyWhitespace:  true,
		Outdir:            outDir,
		Plugins:           plugins,
		Sourcemap:         api.SourceMapLinked,
		Target:            api.ES2018,
		TreeShaking:       api.TreeShakingTrue,
		Write:             true,
	})
	if result.Errors != nil {
		var msg string
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

	return nil
}

// readResolveFile reads the 'resolve.json' file.
func readResolveFile() (map[string]string, error) {
	data, _ := assetsFS.ReadFile("vendor/resolve.json")
	resolve := map[string]string{}
	err := json.Unmarshal(data, &resolve)
	if err != nil {
		return nil, err
	}
	return resolve, nil
}

// pathKey returns the key to use in the resolve.json file, relative to the
// given name when imported from the directory dir.
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

// isPackagePath reports whether name is a package path.
func isPackagePath(name string) bool {
	return name != "." && name != ".." && !strings.HasPrefix(name, "./") &&
		!strings.HasPrefix(name, "../") && !strings.HasPrefix(name, "/")
}

// CopyFS copies the provided file system into the directory dir, creating it
// if not exist.
func copyFS(dir string, fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0644)
	})
}

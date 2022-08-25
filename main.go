//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

func main() {

	// Configure the logger.
	logFile, err := os.OpenFile("error.log", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)

	// Read the configuration file.
	settings, err := parseINIFile()
	if err != nil {
		log.Printf("[error] cannot read configuration file: %s", err)
		return
	}
	_ = settings

	// Run the server.
	http.HandleFunc("/admin/src/", HandleESBuild)
	http.Handle("/", http.FileServer(http.Dir("./")))
	err = http.ListenAndServeTLS(":9090", "cert.pem", "key.pem", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func HandleESBuild(w http.ResponseWriter, r *http.Request) {
	file, err := filepath.Abs("admin/src/index.js")
	if err != nil {
		panic(err)
	}
	result := api.Build(api.BuildOptions{
		Bundle:      true,
		EntryPoints: []string{file},
		// Format:      api.FormatESModule,
		Format:  api.FormatESModule,
		JSXMode: api.JSXModeAutomatic,
		Loader: map[string]api.Loader{
			".js": api.LoaderJSX,
		},
		// MinifyIdentifiers: true,
		// MinifySyntax:      true,
		// MinifyWhitespace:  true,
		Target: api.ES2018,
		// TreeShaking: api.TreeShakingTrue,
		Write:  false,
		Outdir: "out",
	})
	if result.Errors != nil || result.Warnings != nil {
		printMessages := func(messages []api.Message) {
			for _, msg := range messages {
				log.Printf(" -> %v", msg)
			}
			log.Fatal("errors/warnings when executing esbuild, cannot proceed")
		}
		if result.Errors != nil {
			printMessages(result.Errors)
		}
		printMessages(result.Warnings)
	}
	base := path.Base(r.URL.Path)
	for _, out := range result.OutputFiles {
		if strings.HasSuffix(out.Path, base) {
			w.Header().Add("Content-Type", "text/javascript")
			w.Write(out.Contents)
			return
		}
	}
	http.NotFound(w, r)
}

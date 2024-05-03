//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package cmd

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/open2b/chichi/apis"
	"github.com/open2b/chichi/telemetry"
)

type Settings struct {
	Main struct {
		Host  string
		HTTPS bool
	}
	UI struct {
		SessionKey string `yaml:"sessionKey"`
	}
	ESBuild struct {
		PrintWarningsOnStderr bool `yaml:"printWarningsOnStderr"`
	}
	PostgreSQL apis.PostgreSQLConfig `yaml:"postgreSQL"`
	SMTP       struct {
		Host string
		Port int
		User string
		Pass string
	}
	Transformer struct {
		Lambda LambdaConfig
		Local  LocalConfig
	}
	Telemetry struct {
		Enable bool
	}
}

type LambdaConfig struct {
	AccessKeyID     string `yaml:"accessKeyID"`
	SecretAccessKey string `yaml:"secretAccessKey"`
	Region          string
	Role            string
	Node            struct {
		Runtime string
		Layer   string
	}
	Python struct {
		Runtime string
		Layer   string
	}
}

type LocalConfig struct {
	NodeExecutable   string `yaml:"nodeExecutable"`
	PythonExecutable string `yaml:"pythonExecutable"`
	FunctionsDir     string `yaml:"functionsDir"`
}

// Run runs the server.
// Cancel ctx to terminate the execution. If ctx is cancelled, Run does not
// return any error.
func Run(ctx context.Context, settings *Settings, assetsFS fs.FS) error {

	if settings.Telemetry.Enable {
		err := telemetry.Init(ctx)
		if err != nil {
			return err
		}
	}

	config := apis.Config{
		PostgreSQL: settings.PostgreSQL,
		SMTP:       settings.SMTP,
	}

	// Choose the transformer setting.
	if settings.Transformer.Lambda.Node.Runtime != "" || settings.Transformer.Lambda.Python.Runtime != "" {
		config.Transformer = apis.LambdaConfig(settings.Transformer.Lambda)
	}
	if settings.Transformer.Local.NodeExecutable != "" || settings.Transformer.Local.PythonExecutable != "" {
		if config.Transformer != nil {
			return errors.New("cannot specify both the Lambda and the local transformer in the configuration (hint: check your 'config.yaml' file)")
		}
		config.Transformer = apis.LocalConfig(settings.Transformer.Local)
	}

	// Decode the UI session key.
	if settings.UI.SessionKey == "" {
		return errors.New("ui session key is missing from the configuration file")
	}
	if padding := len(settings.UI.SessionKey) % 4; padding > 0 {
		settings.UI.SessionKey += strings.Repeat("=", 4-padding)
	}
	sessionKey, err := base64.StdEncoding.DecodeString(settings.UI.SessionKey)
	if err != nil {
		return errors.New("UI session key in the configuration file is not in Base64 format")
	}
	if len(sessionKey) != 64 {
		return fmt.Errorf("UI session key in the configuration file is not 64 bytes long, but %d", len(sessionKey))
	}

	apis, err := apis.New(&config)
	if err != nil {
		return err
	}
	defer apis.Close()

	apisServer := newAPIsServer(apis, sessionKey)

	assets, err := newAssets(assetsFS)
	if err != nil {
		return err
	}
	defer assets.Close()

	addr := settings.Main.Host
	if addr == "" {
		addr = "127.0.0.1:9090"
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Handle panics.
		defer func() {
			if r := recover(); r != nil {
				panicsFilename, err := filepath.Abs("panics.log")
				if err != nil {
					slog.Error("cannot get absolute filepath of 'panics.log': %s", err)
					return
				}
				slog.Error("a panic occurred, Chichi will exit with status code 1. See the file 'panics.log' for the panic details", "panic reason", r, "panics.log filename", panicsFilename)
				f, err := os.OpenFile(panicsFilename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
				if err != nil {
					slog.Error("cannot open panic file", "err", err)
					return
				}
				defer f.Close()
				_, err = fmt.Fprintf(f, "\n----- %s -----\nPanic reason: %v\nStack trace:\n%s",
					time.Now().Format("2006-01-02 15:04:05.000"), r, debug.Stack())
				if err != nil {
					slog.Error("cannot write on panic file", "err", err)
					return
				}
				os.Exit(1)
			}
		}()

		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/"):
			apis.ServeEvents(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/"):
			apisServer.ServeHTTP(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/webhook/"):
			apis.ServeWebhook(w, r)
			return
		case strings.HasPrefix(r.URL.Path, "/ui/") || strings.HasPrefix(r.URL.Path, "/javascript-sdk/"):
			assets.ServeHTTP(w, r)
			return
		default:
			http.NotFound(w, r)
			return
		}

	})

	httpServer := http.Server{
		Addr:    addr,
		Handler: handler,
	}
	var certPem, keyPem string
	if settings.Main.HTTPS {
		certPem, err = filepath.Abs("cert.pem")
		if err != nil {
			return err
		}
		keyPem, err = filepath.Abs("key.pem")
		if err != nil {
			return err
		}
	}

	exited := make(chan error)
	go func() {
		if settings.Main.HTTPS {
			exited <- httpServer.ListenAndServeTLS(certPem, keyPem)
		} else {
			exited <- httpServer.ListenAndServe()
		}
	}()

	select {
	case <-ctx.Done():
		err = httpServer.Shutdown(context.Background())
		if err != nil {
			return err
		}
		err = <-exited
	case err = <-exited:
	}
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// acceptsGzipCompression reports whether the HTTP request accepts a gzip
// compressed response.
func acceptsGzipCompression(req *http.Request) bool {
	for _, encodings := range req.Header["Accept-Encoding"] {
		for _, enc := range strings.Split(encodings, ",") {
			if e := strings.TrimSpace(enc); e == "gzip" || e == "x-gzip" {
				return true
			}
		}
	}
	return false
}

// fileServer is an HTTP handler that handles file with gzip compression.
type fileServer string

func (dir fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if acceptsGzipCompression(r) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		w = &gzipResponseWriter{w, gz}
	}
	http.FileServer(http.Dir(dir)).ServeHTTP(w, r)
}

// gzipResponseWriter is a wrapper for http.ResponseWriter that writes
// compressed data.
type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}

func (rw *gzipResponseWriter) Write(b []byte) (int, error) {
	return rw.writer.Write(b)
}

// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package testconnector provides functions for testing connectors.
package testconnector

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/connections"
	"github.com/krenalis/krenalis/core/internal/connections/httpclient"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/transformers/mappings"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// CaptureRequestContextKey is the context key used to store the original
// *http.Request. The value must always be a non-nil *http.Request, and the
// request body must be properly closed after use.
const CaptureRequestContextKey = httpclient.CaptureRequestContextKey

// DecodeNDJSON reads an NDJSON stream from r encoded with enc. It returns a
// slice of normalized json.Value, or an error if the input is not valid NDJSON.
//
// If the input is empty, it returns an empty slice and a nil error.
func DecodeNDJSON(r io.Reader, enc connectors.ContentEncoding) ([]json.Value, error) {
	if enc == connectors.Gzip {
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		defer gzr.Close()
		r = gzr
	}
	values := make([]json.Value, 0)
	dec := json.NewDecoder(r)
	for {
		v, err := dec.ReadValue()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if v.Kind() != json.Object {
			return nil, fmt.Errorf("expected a JSON Object, got a JSON %s", v.Kind())
		}
		values = append(values, bytes.Clone(v))
	}
	return values, nil
}

// NewApplication returns an instance of the application connector with the
// specified code for testing purposes. Settings are the connector settings
// passed to the connector instance.
//
// It panics if no application connector with the specified code has been
// registered.
func NewApplication[T any](code string, settings any) (T, error) {
	registeredApplications := connectors.RegisteredApplication(code)
	connector := &state.Connector{
		Code:           code,
		EndpointGroups: registeredApplications.EndpointGroups,
	}
	s, err := json.Marshal(settings)
	if err != nil {
		var t T
		return t, fmt.Errorf("cannot marshal settings: %s", err)
	}
	httpClient := httpclient.New(nil, http.DefaultTransport).ConnectorClient(connector, "", "")
	app, err := registeredApplications.New(&connectors.ApplicationEnv{
		Settings:    s,
		SetSettings: func(ctx context.Context, v json.Value) error { return nil },
		HTTPClient:  httpClient,
	})
	return app.(T), err
}

// ReceivedEvent wraps a map[string]any and returns a value that implements the
// connectors.ReceivedEvent interface.
//
// The provided event must conform to the event schema (Schema), otherwise
// calling methods on the returned value may cause a panic.
func ReceivedEvent(event map[string]any) connectors.ReceivedEvent {
	return connections.ReceivedEvent(event)
}

// TransformEvent transforms an event with a mapping and returned the
// transformed properties. mapping is the mapping, schema is the event type
// schema, and properties are the properties to transform. If mapping is nil, it
// maps each property in the schema with its prefilled value if any.
func TransformEvent(schema types.Type, event map[string]any, mapping map[string]string) (map[string]any, error) {
	if mapping == nil {
		mapping = map[string]string{}
		for _, p := range schema.Properties().All() {
			if p.Prefilled != "" {
				mapping[p.Name] = p.Prefilled
			}
		}
	}
	m, err := mappings.New(mapping, schemas.Event, schema, false, nil)
	if err != nil {
		return nil, err
	}
	return m.Transform(event, mappings.Create)
}

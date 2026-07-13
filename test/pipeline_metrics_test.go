// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/test/krenalistester"
)

// TestPipelineMetricsHTTPContract verifies the public JSON shape and routing of
// pipeline metrics endpoints.
func TestPipelineMetricsHTTPContract(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	t.Run("workspace-scoped request requires grouping", func(t *testing.T) {
		err := k.TryCall("GET", "/v1/pipelines/metrics/minutes/1", nil, nil, nil)
		assertPipelineMetricsStatus(t, err, http.StatusBadRequest)
	})

	t.Run("organization-scoped request requires grouping", func(t *testing.T) {
		err := k.TryCall("GET", "/v1/pipelines/metrics/minutes/1", http.Header{"Krenalis-Workspace": nil}, nil, nil)
		assertPipelineMetricsStatus(t, err, http.StatusBadRequest)
	})

	t.Run("workspace-scoped connection group shape", func(t *testing.T) {
		connection := k.CreateDummy("Dummy source", krenalistester.Source)

		var res pipelineMetricsResponse
		k.Call("GET", "/v1/pipelines/metrics/minutes/1?connections="+connection+"&target=User", nil, nil, &res)
		if len(res.Metrics) != 1 {
			t.Fatalf("expected one connection metric series, got %d", len(res.Metrics))
		}
		series := res.Metrics[0]
		if got := series["connection"]; got != connection {
			t.Fatalf("expected connection %s, got %v", connection, got)
		}
		for _, key := range []string{"workspace", "pipeline"} {
			if _, ok := series[key]; ok {
				t.Fatalf("expected connection metric series to omit %q, got %#v", key, series)
			}
		}
		assertOneMinuteMetricSeriesBuckets(t, series)
	})

	t.Run("non-existing grouped filters return empty metrics", func(t *testing.T) {
		const id = "7QaT3mN7KxP5"
		for _, path := range []string{
			"/v1/pipelines/metrics/minutes/1?pipelines=" + id,
			"/v1/pipelines/metrics/minutes/1?workspaces=" + id,
		} {
			var res pipelineMetricsResponse
			k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &res)
			if res.Metrics == nil {
				t.Fatalf("%s: expected empty metrics list, got nil", path)
			}
			if len(res.Metrics) != 0 {
				t.Fatalf("%s: expected empty metrics list, got %#v", path, res.Metrics)
			}
		}
	})

	t.Run("workspace-scoped workspace group shape", func(t *testing.T) {
		var res pipelineMetricsResponse
		k.Call("GET", "/v1/pipelines/metrics/minutes/1?workspaces="+k.WorkspaceID(), nil, nil, &res)
		if len(res.Metrics) != 1 {
			t.Fatalf("expected one workspace metric series, got %d", len(res.Metrics))
		}
		series := res.Metrics[0]
		if got := series["workspace"]; got != k.WorkspaceID() {
			t.Fatalf("expected workspace %s, got %v", k.WorkspaceID(), got)
		}
		for _, key := range []string{"connection", "pipeline"} {
			if _, ok := series[key]; ok {
				t.Fatalf("expected workspace metric series to omit %q, got %#v", key, series)
			}
		}
		assertOneMinuteMetricSeriesBuckets(t, series)
	})
}

type pipelineMetricsResponse struct {
	Start   string           `json:"start"`
	End     string           `json:"end"`
	Metrics []map[string]any `json:"metrics"`
}

// assertOneMinuteMetricSeriesBuckets verifies that a one-minute metric series
// includes one passed bucket and one failed bucket.
func assertOneMinuteMetricSeriesBuckets(t *testing.T, series map[string]any) {
	t.Helper()
	for _, key := range []string{"passed", "failed"} {
		buckets, ok := series[key].([]any)
		if !ok {
			t.Fatalf("expected metric series %q buckets to be a JSON array, got %T", key, series[key])
		}
		if len(buckets) != 1 {
			t.Fatalf("expected one metric series %q bucket, got %d", key, len(buckets))
		}
	}
}

// assertPipelineMetricsStatus verifies that a pipeline metrics request failed
// with the expected HTTP status.
func assertPipelineMetricsStatus(t *testing.T, err error, status int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected HTTP status %d, got nil error", status)
	}
	statusErr, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T", err)
	}
	if statusErr.Response.Code != status {
		t.Fatalf("expected HTTP status %d, got %d: %s", status, statusErr.Response.Code, statusErr.Response.Text)
	}
}

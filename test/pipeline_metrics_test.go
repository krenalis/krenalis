// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"net/http"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
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

	connection := createPipelineMetricsFixture(t, k)

	t.Run("workspace-scoped connection group shape", func(t *testing.T) {
		var res pipelineMetricsResponse
		k.Call("GET", "/v1/pipelines/metrics/minutes/15?connections="+connection+"&target=User", nil, nil, &res)
		assertPipelineMetricsRange(t, res)
		if len(res.Metrics) != 1 {
			t.Fatalf("expected one connection metric series, got %d", len(res.Metrics))
		}
		series := res.Metrics[0]
		assertMetricSeriesString(t, series, "connection", connection)
		for _, key := range []string{"workspace", "pipeline"} {
			if _, ok := series[key]; ok {
				t.Fatalf("expected connection metric series to omit %q, got %#v", key, series)
			}
		}
		assertMetricSeriesBuckets(t, series, 15)
	})

	t.Run("non-existing grouped selections return empty metrics", func(t *testing.T) {
		const id = "7QaT3mN7KxP5"
		for _, path := range []string{
			"/v1/pipelines/metrics/minutes/1?pipelines=" + id,
			"/v1/pipelines/metrics/minutes/1?workspaces=" + id,
		} {
			var res pipelineMetricsResponse
			k.Call("GET", path, http.Header{"Krenalis-Workspace": nil}, nil, &res)
			assertPipelineMetricsRange(t, res)
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
		k.Call("GET", "/v1/pipelines/metrics/minutes/15?workspaces="+k.WorkspaceID(), nil, nil, &res)
		assertPipelineMetricsRange(t, res)
		if len(res.Metrics) != 1 {
			t.Fatalf("expected one workspace metric series, got %d", len(res.Metrics))
		}
		series := res.Metrics[0]
		assertMetricSeriesString(t, series, "workspace", k.WorkspaceID())
		for _, key := range []string{"connection", "pipeline"} {
			if _, ok := series[key]; ok {
				t.Fatalf("expected workspace metric series to omit %q, got %#v", key, series)
			}
		}
		assertMetricSeriesBuckets(t, series, 15)
	})
}

func createPipelineMetricsFixture(t *testing.T, k *krenalistester.Krenalis) string {
	t.Helper()
	k.UpdateIdentityResolutionSettings(false, nil)
	connection := k.CreateDummy("Dummy source", krenalistester.Source)
	pipeline := k.CreatePipeline(connection, "User", krenalistester.PipelineToSet{
		Name:    "Import users from Dummy",
		Enabled: true,
		InSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), Nullable: true},
		}),
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String().WithMaxLength(300), ReadOptional: true},
		}),
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "email",
			},
		},
	})
	run := k.StartPipelineRun(pipeline)
	k.WaitForRunsCompletion(run)
	return connection
}

type pipelineMetricsResponse struct {
	Start   string                  `json:"start"`
	End     string                  `json:"end"`
	Metrics []map[string]json.Value `json:"metrics"`
}

func assertMetricSeriesString(t *testing.T, series map[string]json.Value, key string, expected string) {
	t.Helper()
	value, ok := series[key]
	if !ok {
		t.Fatalf("expected metric series %q field to exist, got %#v", key, series)
	}
	if !value.IsString() {
		t.Fatalf("expected metric series %q field to be a JSON string, got %s", key, value)
	}
	if got := string(value.Bytes()); got != expected {
		t.Fatalf("expected metric series %q field to be %q, got %q", key, expected, got)
	}
}

// assertPipelineMetricsRange verifies that the response includes a valid
// non-empty time range.
func assertPipelineMetricsRange(t *testing.T, res pipelineMetricsResponse) {
	t.Helper()
	start, err := time.Parse(time.RFC3339Nano, res.Start)
	if err != nil {
		t.Fatalf("expected valid start timestamp, got %q: %s", res.Start, err)
	}
	end, err := time.Parse(time.RFC3339Nano, res.End)
	if err != nil {
		t.Fatalf("expected valid end timestamp, got %q: %s", res.End, err)
	}
	if !end.After(start) {
		t.Fatalf("expected end timestamp %s to be after start timestamp %s", end, start)
	}
}

// assertMetricSeriesBuckets verifies that a metric series includes the expected
// number of passed and failed buckets, each with the six pipeline step
// counters.
func assertMetricSeriesBuckets(t *testing.T, series map[string]json.Value, expectedBuckets int) {
	t.Helper()
	for _, key := range []string{"passed", "failed"} {
		bucketsValue, ok := series[key]
		if !ok {
			t.Fatalf("expected metric series %q buckets to exist, got %#v", key, series)
		}
		if !bucketsValue.IsArray() {
			t.Fatalf("expected metric series %q buckets to be a JSON array, got %s", key, bucketsValue)
		}
		bucketCount := 0
		for bucketIndex, bucket := range bucketsValue.Elements() {
			if !bucket.IsArray() {
				t.Fatalf("expected metric series %q bucket %d to be a JSON array, got %s", key, bucketIndex, bucket)
			}
			counterCount := 0
			for counterIndex, value := range bucket.Elements() {
				if !value.IsNumber() {
					t.Fatalf("expected metric series %q bucket %d counter %d to be a JSON number, got %s", key, bucketIndex, counterIndex, value)
				}
				counterCount++
			}
			if counterCount != 6 {
				t.Fatalf("expected metric series %q bucket %d to contain six counters, got %d", key, bucketIndex, counterCount)
			}
			bucketCount++
		}
		if bucketCount != expectedBuckets {
			t.Fatalf("expected %d metric series %q buckets, got %d", expectedBuckets, key, bucketCount)
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

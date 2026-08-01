// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/krenalis/analytics-go"
)

func TestDispatchEventsToDummy(t *testing.T) {
	const (
		ackWait              = time.Second
		transformationStep   = 3
		outputValidationStep = 4
	)
	t.Setenv("KRENALIS_NATS_ACK_WAIT", ackWait.String())

	// Create a test HTTP server that will receive request sent to it from
	// Dummy. The first received request is written on a channel.
	request := make(chan string, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		select {
		case request <- string(body):
		default:
		}
	}))
	defer ts.Close()

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.Start()
	defer k.Stop()

	// Create a connection that exports to Dummy
	dummyID := k.CreateDummyWithSettings("Dummy", krenalistester.Destination, krenalistester.DummySettings{
		URLForDispatchingEvents: ts.URL,
		OperationDelay:          "3s",
	})
	pipelineID := k.CreateEventPipeline(dummyID, "send_identity", krenalistester.PipelineToSet{
		Name:    "Send events",
		Enabled: true,
		Transformation: &krenalistester.Transformation{
			Mapping: map[string]string{
				"email": "'dummy@email.example.com'",
			},
		},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.String(), CreateRequired: true},
		}),
	})

	// Create a JavaScript event source connection.
	javaScriptID := k.CreateJavaScriptSource("JavaScript (source)", []string{dummyID})
	key := k.EventWriteKeys(javaScriptID)[0]

	k.SendEvent(key, analytics.Identify{
		UserId: "f4ca124298",
	})

	k.RunIdentityResolutionAndWait()

	// Wait for an HTTP request to be sent to Dummy, which will then send it to
	// the test HTTP server. Then check that the request body is correct.
	var received string
	select {
	case received = <-request:
	case <-time.After(10 * time.Second):
		t.Fatalf("expected an event within 10 seconds, got none")
	}
	const expected = `{"email":"dummy@email.example.com"}`
	if received != expected {
		t.Fatalf("expected %q, got %q", expected, received)
	}

	// The dispatch takes longer than AckWait. Without heartbeats NATS redelivers
	// the event while the first dispatch is still waiting, causing these
	// processing metrics to be incremented more than once.
	passed := waitPipelinePassedMetrics(t, k, pipelineID)
	if got := passed[transformationStep]; got != 1 {
		t.Fatalf("expected one transformation, got %d", got)
	}
	if got := passed[outputValidationStep]; got != 1 {
		t.Fatalf("expected one output validation, got %d", got)
	}
}

// waitPipelinePassedMetrics waits until processing metrics for pipelineID are
// visible and returns their totals across all minute buckets.
func waitPipelinePassedMetrics(t *testing.T, k *krenalistester.Krenalis, pipelineID string) [6]int {
	t.Helper()
	poll := time.NewTicker(100 * time.Millisecond)
	defer poll.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	var totals [6]int
	var series int
	for {
		var response struct {
			Metrics []struct {
				Passed [][6]int `json:"passed"`
			} `json:"metrics"`
		}
		k.Call("GET", "/v1/pipelines/metrics/minutes/1?pipelines="+pipelineID, nil, nil, &response)
		totals = [6]int{}
		series = len(response.Metrics)
		if series == 1 {
			for _, bucket := range response.Metrics[0].Passed {
				for step, count := range bucket {
					totals[step] += count
				}
			}
			if totals[3] > 0 && totals[4] > 0 {
				return totals
			}
		}
		select {
		case <-poll.C:
		case <-timeout.C:
			t.Fatalf("expected pipeline processing metrics, got %d series with totals %v", series, totals)
		}
	}
}

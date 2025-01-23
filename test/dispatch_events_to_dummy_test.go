//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meergo/meergo/test/analytics-go"
	"github.com/meergo/meergo/test/meergotester"
	"github.com/meergo/meergo/types"
)

func TestDispatchEventsToDummy(t *testing.T) {

	// Create an test HTTP server that will receive request sent to it from
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
			panic("request already written")
		}
	}))
	defer ts.Close()

	// Test's header (copy-paste me in other tests).
	if testing.Short() {
		t.Skip()
	}
	c := meergotester.InitAndLaunch(t)
	defer c.Stop()

	// Create a connection that exports to Dummy
	dummyID := c.CreateDummyWithSettings("Dummy", meergotester.Destination, meergotester.DummySettings{
		URLForDispatchingEvents: ts.URL,
	})
	c.CreateEventAction(dummyID, "send_identity", meergotester.ActionToSet{
		Name:    "Send events",
		Enabled: true,
		Transformation: &meergotester.Transformation{
			Mapping: map[string]string{
				"email": "'dummy@email.example.com'",
			},
		},
		OutSchema: types.Object([]types.Property{
			{Name: "email", Type: types.Text(), CreateRequired: true},
		}),
	})

	// Create a JavaScript event source connection.
	javaScriptID := c.CreateJavaScriptSource("JavaScript (source)", "example.com", []int{dummyID})
	key := c.EventWriteKeys(javaScriptID)[0]

	c.SendEvent(key, analytics.Identify{
		UserId: "f4ca124298",
	})

	c.StartIdentityResolution()

	// Wait for an HTTP request to be sent to Dummy, which will then send it to
	// the test HTTP server. Then check that the request body is correct.
	var received string
	select {
	case received = <-request:
	case <-time.After(5 * time.Second):
		t.Fatalf("no events received within time limit")
	}
	const expected = `{"email":"dummy@email.example.com"}`
	if received != expected {
		t.Fatalf("expected %q, but Dummy sent %q", expected, received)
	}

}

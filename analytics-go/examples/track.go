package main

import (
	"time"

	"github.com/meergo/meergo/analytics-go"
)

// Use your write key and endpoint.
const WRITE_KEY = ""
const ENDPOINT = ""

func main() {
	// Instantiates a client to send messages to the Meergo API.

	client, _ := analytics.NewWithConfig(WRITE_KEY,
		analytics.Config{
			Endpoint:  ENDPOINT,
			Interval:  30 * time.Second,
			BatchSize: 100,
			Verbose:   true,
		})

	// Enqueues a track event that will be sent asynchronously.

	client.Enqueue(analytics.Track{
		UserId: "test-user",
		Event:  "test-snippet",
	})

	// Flushes any queued messages and closes the client.

	client.Close()
}

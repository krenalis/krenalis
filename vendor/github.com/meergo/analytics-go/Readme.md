# Meergo Go SDK

The Meergo Go SDK lets you send customer event data from your Go applications to your specified destinations.

## SDK setup requirements

- Set up a Meergo account.
- Set up a Go source in the dashboard.
- Copy the write key and the endpoint.

## Installation

You can install the Go SDK via the `go get` command.

To install the SDK in the `GOPATH`, run the following:

```go
go get github.com/meergo/analytics-go
```

## Using the SDK

```go
package main

import (
    "github.com/meergo/analytics-go"
)

func main() {
    // Instantiates a client to send messages to the Meergo API.
    
    // Use your write key and the endpoint in the below placeholders:
    
    client := analytics.New(<WRITE_KEY>, <ENDPOINT>)

    // Enqueues a track event that will be sent asynchronously.
    client.Enqueue(analytics.Track{
        UserId: "test-user",
        Event:  "test-snippet",
    })

    // Flushes any queued messages and closes the client.
    client.Close()
}
```

Alternatively, you can run the following snippet:

```go
package main

import (
    "github.com/meergo/analytics-go"
)

func main() {
    // Instantiates a client to send messages to the Meergo API.
    
    // Use your write key and the endpoint in the below placeholders:
    
    client, _ := analytics.NewWithConfig(<WRITE_KEY>,
		analytics.Config{
			Endpoint:  <ENDPOINT>,
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
```

## Sending events

Refer to the Meergo events documentation for more information on the supported event types.

## License

The Meergo Go SDK is released under the [MIT license](License.md).

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
go get github.com/open2b/analytics-go
```

## Using the SDK

```go
package main

import (
    "github.com/open2b/analytics-go"
)

func main() {
    // Instantiates a client to send messages to the Meergo API.
    
    // Use your write key in the below placeholder:
    
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
    "github.com/open2b/analytics-go"
)

func main() {
    // Instantiates a client to use send messages to the Meergo API.
    
    // User your write key in the below placeholder:
    
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
```

## Gzip support

The Go SDK supports Gzip compression and it is enabled by default. However, you can disable this feature by setting the `DisableGzip` parameter to `true` while initializing the SDK, as shown:

```go
client, _ := analytics.NewWithConfig(WRITE_KEY,
		analytics.Config{
			Endpoint:    ENDPOINT,
			Interval:    30 * time.Second,
			BatchSize:   100,
			Verbose:     true,
			DisableGzip: false  // Enables Gzip compression - set true to disable Gzip.
		})
```

## Sending events

Refer to the Meergo events documentation for more information on the supported event types.

## License

The Meergo Go SDK is released under the [MIT license](License.md).

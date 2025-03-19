{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo Go SDK{% end %}
{% Article %}

<span>Send events</span>
# Go SDK

The Meergo Go SDK lets you send customer event data from your Go applications to your specified destinations.

## Step 1: Create a source Go connection

To create a source Go connection in Meergo:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Go** connector.
4. Click on **Add**.

## Step 2: Import the SDK

1. In the new created Go connection, navigate to **Settings**.
2. Select **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. In your Go module, go get the `"github.com/open2b/analytics-go"` package:
    ```sh
    $ go get github.com/open2b/analytics-go
    ```
5. Import and use the package, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```go
    import "github.com/open2b/analytics-go"

    client := analytics.New("<write key>", "<endpoint>")
    client.Enqueue(analytics.Track{
        UserId: "test-user",
        Event:  "test-snippet",
    })
   ```

## Step 3: Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Go connection you just created and click on **Actions**.
2. Under the **Import Events** action, click on **Add**.
3. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

## Step 4: Test the integration

1. Go to the Go connection you just created and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](./events) for more information on the supported event types.

## Code examples

```go
package main

import (
    "github.com/open2b/analytics-go"
)

func main() {
    // Instantiates a client to send messages to the Meergo API.

    // Use your write key in the below placeholder:

    client := analytics.New("<write key>", "<endpoint>")

    // Enqueues a track event that will be sent asynchronously.
    client.Enqueue(analytics.Track{
        UserId: "test-user",
        Event:  "test-snippet",
    })

    // Flushes any queued messages and closes the client.
    client.Close()
}
```

Alternatively, you can run the following program:

```go
package main

import (
    "time"

    "github.com/open2b/analytics-go"
)

func main() {
    // Instantiates a client to use send messages to the Meergo API.

    // User your write key in the below placeholder:

    client, _ := analytics.NewWithConfig("<write key>",
        analytics.Config{
            Endpoint:  "<endpoint>",
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

### License

The Meergo Go SDK is released under the MIT license.

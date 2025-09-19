{% extends "/layouts/doc.html" %}
{% macro Title string %}Go SDK data source{% end %}
{% Article %}

# Go SDK data source

The **Go** data source allows you to send customer event data using the **Go SDK** from your Go applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Create a source Go connection](#1-create-a-source-go-connection)
  - [2. Import the SDK in your Go application](#2-import-the-sdk-in-your-go-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [License](#license)

## Using the SDK

### 1. Create a source Go connection

First of all, you need a connection in Meergo that can receive events from the Go SDK. To do so:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Go** connector.
4. Click on **Add**.

### 2. Import the SDK in your Go application

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

### 3. Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Go connection you just created and click on **Actions**.
2. Choose **Import events** (to import event data) or **Import users** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the Go connection created at step 1 and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## License

The Meergo Go SDK is released under the MIT license.

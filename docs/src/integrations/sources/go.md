{% extends "/layouts/doc.html" %}
{% macro Title string %}Go (Source){% end %}
{% Article %}

# Go SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_Go_SDK-blue?logo=github)](https://github.com/open2b/analytics-go)

The source connector for Go allows you to send customer event data using the Go SDK from your Go applications to Meergo.

## Using the SDK

### 1. Add source connection for Go

First of all, you need a connection in Meergo that can receive events from the Go SDK. To do so:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Go**; you can use the search bar at the top or filter by category.
4. Click on the connector for **Go**. A panel will open on the right.
5. Click on **Add source...**. The `Add source connection for Go` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

### 2. Import the SDK in your Go application

1. In the new created source connection for Go, navigate to **Settings**.
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

1. Go to the source connection for Go you just created and click on **Actions**.
2. Choose **Import events into warehouse** (to import event data) or **Import users into warehouse** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
5. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the source connection for Go created at step 1 and click on **Event debugger**.
2. Execute your application to send some events.
3. Click on a received event in the **Live events** section to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## SDK source code

The source code of the Meergo Go SDK is [available on GitHub](https://github.com/open2b/analytics-go) and distributed under the MIT license.

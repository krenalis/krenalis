{% extends "/layouts/doc.html" %}
{% macro Title string %}C# SDK data source{% end %}
{% Article %}

# C# SDK data source

The **.NET** data source allows you to send customer event data using the **C# SDK** from your .NET applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Create a source C# connection](#1-create-a-source-c-connection)
  - [2. Import the SDK in your C# application](#2-import-the-sdk-in-your-c-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [License](#license)

## Using the SDK

### 1. Create a source C# connection

First of all, you need a connection in Meergo that can receive events from the C# SDK. To do so:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **.NET** connector.
4. Click on **Add**.

### 2. Import the SDK in your C# application

1. In the new created .NET connection, navigate to **Settings**.
2. Select **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. Install `Meergo.Analytics.CSharp` using NuGet:
    ```sh
    Install-Package Meergo.Analytics.CSharp -Version <version>
    ```
5. Import and use the package, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```csharp
    using Meergo.Analytics;

    var config = new Config()
        .SetEndpoint("<endpoint>");

    Analytics.Initialize("<write key>", config);

    Analytics.Client.Track("Efg678Mnu", "Product added to cart", new Properties() {
        { "price", 32.17 }
    });
    ```

### 3. Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the .NET connection you just created and click on **Actions**.
2. Choose **Import events** (to import event data) or **Import users** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
5. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the .NET connection created at step 1 and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## License

The Meergo C# SDK is released under the MIT license.

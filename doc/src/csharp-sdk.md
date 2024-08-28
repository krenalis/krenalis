# Meergo C# SDK

The Meergo C# SDK lets you send customer event data from your .NET applications to your specified destinations.

## Step 1: Create a Source .NET Connection

To create a source .NET connection in Meergo:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **.NET** connector.
4. Click on **Add**.

## Step 2: Import the SDK

1. In the new created .NET connection, navigate to **Settings**.
2. Select **Write Keys**.
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

## Step 3: Add an Action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the .NET connection you just created and click on **Actions**.
2. Under the **Import Events** action, click on **Add**.
3. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

## Step 4: Test the integration

1. Go to the .NET connection you just created and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](./events.md) for more information on the supported event types.

### License

The Meergo C# SDK is released under the MIT license.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Node.js SDK data source{% end %}
{% Article %}

# Node.js SDK data source

The **Node.js** data source allows you to send customer event data using the **Node.js SDK** from your Node.js applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Create a source Node.js connection](#1-create-a-source-Node.js-connection)
  - [2. Import the SDK in your Node.js application](#2-import-the-sdk-in-your-Node.js-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [License](#license)

## Using the SDK

### 1. Create a source Node.js connection

First of all, you need a connection in Meergo that can receive events from the Node.js SDK. To do so:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search for the **Node.js** source; you can use the search bar at the top or filter by category.
4. Click on the **Node.js** connector. A panel will open on the right with information about **Node.js**.
5. Click on **Add source**. The `Add Node.js source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

### 2. Import the SDK in your Node.js application

The Node SDK can be imported with `import` into Node projects, using ES6 modules.

1. In the new created Node connection, navigate to **Settings**.
2. Click on **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. In your project, install the `meergo-node-sdk` npm package:
    ```sh
    $ npm install meergo-node-sdk --save
    ```
5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```javascript
    import Analytics from 'meergo-node-sdk';
   
    const meergoAnalytics = new Analytics('<write key>', '<endpoint>');
    meergoAnalytics.track({
        userId: "test-user",
        event:  "test-snippet",
    });
    ```

### 3. Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Node.js connection you just created and click on **Actions**.
2. Choose **Import events into warehouse** (to import event data) or **Import users into warehouse** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the Node.js connection created at step 1 and click on **Event debugger**.
2. Execute your application to send some events.
3. Click on a received event in the **Live events** section to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## License

The Meergo Node.js SDK is released under the MIT license.

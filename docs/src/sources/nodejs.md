{% extends "/layouts/doc.html" %}
{% macro Title string %}Node.js (Source){% end %}
{% Article %}

# Node.js SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_Node_SDK-blue?logo=github)](https://github.com/open2b/analytics-node)

The source connector for Node.js allows you to send customer event data using the Node.js SDK from your Node.js applications to Meergo.

## Using the SDK

### 1. Add source connection for Node.js

First of all, you need a connection in Meergo that can receive events from the Node.js SDK. To do so:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Node.js**; you can use the search bar at the top or filter by category.
4. Click on the connection for **Node.js** connector. A panel will open on the right.
5. Click on **Add source...**. The `Add Node.js source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

### 2. Import the SDK in your Node.js application

The Node SDK can be imported with `import` into Node projects, using ES6 modules.

1. In the new created connection for Node, navigate to **Settings**.
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

1. Go to the source commection for Node.js you just created and click on **Actions**.
2. Choose **Import events into warehouse** (to import event data) or **Import users into warehouse** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the source connection for Node.js created at step 1 and click on **Event debugger**.
2. Execute your application to send some events.
3. Click on a received event in the **Live events** section to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## SDK source code

The source code of the Meergo Node SDK is [available on GitHub](https://github.com/open2b/analytics-node) and distributed under the **MIT license**.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo Node SDK{% end %}
{% Article %}

<span>Send events</span>
# Node SDK

The Meergo Node SDK lets you send customer event data from your Node applications to your specified destinations.

## Step 1: Create a source Node connection

To create a source Node connection in Meergo:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Node** connector.
4. Click on **Add**.

## Step 2: Import the SDK

Below are outlined the various alternative methods for importing the SDK to suit your requirements.

### Import using `import`

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

### Import using `require`

The Node SDK can be imported with `require` into Node projects, using CommonJS modules.

1. In the new created Node connection, navigate to **Settings**.
2. Click on **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. In your project, install the `meergo-node-sdk` npm package:
    ```sh
    $ npm install meergo-node-sdk --save
    ```
5. Import and use the SDK, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```javascript
    const Analytics = require('meergo-node-sdk');
   
    const meergoAnalytics = new Analytics('<write key>', '<endpoint>');
    meergoAnalytics.track({
        userId: "test-user",
        event:  "test-snippet",
    });
    ```

## Step 3: Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Node connection you just created and click on **Actions**.
2. Under the **Import Events** action, click on **Add**.
3. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

## Step 4: Test the integration

1. Go to the Node connection you just created and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](./events) for more information on the supported event types.

### License

The Meergo Node SDK is released under the MIT license.

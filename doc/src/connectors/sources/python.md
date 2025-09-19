{% extends "/layouts/doc.html" %}
{% macro Title string %}Python SDK data source{% end %}
{% Article %}

# Python SDK data source

The **Python** data source allows you to send customer event data using the **Python SDK** from your Python applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Create a source Python connection](#1-create-a-source-python-connection)
  - [2. Import the SDK in your Python application](#2-import-the-sdk-in-your-python-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [License](#license)

## Using the SDK

### 1. Create a source Python connection

First of all, you need a connection in Meergo that can receive events from the Python SDK. To do so:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Python** connector.
4. Click on **Add**.

### 2. Import the SDK in your Python application

1. In the new created Python connection, navigate to **Settings**.
2. Select **Event write keys**.
3. Copy the Write Key and the Endpoint.
4. Install `meergo-analytics-python` using pip:
    ```sh
    pip3 install meergo-analytics-python
    ```
5. Import and use the package in your application, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```python
    import meergo.analytics as analytics

    analytics.write_key = '<write key>'
    analytics.endpoint = '<endpoint>'

    analytics.track('er56789012', 'Product added to cart', {
        'price': 32.50
    })
    ```

### 3. Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Python connection you just created and click on **Actions**.
2. Choose **Import events** (to import event data) or **Import users** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the Python connection created at step 1 and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## License

The Meergo Python SDK is released under the MIT license.

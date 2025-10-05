{% extends "/layouts/doc.html" %}
{% macro Title string %}Python (Source){% end %}
{% Article %}

# Python SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_Python_SDK-blue?logo=github)](https://github.com/open2b/analytics-python)

The source connector for Python allows you to send customer event data using the Python SDK from your Python applications to Meergo.

- [Using the SDK](#using-the-sdk)
  - [1. Add source connection for Python](#1-add-source-connection-for-python)
  - [2. Import the SDK in your Python application](#2-import-the-sdk-in-your-python-application)
  - [3. Add an action](#3-add-an-action)
  - [4. Test the integration](#4-test-the-integration)
- [SDK source code](#sdk-source-code)
- [License](#license)

## Using the SDK

### 1. Add source connection for Python

First of all, you need a connection in Meergo that can receive events from the Python SDK. To do so:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Python**; you can use the search bar at the top or filter by category.
4. Click on the connector for **Python**. A panel will open on the right.
5. Click on **Add source...**. The `Add source connection for Python` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

### 2. Import the SDK in your Python application

1. In the new created source connection for Python, navigate to **Settings**.
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
2. Choose **Import events into warehouse** (to import event data) or **Import users into warehouse** (to import user data from events).
3. Fill in the necessary information in the action.
4. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

### 4. Test the integration

1. Go to the source connection for Python created at step 1 and click on **Event debugger**.
2. Execute your application to send some events.
3. Click on a received event in the **Live events** section to view its details.

Refer to the [Meergo events documentation](../../events) for more information on the supported event types.

## SDK source code

The source code of the Meergo Python SDK is [available on GitHub](https://github.com/open2b/analytics-python).

## License

The Meergo Python SDK is released under the MIT license.

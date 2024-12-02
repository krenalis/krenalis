{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo Python SDK{% end %}
{% Article %}

<span>Send events</span>
# Python SDK

The Meergo Python SDK lets you send customer event data from your Python applications to your specified destinations.

## Step 1: Create a source Python connection

To create a source Python connection in Meergo:

1. Click on **Connections**.
2. Click on **Add a new source**.
3. From the list of connectors, select the **Python** connector.
4. Click on **Add**.

## Step 2: Import the SDK

1. In the new created Python connection, navigate to **Settings**.
2. Select **Write Keys**.
3. Copy the Write Key and the Endpoint.
4. Install `meergo-analytics-python` using pip:
    ```sh
    pip3 install meergo-analytics-python
    ```
5. Import and use the package, replacing `<write key>` and `<endpoint>` respectively with the previously copied Write Key and Endpoint:
    ```python
    import meergo.analytics as analytics

    analytics.write_key = '<write key>'
    analytics.endpoint = '<endpoint>'

    analytics.track('er56789012', 'Product added to cart', {
        'price': 32.50
    })
    ```

## Step 3: Add an action

Now you can choose to collect only the events, or import the users, or both:

1. Go to the Python connection you just created and click on **Actions**.
2. Under the **Import Events** action, click on **Add**.
3. Confirm by clicking **Add**.
4. Enable the action by toggling the switch in the **Enabled** column.

## Step 4: Test the integration

1. Go to the Python connection you just created and click on **Live events**.
2. Execute your application to send some events.
3. Click on a received event in **Live events** to view its details.

Refer to the [Meergo events documentation](./events.md) for more information on the supported event types.

### License

The Meergo Python SDK is released under the MIT license.

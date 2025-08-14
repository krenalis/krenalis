{% extends "/layouts/doc.html" %}
{% macro Title string %}Python data source{% end %}
{% Article %}

# Python data source

The **Python** data source is designed for applications built with Python that require integration with Meergo for event tracking and user data management. This data source enables you to receive events from a server-based Python application, including user information. Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the Python data source, you will need the [Python SDK](../../developers/python-sdk) from Meergo. The SDK provides the necessary functionality for sending different types of events, ensuring a smooth integration with the Meergo platform.

> The [Python SDK](../../developers/python-sdk) is an open-source Python library licensed under the MIT License.

### On this page

* [Add a Python data source](#add-a-python-data-source)
* [Import events into the workspace's data warehouse](#import-events-into-the-workspaces-data-warehouse)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a Python data source

1. From the **Meergo admin**, navigate to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Python** source; you can use the search bar at the top or filter by category.
4. Click on the **Python** connector. A panel will open on the right with information about **Python**.
5. Click on **Add source**. The `Add Python source connection` page will appear.
6. In the **Name** field, provide a name to easily identify the source later (e.g., the name of the Python application or server).
7. Click **Add**.

Once the Python data source is added, you will be directed to the **Actions** page, where you can view the specific actions that will be performed with the events received from this source.

### Import events into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Python data source from which you want to import the events.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import events**.
5. Click **Add** to add the action.

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Python data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import users**.
5. Click **Add** to add the action.

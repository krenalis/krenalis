{% extends "/layouts/doc.html" %}
{% macro Title string %}Node.js data source{% end %}
{% Article %}

# Node.js data source

The **Node.js** data source is designed for applications built on the Node.js platform that require integration with Meergo for event tracking and user data management. This data source enables you to receive events from a server-based Node.js application, including user information. Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the Node.js data source, you will need the [Node SDK](../../developers/node-sdk) from Meergo. The SDK provides the necessary functionality for sending different types of events, ensuring a smooth integration with the Meergo platform.

> The [Node SDK](../../developers/node-sdk) is an open-source Node.js library licensed under the MIT License.

### On this page

* [Add a Node.js data source](#add-a-nodejs-data-source)
* [Import events into the workspace's data warehouse](#import-events-into-the-workspaces-data-warehouse)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a Node.js data source

1. From the **Meergo admin**, navigate to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Node.js** source; you can use the search bar at the top or filter by category.
4. Click on the **Node.js** connector. A panel will open on the right with information about **Node.js**.
5. Click on **Add source**. The `Add Node.js source connection` page will appear.
6. In the **Name** field, provide a name to easily identify the source later (e.g., the name of the Node.js application or server).
7. Click **Add**.

Once the Node.js data source is added, you will be directed to the **Actions** page, where you can view the specific actions that will be performed with the events received from this source.

### Import events into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Node.js data source from which you want to import the events.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import events**.
5. Click **Add** to add the action.

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Node.js data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import users**.
5. Click **Add** to add the action.

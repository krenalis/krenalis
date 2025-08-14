{% extends "/layouts/doc.html" %}
{% macro Title string %}Java data source{% end %}
{% Article %}

# Java data source

The **Java** data source is designed for applications built on the Java platform that require integration with Meergo for event tracking and user data management. This data source enables you to receive events from a server-based Java application, including user information. Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the Java data source, you will need the [Java SDK](../../developers/java-sdk) from Meergo. The SDK provides the necessary functionality for sending different types of events, ensuring a smooth integration with the Meergo platform.

> The [Java SDK](../../developers/java-sdk) is an open-source Java library licensed under the MIT License.

### On this page

* [Add a Java data source](#add-a-java-data-source)
* [Import events into the workspace's data warehouse](#import-events-into-the-workspaces-data-warehouse)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a Java data source

1. From the **Meergo Admin console**, navigate to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Java** source; you can use the search bar at the top or filter by category.
4. Click on the **Java** connector. A panel will open on the right with information about **Java**.
5. Click on **Add source**. The `Add Java source connection` page will appear.
6. In the **Name** field, provide a name to easily identify the source later (e.g., the name of the Java application or server).
7. Click **Add**.

Once the Java data source is added, you will be directed to the **Actions** page, where you can view the specific actions that will be performed with the events received from this source.

### Import events into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the Java data source from which you want to import the events.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import events**.
5. Click **Add** to add the action.

### Import users into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the Java data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import users**.
5. Click **Add** to add the action.

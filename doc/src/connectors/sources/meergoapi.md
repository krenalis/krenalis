{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo API data source{% end %}
{% Article %}

# Meergo API data source

**Meergo API** is a connector to interact directly with Meergo APIs from your application.

This connector allows you to call Meergo APIs and import events and users directly from your application, regardless of the technology and programming language you use, since it allows you to interact directly via HTTP calls with Meergo APIs.

**This connector is useful in those cases where Meergo does not provide a dedicated SDK for your language**. It can also be used in those cases where you want to interact with Meergo directly via the command line, making the calls with tools like `curl`.

In this regard, it is therefore useful to see the [API reference documentation](/api/).

Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the Meergo API data source, you will need any language or application that can make HTTP calls.

### On this page

* [Add a Meergo API data source](#add-a-meergo-api-data-source)
* [Import events into the workspace's data warehouse](#import-events-into-the-workspaces-data-warehouse)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a Meergo API data source

1. From the **Meergo admin**, navigate to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Meergo API** source using the search bar at the top.
4. Next to the **Meergo API** source, click the **+** icon to open the source addition page.
5. (Optional) In the **Name** field, provide a name to easily identify the source later (e.g., the name of the application or server).
6. Click **Add**.

Once the Meergo API data source is added, you will be directed to the **Actions** page, where you can view the specific actions that will be performed with the events received from this source.

### Import events into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Meergo API data source from which you want to import the events.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import events**.
5. Click **Add** to add the action.

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Meergo API data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import users**.
5. Click **Add** to add the action.

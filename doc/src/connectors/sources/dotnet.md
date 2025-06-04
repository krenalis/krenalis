{% extends "/layouts/doc.html" %}
{% macro Title string %}.NET data source{% end %}
{% Article %}

# .NET data source

The **.NET** data source is designed for applications built on the .NET platform that require integration with Meergo for event tracking and user data management. This data source enables you to receive events from a server-based .NET application, including user information. Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the .NET data source, you will need the [C# SDK](../../developers/csharp-sdk) from Meergo. The SDK provides the necessary functionality for sending different types of events, ensuring a smooth integration with the Meergo platform.

> The [C# SDK](../../developers/csharp-sdk) is an open-source C# library licensed under the MIT License.

### On this page

* [Add a .NET data source](#add-a-net-data-source)
* [Import events into the workspace's data warehouse](#import-events-into-the-workspaces-data-warehouse)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a .NET data source

1. From the **Meergo admin**, navigate to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **.NET** source using the search bar at the top.
4. Next to the **.NET** source, click the **+** icon to open the source addition page.
5. (Optional) In the **Name** field, provide a name to easily identify the source later (e.g., the name of the .NET application or server).
6. Click **Add**.

Once the .NET data source is added, you will be directed to the **Actions** page, where you can view the specific actions that will be performed with the events received from this source.

### Import events into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the .NET data source from which you want to import the events.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import events**.
5. Click **Add** to add the action.

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the .NET data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import users**.
5. Click **Add** to add the action.

# Python data source

The **Python** data source is designed for applications built with Python that require integration with Meergo for event tracking and user data management. This data source enables you to receive events from a server-based Python application, including user information. Once events are received, you can:

- **Send events to destinations**: These are applications or services capable of processing the events.
- **Store events in the workspace's data warehouse**: Ideal for data analysis and reporting purposes.
- **Extract user data for identification**: Helps in identifying both authenticated and anonymous users, facilitating unification within the workspace's data warehouse.

To use the Python data source, you will need the [Python SDK](../python-sdk.md) from Meergo. The SDK provides the necessary functionality for sending different types of events, ensuring a smooth integration with the Meergo platform.

> The [Python SDK](../python-sdk.md) is an open-source Python library licensed under the MIT License.

### On this page

- [Add a Python data source](#add-a-python-data-source)

### Add a Python data source

1. From the **Meergo admin**, navigate to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Python** source using the search bar at the top.
4. Next to the **Python** source, click the **+** icon to open the source addition page.
5. (Optional) In the **Name** field, provide a name to easily identify the source later (e.g., the name of the Python application or server).
6. (Optional) From the **Strategy** dropdown, select a strategy. You can modify this later if needed.
7. Click **Add**.

Once the Python data source is added, you will be directed to the **Actions** page, where you can view the specific actions that will be performed with the events received from this source.

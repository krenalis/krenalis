{% extends "/layouts/doc.html" %}
{% macro Title string %}HTTP GET (Source){% end %}
{% Article %}

# HTTP GET (Source)

The source connector HTTP GET allows you to import users from files accessed via HTTP endpoints. You can unify this data as users within Meergo by importing files in various formats, such as CSV, Excel, and others.

Once a source connection for HTTP GET is configured, you can easily customize how the data is read and processed.

### On this page

- [Add source connection HTTP GET](#add-source-connection-http-get)
- [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add source connection HTTP GET

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **HTTP GET**; you can use the search bar at the top or filter by category.
4. Click on the connector **HTTP GET**. A panel will open on the right with information about **HTTP GET**.
5. Click on **Add source**. The `Add HTTP GET source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Host** field, enter the host where the file or files you wish to read are located.
8. In the **Port** field, enter the port.
9. (Optional) In the **Headers** fields,  specify any headers that should be included in the request for reading the files. 
10. Click **Add**.

Once the source connection HTTP GET is added, the **Actions** page will be displayed. Here, you can add an action for each file to be read using the newly added HTTP GET data source. Configure each action with the desired settings for file format, filters for user data, and any additional processing requirements.

### Import users into the workspace's data warehouse

1. In the Meergo Admin console panel, navigate to **Connections > Sources**.
2. Click on the source connection HTTP GET from which you wish to import users.
3. Click **Add New Action**, then select **Import Users**.
4. From the **Format** menu, choose the file format from which you want to import users.

Continue with step 5 based on the selected file format:
* [CSV](csv#import-users-into-the-workspaces-data-warehouse)
* [Excel](excel#import-users-into-the-workspaces-data-warehouse)
* [JSON](json#import-users-into-the-workspaces-data-warehouse)
* [Parquet](parquet#import-users-into-the-workspaces-data-warehouse)

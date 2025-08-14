{% extends "/layouts/doc.html" %}
{% macro Title string %}HTTP Files data source{% end %}
{% Article %}

# HTTP Files data source

The HTTP Files data source allows you to import users from files accessed via HTTP endpoints. You can unify this data as users within Meergo by importing files in various formats, such as CSV, Excel, and others.

Once the HTTP Files data source is configured, you can easily customize how the data is read and processed.

### On this page

* [Add an HTTP Files data source](#add-an-http-files-data-source)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add an HTTP Files data source

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **HTTP Files** source; you can use the search bar at the top or filter by category.
4. Click on the **HTTP Files** connector. A panel will open on the right with information about **HTTP Files**.
5. Click on **Add source**. The `Add HTTP Files source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the `Host` field, enter the host where the file or files you wish to read are located.
8. In the `Port` field, enter the port.
9. (Optional) In the `Headers` fields,  specify any headers that should be included in the request for reading the files. 
10. Click **Add**.

Once the HTTP Files data source is added, the **Actions** page will be displayed. Here, you can add an action for each file to be read using the newly added HTTP Files data source. Configure each action with the desired settings for file format, filters for user data, and any additional processing requirements.

### Import users into the workspace's data warehouse

1. In the Meergo Admin console panel, navigate to **Connections > Sources**.
2. Click on the HTTP Files source from which you wish to import users.
3. Click **Add New Action**, then select **Import Users**.
4. From the **Format** menu, choose the file format from which you want to import users.

Continue with step 5 based on the selected file format:
* [CSV](csv#import-users-into-the-workspaces-data-warehouse)
* [Excel](excel#import-users-into-the-workspaces-data-warehouse)
* [JSON](json#import-users-into-the-workspaces-data-warehouse)
* [Parquet](parquet#import-users-into-the-workspaces-data-warehouse)

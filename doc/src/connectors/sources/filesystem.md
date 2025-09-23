{% extends "/layouts/doc.html" %}
{% macro Title string %}Filesystem data source{% end %}
{% Article %}

# Filesystem data source

Filesystem is a connector for testing the import of files on the local filesystem.

Using this connector you can import files that are in the filesystem of the installation that Meergo is running on.

Its sole purpose is to test file imports and explore the various file formats supported by Meergo, and it is strongly discouraged to use this connector in production, as it can freely access the filesystem.

### On this page

- [Add a Filesystem data source](#add-a-filesystem-data-source)
- [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a Filesystem data source

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search for the **Filesystem** source; you can use the search bar at the top or filter by category.
4. Click on the **Filesystem** connector. A panel will open on the right with information about **Filesystem**.
5. Click on **Add source**. The `Add Filesystem source connection` page will appear.
6. Click **Add**.

Once the Filesystem data source is added, the **Actions** page will be displayed. Here, you can add an action for each file to be read using the newly added Filesystem data source. Configure each action with the desired settings for file format, filters for user data, and any additional processing requirements.

### Import users into the workspace's data warehouse

1. In the Meergo Admin console panel, navigate to **Connections > Sources**.
2. Click on the Filesystem source from which you wish to import users.
3. Click **Add New Action**, then select **Import Users**.
4. From the **Format** menu, choose the file format from which you want to import users.

Continue with step 5 based on the selected file format:
* [CSV](csv#import-users-into-the-workspaces-data-warehouse)
* [Excel](excel#import-users-into-the-workspaces-data-warehouse)
* [JSON](json#import-users-into-the-workspaces-data-warehouse)
* [Parquet](parquet#import-users-into-the-workspaces-data-warehouse)

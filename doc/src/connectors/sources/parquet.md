{% extends "/layouts/doc.html" %}
{% macro Title string %}Parquet data source{% end %}
{% Article %}

# Parquet data source

The Parquet data source allows you to import user data from Parquet files, which can then be unified as users within Meergo.

> Before adding a Parquet data source, ensure that you have configured a storage data source such as S3, SFTP, or HTTP Files. If you haven’t set up a storage source yet, please do so before proceeding with the Parquet file import.

### On this page

* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the storage data source from which you want to import the Parquet file.
3. If there are no actions, click  **Add**; otherwise, click  **Add new action**.
4. From the **Format** menu, select **Parquet**.
5. In the **Path** field, enter the path of the Parquet file relative to the storage root path. When you enter the relative path, the absolute path of the file will be displayed, allowing you to verify that the path you have entered is correct.
6. Optional: In the **Compression** field, if the JSON file is compressed, select the compression format; Meergo automatically decompresses the file upon reading.
7. Click **Preview** to view a preview of the file.
8. Click **Confirm** to confirm your selections. You can change them at any time later if needed.

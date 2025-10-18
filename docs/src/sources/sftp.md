{% extends "/layouts/doc.html" %}
{% macro Title string %}SFTP Data Source{% end %}
{% Article %}

# SFTP (Source)

The source connector for SFTP allows you to import users from files stored on an SFTP server. You can unify this data as users within Meergo by importing files in various formats, such as CSV, Excel, and others.

SFTP (Secure File Transfer Protocol) is a secure network protocol used for transferring files over a secure connection, providing encryption and data integrity.

### On this page

* [Add source connection for SFTP](#add-source-connection-for-sftp)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add source connection for SFTP

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **SFTP**; you can use the search bar at the top or filter by category.
4. Click on the connector for **SFTP**. A panel will open on the right.
5. Click on **Add source...**. The `Add source connection for SFTP` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Host** field, enter the hostname or IP address of the SFTP server where the files are stored.
8. In the **Port** field, specify the port number used for the SFTP connection (default is usually 22).
9. In the **Username** field, enter the username required to authenticate with the SFTP server.
10. In the **Password** field, enter the password associated with the username for authentication.
11. Click **Add**.

Once the source connection for SFTP is added, the **Actions** page will be displayed. Here, you can add an action for each file to be read using the newly added SFTP data source. Configure each action with the desired settings for file format, filters for user data, and any additional processing requirements.

### Import users into the workspace's data warehouse

1. In the Meergo Admin console panel, navigate to **Connections > Sources**.
2. Click on the source connections for SFTP from which you wish to import users.
3. Click **Add New Action**, then select **Import Users**.
4. From the **Format** menu, choose the file format from which you want to import users.

Continue with step 5 based on the selected file format:
* [CSV](csv#import-users-into-the-workspaces-data-warehouse)
* [Excel](excel#import-users-into-the-workspaces-data-warehouse)
* [JSON](json#import-users-into-the-workspaces-data-warehouse)
* [Parquet](parquet#import-users-into-the-workspaces-data-warehouse)

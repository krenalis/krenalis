# SFTP Data Source

The SFTP data source allows you to import users from files stored on an SFTP server. You can unify this data as users within Meergo by importing files in various formats, such as CSV, Excel, and others.

SFTP (Secure File Transfer Protocol) is a secure network protocol used for transferring files over a secure connection, providing encryption and data integrity.

### On this page

* [Add an SFTP data source](#add-an-sftp-data-source)

### Add an SFTP Data Source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **SFTP** source; you can use the search bar at the top to assist you.
4. Next to the **SFTP** source, click the **+** icon.
5. On the `Add SFTP source connection` page, in the **Name** field, enter a name for the source to easily recognize it later.
6. In the `Host` field, enter the hostname or IP address of the SFTP server where the files are stored.
7. In the `Port` field, specify the port number used for the SFTP connection (default is usually 22).
8. In the `Username` field, enter the username required to authenticate with the SFTP server.
9. In the `Password` field, enter the password associated with the username for authentication.
10. Click **Add**.

Once the SFTP data source is added, the **Actions** page will be displayed. Here, you can add an action for each file to be read using the newly added SFTP data source. Configure each action with the desired settings for file format, filters for user data, and any additional processing requirements.

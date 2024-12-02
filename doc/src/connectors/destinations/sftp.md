{% extends "/layouts/doc.html" %}
{% macro Title string %}SFTP Data Destination{% end %}
{% Article %}

# SFTP Data Destination

The SFTP data destination allows you to export unified users (i.e., users consolidated through identity resolution) in various file formats, such as CSV or Excel, and send them directly to an SFTP server. The files are uploaded securely using the SFTP protocol.

SFTP (Secure File Transfer Protocol) is a secure network protocol used for transferring files over a secure connection, providing encryption and data integrity.

### On this page

* [Add an SFTP data destination](#add-an-sftp-data-destination)

### Add an SFTP Data Destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destination**.
3. Search for the **SFTP** destination; you can use the search bar at the top to assist you.
4. Next to the **SFTP** destination, click the **+** icon.
5. On the `Add SFTP destination connection` page, in the **Name** field, enter a name for the destination to easily recognize it later.
6. In the `Host` field, enter the hostname or IP address of the SFTP server where the files will be uploaded.
7. In the `Port` field, specify the port number used for the SFTP connection (default is usually 22).
8. In the `Username` field, enter the username required to authenticate with the SFTP server.
9. In the `Password` field, enter the password associated with the username for authentication.
10. Click **Add**.

Once the SFTP data destination is added, the **Actions** page will be displayed. Here, you can configure multiple files for export by selecting the file format, applying filters to determine which users to include, specifying the target SFTP server, and setting a schedule for how frequently each export should occur.

{% extends "/layouts/doc.html" %}
{% macro Title string %}SFTP Data Destination{% end %}
{% Article %}

# SFTP Data Destination

The SFTP data destination allows you to export unified users (i.e., users consolidated through identity resolution) in various file formats, such as CSV or Excel, and send them directly to an SFTP server. The files are uploaded securely using the SFTP protocol.

SFTP (Secure File Transfer Protocol) is a secure network protocol used for transferring files over a secure connection, providing encryption and data integrity.

### On this page

* [Add an SFTP data destination](#add-an-sftp-data-destination)

### Add an SFTP Data Destination

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search for the **SFTP** destination; you can use the search bar at the top or filter by category.
4. Click on the **SFTP** connector. A panel will open on the right with information about **SFTP**.
5. Click on **Add destination**. The `Add SFTP destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **Host** field, enter the hostname or IP address of the SFTP server where the files will be uploaded.
8. In the **Port** field, specify the port number used for the SFTP connection (default is usually 22).
9. In the **Username** field, enter the username required to authenticate with the SFTP server.
10. In the **Password** field, enter the password associated with the username for authentication.
11. Click **Add**.

Once the SFTP data destination is added, the **Actions** page will be displayed. Here, you can configure multiple files for export by selecting the file format, applying filters to determine which users to include, specifying the target SFTP server, and setting a schedule for how frequently each export should occur.

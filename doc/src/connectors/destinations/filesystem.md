{% extends "/layouts/doc.html" %}
{% macro Title string %}Filesystem (Destination){% end %}
{% Article %}

# Filesystem (Destination)

The destination connection Filesystem is a connector for testing the export of files to the local filesystem.

Using this connector you can export files to the filesystem of the installation that Meergo is running on.

Its sole purpose is to test file exports and explore the various file formats supported by Meergo, and it is strongly discouraged to use this connector in production, as it can freely access the filesystem.

### On this page

- [Add destionation connection Filesystem](#add-destination-connection-filesystem)

### Add destination connection Filesystem

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **Filesystem** ; you can use the search bar at the top or filter by category.
4. Click on the connector **Filesystem**. A panel will open on the right.
5. Click on **Add destination...**. The `Add destination connection for Filesystem` page will appear.
6. Click **Add**.

Once the destination connection Filesystem is added, the **Actions** page will be displayed. Here, you can configure multiple files for export by selecting the file format, applying filters to determine which users to include, and setting a schedule for how frequently each export should occur.

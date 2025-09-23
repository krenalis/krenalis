{% extends "/layouts/doc.html" %}
{% macro Title string %}Filesystem data destination{% end %}
{% Article %}

# Filesystem data destination

Filesystem is a connector for testing the export of files to the local filesystem.

Using this connector you can export files to the filesystem of the installation that Meergo is running on.

Its sole purpose is to test file exports and explore the various file formats supported by Meergo, and it is strongly discouraged to use this connector in production, as it can freely access the filesystem.

### On this page

- [Add an Filesystem data destination](#add-an-filesystem-data-destination)

### Add an Filesystem data destination

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search for the **Filesystem** destination; you can use the search bar at the top or filter by category.
4. Click on the **Filesystem** connector. A panel will open on the right with information about **Filesystem**.
5. Click on **Add destination**. The `Add Filesystem destination connection` page will appear.
6. Click **Add**.

Once the Filesystem data destination is added, the **Actions** page will be displayed. Here, you can configure multiple files for export by selecting the file format, applying filters to determine which users to include, and setting a schedule for how frequently each export should occur.

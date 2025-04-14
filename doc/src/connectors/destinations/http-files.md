{% extends "/layouts/doc.html" %}
{% macro Title string %}HTTP Files data destination{% end %}
{% Article %}

# HTTP Files data destination

The HTTP Files data destination allows you to export unified users (i.e., users consolidated through identity resolution) in various file formats, such as CSV or Excel, and send them directly to HTTP endpoints via POST. The receiving server is responsible for processing and saving the files.

Once the HTTP Files data destination is configured, you can easily customize the file generation. You can configure multiple files for export by selecting the file format, applying filters to determine which users to include, specifying the target endpoint, and setting a schedule for how frequently each export should occur.

### On this page

* [Add an HTTP Files data destination](#add-an-http-files-data-destination)

### Add an HTTP Files data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destination**.
3. Search for the **HTTP Files** destination; you can use the search bar at the top to help you.
4. Next to the **HTTP Files** destination, click the **+** icon.
5. On the `Add HTTP Files destination connection` page, in the **Name** field, enter a name for the destination to easily recognize it later.
6. In the `Host` field, enter the host where the file or files you wish to write are located.
7. In the `Port` field, enter the port.
8. (Optional) In the `Headers` fields,  specify any headers that should be included in the request for writing the files.
9. Click **Add**.

Once the HTTP Files data destination is added, the **Actions** page will be displayed. Here, you can add an action for each file you want to generate using the newly added HTTP Files data destination. Configure each action with the desired settings for file format, user filters, endpoint, and scheduling.

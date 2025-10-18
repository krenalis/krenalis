{% extends "/layouts/doc.html" %}
{% macro Title string %}HTTP POST (Destination){% end %}
{% Article %}

# HTTP POST (Destination)

The destination connector HTTP POST allows you to export unified users (i.e., users consolidated through identity resolution) in various file formats, such as CSV or Excel, and send them directly to HTTP endpoints via POST. The receiving server is responsible for processing and saving the files.

Once a destination connection HTTP POST is configured, you can easily customize the file generation. You can configure multiple files for export by selecting the file format, applying filters to determine which users to include, specifying the target endpoint, and setting a schedule for how frequently each export should occur.

### On this page

* [Add destination connection HTTP POST](#add-destination-connection-http-post)

### Add destination connection HTTP POST

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **HTTP POST**; you can use the search bar at the top or filter by category.
4. Click on the **HTTP POST** connector. A panel will open on the right.
5. Click on **Add destination...**. The `Add destination connection for HTTP POST` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **Host** field, enter the host where the file or files you wish to write are located.
8. In the **Port** field, enter the port.
9. (Optional) In the **Headers** fields,  specify any headers that should be included in the request for writing the files.
10. Click **Add**.

Once the destination connection HTTP POST is added, the **Actions** page will be displayed. Here, you can add an action for each file you want to generate using the newly added destination connection HTTP POST. Configure each action with the desired settings for file format, user filters, endpoint, and scheduling.

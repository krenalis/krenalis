# HTTP data source

The HTTP data source allows you to import users from files accessed via HTTP endpoints. You can unify this data as users within Meergo by importing files in various formats, such as CSV, Excel, and others.

Once the HTTP data source is configured, you can easily customize how the data is read and processed.

### On this page

* [Add an HTTP data source](#add-an-http-data-source)

### Add an HTTP data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **HTTP** source; you can use the search bar at the top to help you.
4. Next to the **HTTP** source, click the **+** icon.
5. On the `Add HTTP source connection` page, in the **Name** field, enter a name for the source to easily recognize it later.
6. In the `Host` field, enter the host where the file or files you wish to read are located.
7. In the `Port` field, enter the port.
8. Optional: In the `Headers` fields,  specify any headers that should be included in the request for reading the files. 
9. Click **Add**.

Once the HTTP data source is added, the **Actions** page will be displayed. Here, you can add an action for each file to be read using the newly added HTTP data source. Configure each action with the desired settings for file format, filters for user data, and any additional processing requirements.

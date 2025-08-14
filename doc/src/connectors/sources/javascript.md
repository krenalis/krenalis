{% extends "/layouts/doc.html" %}
{% macro Title string %}JavaScript data source{% end %}
{% Article %}

# JavaScript data source

The **JavaScript** data source allows you to receive real-time events, including user information, from a website to a web application. The received events can be:

* Sent to destinations, particularly applications that can receive events.
* Stored in the workspace's data warehouse.
* Extracted to identify users, both recognized and anonymous, for unification in the workspace's data warehouse.

The JavaScript data source requires the use of the [JavaScript SDK](/developers/javascript-sdk) from Meergo to send events such as page views, clicks, and information about anonymous and authenticated users (for example, users who have logged in). In most cases, you only need to include the snippet in your site's pages. You can copy the snippet from the **Settings > Snippet** page of the JavaScript data source.

> The [JavaScript SDK](/developers/javascript-sdk) is an open-source JavaScript library licensed under MIT, compatible with both Segment's Analytics.js and RudderStack's JavaScript SDK.

### On this page

* [Add a JavaScript data source](#add-a-javascript-data-source)
* [Add the JavaScript snippet to website](#add-the-javascript-snippet-to-website)
* [Import events into the workspace's data warehouse](#import-events-into-the-workspaces-data-warehouse)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Add a JavaScript data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **JavaScript** source; you can use the search bar at the top or filter by category.
4. Click on the **JavaScript** connector. A panel will open on the right with information about **JavaScript**.
5. Click on **Add source**. The `Add JavaScript source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later. This could be the name of the site, for example.
7. (Optional) From the **Strategy** menu, select a strategy. You can change it later if needed.
8. (Optional) In the **Host** field, enter the domain of the site for tracking events.
9. Click **Add**.

Once the JavaScript data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the events received from this source.

### Add the JavaScript snippet to website

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click on the JavaScript data source.
3. Click on **Settings > Snippet**. The **Snippet** page will open.
4. Click **Copy** to copy the snippet.
5. Add the snippet to your site's pages between the **<head>** and **</head>** tags.

> Refer to the [JavaScript SDK](/developers/javascript-sdk) documentation for more details and other ways to use the SDK.

### Import events into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the JavaScript data source from which you want to import the events.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import events**.
5. Click **Add** to add the action.

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the JavaScript data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. Click **Import users**.
5. Click **Add** to add the action.

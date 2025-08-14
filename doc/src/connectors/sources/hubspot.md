{% extends "/layouts/doc.html" %}
{% macro Title string %}HubSpot data source{% end %}
{% Article %}

# HubSpot data source

The **HubSpot** data source allows you to read contacts from HubSpot and then unify them as users in Meergo.

HubSpot is a cloud application that offers tools for customer relationship management (CRM), marketing, and sales. It helps businesses organize contacts, automate marketing campaigns, and improve customer support.

### On this page

- [Add a HubSpot data source](#add-a-hubspot-data-source)
- [Import contacts into the workspace's data warehouse](#import-contacts-into-the-workspaces-data-warehouse)

### Add a HubSpot data source

Before you can add a HubSpot data source, you need to create an app in HubSpot and configure the Meergo settings file with OAuth credentials:

1. Create a <a href="https://developers.hubspot.com/" target="_blank">HubSpot developer account</a> or log in to an existing developer account. Note that this developer account will not be linked as a data source; you will specify that later.
2. Open the **Apps** page.
3. Click on **Create app**.
4. (Optional) Fill in the **Public app name** and **Description** fields to help you recognize the app later. The app does not need to be published on the HubSpot Marketplace, so it can remain private.
5. Click on the **Auth** tab.
6. Click **Add new scope** and add the scope **crm.objects.contacts.read**. If you also intend to use this app for a HubSpot data destination, add the **crm.objects.contacts.write** scope as well.
7. Click **Create app** and confirm the creation.
8. Copy the client ID and the client secret.
9. Set the `MEERGO_OAUTH_HUBSPOT_CLIENT_ID` and `MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET` environment variables  with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the `.env` file in the same directory of the Meergo executable.
10. Restart the Meergo server.

Now proceed to add a HubSpot data source:

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **HubSpot** source; you can use the search bar at the top or filter by category.
4. Click on the **HubSpot** connector. A panel will open on the right with information about **HubSpot**.
5. Click on **Add source**.
6. Follow the instructions provided by HubSpot to authorize access to your account to read contacts. Once finished, the `Add HubSpot source connection` page will appear.
7. In the **Name** field, enter a name for the source to easily recognize it later.
8. Click **Add**.

Once the HubSpot data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the contacts read from HubSpot.

### Import contacts into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the HubSpot data source from which you want to import the contacts.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To import only specific contacts, apply a [filter](/filters) to refine your selection.
5. (Optional) To import only updated contacts (i.e., those modified since the last import), select the **Run incremental import** option.
6. Define the mapping or use a transformation function to convert the contacts from MailChimp into users in your workspace's data warehouse.
7. Click **Add**.

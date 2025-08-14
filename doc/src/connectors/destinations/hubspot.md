{% extends "/layouts/doc.html" %}
{% macro Title string %}HubSpot data destination{% end %}
{% Article %}

# HubSpot data destination

The **HubSpot** data destination allows you to add and update unified Meergo users in HubSpot as contacts.

HubSpot is a cloud application that offers tools for customer relationship management (CRM), marketing, and sales. It helps businesses organize contacts, automate marketing campaigns, and improve customer support.

### On this page

- [Add a HubSpot data destination](#add-a-hubspot-data-destination)
- [Export users as HubSpot contacts](#export-users-as-hubspot-contacts)

### Add a HubSpot data destination

Before you can add a HubSpot data destination, you need to create an app in HubSpot and configure the Meergo settings file with OAuth credentials:

1. Create a <a href="https://developers.hubspot.com/" target="_blank">HubSpot developer account</a> or log in to an existing developer account. Note that this developer account will not be linked as a data destination; you will specify that later.
2. Open the **Apps** page.
3. Click on **Create app**.
4. (Optional) Fill in the **Public app name** and **Description** fields to help you recognize the app later. The app does not need to be published on the HubSpot Marketplace, so it can remain private.
5. Click on the **Auth** tab.
6. Click **Add new scope** and add the scopes **crm.objects.contacts.read** and **crm.objects.contacts.write**.
7. Click **Create app** and confirm the creation.
8. Copy the client ID and the client secret.
9. Set the `MEERGO_OAUTH_HUBSPOT_CLIENT_ID` and `MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET` environment variables  with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the `.env` file in the same directory of the Meergo executable.
10. Restart the Meergo server.

Now proceed to add a HubSpot data destination:

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destination**.
3. Search for the **HubSpot** destination; you can use the search bar at the top or filter by category.
4. Click on the **HubSpot** connector. A panel will open on the right with information about **HubSpot**.
5. Click on **Add destination**.
6. Follow the instructions provided by HubSpot to authorize access to your account for writing contacts. Once finished, the `Add HubSpot destination connection` page will appear.
7. In the **Name** field, enter a name for the destination to easily recognize it later.
8. Click **Add**.

Once the HubSpot data destination is added, the **Actions** page will be displayed, indicating the actions required to add and update contacts in HubSpot.

### Export users as HubSpot contacts

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the HubSpot data destination where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To export only specific users, apply a [filter](/filters) to refine your selection.
5. Select the matching properties that define how users in your workspace correspond to HubSpot contacts.
6. Choose what can be done with the users: **Create and update**, **Create only**, or **Update only**.
7. (Optional) If multiple HubSpot contacts match a single user in Meergo, choose whether to update them anyway or skip the update.
8. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse to HubSpot contacts.
9. Click **Add**.

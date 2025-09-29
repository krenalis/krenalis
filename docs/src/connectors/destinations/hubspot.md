{% extends "/layouts/doc.html" %}
{% macro Title string %}HubSpot (Destination){% end %}
{% Article %}

# HubSpot (Destination)

The destination connector for HubSpot allows you to add and update unified Meergo users in HubSpot as contacts.

HubSpot is a cloud application that offers tools for customer relationship management (CRM), marketing, and sales. It helps businesses organize contacts, automate marketing campaigns, and improve customer support.

### On this page

- [Add destination connection for HubSpot](#add-destination-connection-for-hubspot)
- [Export users as HubSpot contacts](#export-users-as-hubspot-contacts)

### Add destination connection for HubSpot

Before you can add destination connection for HubSpot, you need to create an app in HubSpot and configure the Meergo settings file with OAuth credentials. You only need to create the HubSpot app once, even if you add, remove, or re-add data destinations.

1. Log in to your <a href="https://www.hubspot.com/" target="_blank">HubSpot</a> account.
2. In the left sidebar, click the last item **Development** (the name may vary depending on the language of your account).
3. Click on **Legacy Apps**.
4. Click on **Create**.
5. Click on **Public** (the app will remain private and does not need to be made public).
6. Fill in the **Public app name** and **Description** fields to help you recognize the app later.
7. Click on the **Auth** tab.
8. Under **Redirect URLs**, enter the (external) URL of Meergo with the `/admin/oauth/authorize` path.
   * For example: `https://example.com/admin/oauth/authorize`. 
   * If your Meergo external URL is configured as `127.0.0.1` (the default), replace it with `localhost` when entering the value in HubSpot (e.g., use `http://localhost:2022/admin/oauth/authorize` instead of `http://127.0.0.1:2022/admin/oauth/authorize`).
9. Click **Add new scope**.
10. Select the following scopes:
    * `crm.objects.contacts.read` - leave as **Required**
    * `crm.objects.contacts.write` - mark as **Conditionally required**
    * `crm.schemas.contacts.read` - leave as **Required**
11. Click **Update**.
12. At the bottom left, click **Create app**. The app will be created and its configuration page will open.
13. Click on the **Auth** tab again.
14. Copy the Client ID and the Client secret.
15. Set the `MEERGO_OAUTH_HUBSPOT_CLIENT_ID` and `MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET` environment variables with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the `.env` file in the same directory of the Meergo executable.
16. Restart the Meergo server to load the new environment variables.

Now proceed to add destination connection for a:

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **HubSpot**; you can use the search bar at the top or filter by category.
4. Click on the connector for **HubSpot**. A panel will open on the right.
5. Click on **Add destination...**.
6. Follow the instructions provided by HubSpot to authorize access to your account for writing contacts. Once finished, the `Add destination connection for HubSpot` page will appear.
7. In the **Name** field, enter a name for the destination to easily recognize it later.
8. Click **Add**.

Once the destination connection for HubSpot is added, the **Actions** page will be displayed, indicating the actions required to add and update contacts in HubSpot.

### Export users as HubSpot contacts

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the destination connection for HubSpot where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To export only specific users, apply a [filter](/filters) to refine your selection.
5. Select the matching properties that define how users in your workspace correspond to HubSpot contacts.
6. Choose what can be done with the users: **Create and update**, **Create only**, or **Update only**.
7. (Optional) If multiple HubSpot contacts match a single user in Meergo, choose whether to update them anyway or skip the update.
8. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse to HubSpot contacts.
9. Click **Add**.

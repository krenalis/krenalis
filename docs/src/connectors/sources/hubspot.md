{% extends "/layouts/doc.html" %}
{% macro Title string %}HubSpot (Source){% end %}
{% Article %}

# HubSpot (Source)

The source connector for HubSpot allows you to read contacts from HubSpot and then unify them as users in Meergo.

HubSpot is a cloud application that offers tools for customer relationship management (CRM), marketing, and sales. It helps businesses organize contacts, automate marketing campaigns, and improve customer support.

### On this page

- [Add source connection for HubSpot](#add-source-connection-for-hubspot)
- [Import contacts into the workspace's data warehouse](#import-contacts-into-the-workspaces-data-warehouse)

### Add source connection for HubSpot

Before you can add source connection for HubSpot, you need to create an app in HubSpot and configure the Meergo settings file with OAuth credentials. You only need to create the HubSpot app once, even if you add, remove, or re-add data sources.

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
9.  Click **Add new scope**.
10. Select the following scopes:
    * `crm.objects.contacts.read` - leave as **Required**
    * `crm.schemas.contacts.read` - leave as **Required**
11. Click **Update**.
12. At the bottom left, click **Create app**. The app will be created and its configuration page will open.
13. Click on the **Auth** tab again. 
14. Copy the Client ID and the Client secret.
15. Set the `MEERGO_OAUTH_HUBSPOT_CLIENT_ID` and `MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET` environment variables with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the _.env_ file in the same directory of the Meergo executable.
16. Restart the Meergo server to load the new environment variables.

Now proceed to add source connection for a:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **HubSpot**; you can use the search bar at the top or filter by category.
4. Click on the connector for **HubSpot**. A panel will open on the right.
5. Click on **Add source...**.
6. Follow the instructions provided by HubSpot to authorize access to your account to read contacts. Once finished, the `Add source connection for HubSpot` page will appear.
7. In the **Name** field, enter a name for the source to easily recognize it later.
8. Click **Add**.

Once the source connection for HubSpot is added, the **Actions** page will be displayed. This page indicates what actions to perform with the contacts read from HubSpot.

### Import contacts into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the source connection for HubSpot from which you want to import the contacts.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To import only specific contacts, apply a [filter](/filters) to refine your selection.
5. (Optional) To import only updated contacts (i.e., those modified since the last import), select the **Run incremental import** option.
6. Define the mapping or use a transformation function to convert the contacts from HubSpot into users in your workspace's data warehouse.
7. Click **Add**.

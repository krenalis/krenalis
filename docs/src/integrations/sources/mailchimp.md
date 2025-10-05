{% extends "/layouts/doc.html" %}
{% macro Title string %}Mailchimp (Source){% end %}
{% Article %}

# Mailchimp (Source)

The source connector for Mailchimp allows you to read contacts from a Mailchimp audience and then unify them as users in Meergo.

Mailchimp is an email marketing platform that helps businesses design and send marketing emails, manage mailing lists, and automate campaigns. It also offers tools for audience segmentation, performance tracking, and basic CRM functionalities to improve customer engagement.

### On this page

* [Add source connection for Mailchimp](#add-source-connection-for-mailchimp)
* [Import contacts into the workspace's data warehouse](#import-contacts-into-the-workspaces-data-warehouse)

### Add source connection for Mailchimp

Before you can add source connection for Mailchimp, you need to create a private key in your Mailchimp account:

1. Log in to your <a href="https://mailchimp.com/" target="_blank">Mailchimp</a> account.
2. Click the account box at the top right and then click **Profile**.
3. On the **Profile** page, click **Extras > Registered apps**.
4. On the **Registered apps** page, click **Register An App**.
5. In the **App name** field, enter a name for the new app, for example, "Meergo data source."
6. In the **Redirect URI** field, enter the (external) URL of Meergo with the `/admin/oauth/authorize` path.
   * For example: `https://example.com/admin/oauth/authorize`.
   * If your Meergo external URL is configured as `localhost`, replace it with `127.0.0.1` when entering the value in Mailchimp (e.g., use `http://127.0.0.1:2022/admin/oauth/authorize` instead of `http://localhost:2022/admin/oauth/authorize`).
7. Fill in the remaining fields as desired.
8. Click **Create**.
9. Copy the **Client ID** and **Client Secret** field values.
10. Set the `MEERGO_OAUTH_MAILCHIMP_CLIENT_ID` and `MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET` environment variables  with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the _.env_ file in the same directory of the Meergo executable.
11. Restart the Meergo server.

Now proceed to add a source connection for Mailchimp:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Mailchimp**; you can use the search bar at the top or filter by category.
4. Click on the connection for **Mailchimp**. A panel will open on the right.
5. Follow the instructions provided by Mailchimp to authorize access to your account to read contacts. Once finished, the `Add source connection for Mailchimp` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Audience** field, select the Mailchimp audience from which to read the contacts. You can change it later.
8. Click **Add**.

Once the source connection for Mailchimp is added, the **Actions** page will be displayed. This page indicates what actions to perform with the contacts read from Mailchimp.

### Import contacts into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the source connection for Mailchimp from which you want to import the contacts.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To import only specific contacts, apply a [filter](/filters) to refine your selection.
5. (Optional) To import only updated contacts (i.e., those modified since the last import), select the **Run incremental import** option. 
6. Define the mapping or use a transformation function to convert the contacts from Mailchimp into users in your workspace's data warehouse.
7. Click **Add**.

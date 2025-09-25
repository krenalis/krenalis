{% extends "/layouts/doc.html" %}
{% macro Title string %}Mailchimp data destination{% end %}
{% Article %}

# Mailchimp data destination

The **Mailchimp** data destination allows you to add and update unified Meergo users in Mailchimp as contacts.

Mailchimp is an email marketing platform that helps businesses design and send marketing emails, manage mailing lists, and automate campaigns. It also offers tools for audience segmentation, performance tracking, and basic CRM functionalities to improve customer engagement.

### On this page

* [Add a Mailchimp data destination](#add-a-mailchimp-data-destination)
* [Export users as Mailchimp contacts](#export-users-as-mailchimp-contacts)

### Add a Mailchimp data destination

Before you can add a Mailchimp data destination, you need to create a private key in your Mailchimp account:

1. Log in to your <a href="https://mailchimp.com/" target="_blank">Mailchimp</a> account.
2. Click the account box at the top right and then click **Profile**.
3. On the **Profile** page, click **Extras > Registered apps**.
4. On the **Registered apps** page, click **Register An App**.
5. In the **App name** field, enter a name for the new app, for example, "Meergo data destination."
6. In the **Redirect URI** field, enter the (external) URL of Meergo with the `/admin/oauth/authorize` path.
   * For example: `https://example.com/admin/oauth/authorize`.
   * If your Meergo external URL is configured as `localhost`, replace it with `127.0.0.1` when entering the value in Mailchimp (e.g., use `http://127.0.0.1:2022/admin/oauth/authorize` instead of `http://localhost:2022/admin/oauth/authorize`).
7. Fill in the remaining fields as desired.
8. Click **Create**.
9. Copy the **Client ID** and **Client Secret** field values.
10. Set the `MEERGO_OAUTH_MAILCHIMP_CLIENT_ID` and `MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET` environment variables  with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the `.env` file in the same directory of the Meergo executable.
11. Restart the Meergo server.

Now proceed to add a Mailchimp data destination:

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Sources** page, click **Add a new destination ⊕** .
3. Search for the **Mailchimp** destination; you can use the search bar at the top or filter by category.
4. Click on the **Mailchimp** connector. A panel will open on the right with information about **Mailchimp**.
5. Click on **Add destination**.
6. Follow the instructions provided by Mailchimp to authorize access to your account to write contacts. Once finished, the `Add Mailchimp destination connection` page will appear.
7. In the **Name** field, enter a name for the destination to easily recognize it later.
8. In the **List** field, select the Mailchimp list to which write the contacts. You can change it later.
9. Click **Add**.

Once the Mailchimp data destination is added, the Actions page will be displayed, indicating the actions required to add and update contacts in Mailchimp.

### Export users as Mailchimp contacts

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the Mailchimp data destination where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To export only specific users, apply a [filter](/filters) to refine your selection.
5. Select the matching properties that define how users in your workspace correspond to Mailchimp contacts.
6. Choose what can be done with the users: **Create and update**, **Create only**, or **Update only**.
7. (Optional) If multiple Mailchimp contacts match a single user in Meergo, choose whether to update them anyway or skip the update.
8. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse to Mailchimp contacts.
9. Click **Add**.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Mailchimp data source{% end %}
{% Article %}

# Mailchimp data source

The **Mailchimp** data source allows you to read contacts from a Mailchimp audience and then unify them as users in Meergo.

Mailchimp is an email marketing platform that helps businesses design and send marketing emails, manage mailing lists, and automate campaigns. It also offers tools for audience segmentation, performance tracking, and basic CRM functionalities to improve customer engagement.

### On this page

* [Add a Mailchimp data source](#add-a-mailchimp-data-source)
* [Import contacts into the workspace's data warehouse](#import-contacts-into-the-workspaces-data-warehouse)

### Add a Mailchimp data source

Before you can add a Mailchimp data source, you need to create a private key in your Mailchimp account:

1. Log in to your <a href="https://mailchimp.com/" target="_blank">Mailchimp</a> account.
2. Click the account box at the top right and then click **Profile**.
3. On the **Profile** page, click **Extras > Registered apps**.
4. On the **Registered apps** page, click **Register An App**.
5. In the **App name** field, enter a name for the new app, for example, "Meergo data source."
6. In the **Redirect URI** field, enter “https://your-meergo-domain/admin/oauth/authorize” where “your-meergo-domain” is the domain, and port if present, of your Meergo domain. If you are running Meergo locally through Docker Compose, with the default configuration provided within the Meergo repository, you can use `http://127.0.0.1:9090/admin/oauth/authorize` as the Redirect URI.
7. Fill in the remaining fields as desired.
8. Click **Create**.
9. Copy the **Client ID** and **Client Secret** field values.
10. Set the `MEERGO_OAUTH_MAILCHIMP_CLIENT_ID` and `MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET` environment variables  with the values that you copied earlier, so that these are passed to the Meergo server. Alternatively, you can declare these environment variables in the `.env` file in the same directory of the Meergo executable.
11. Restart the Meergo server.

> Mailchimp does not allow authentication via the "localhost" domain, so if you are using "localhost" as the Meergo domain, you should use "127.0.0.1" instead, at least when adding a Mailchimp data source.

Now proceed to add a Mailchimp data source:

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Mailchimp** source; you can use the search bar at the top to assist you.
4. Next to the **Mailchimp** source, click the **+** icon.  A page will open on the Mailchimp site.
5. Follow the instructions provided by Mailchimp to authorize access to your account to read contacts. Once finished, you will return to the Meergo admin.
6. (Optional) In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Audience** field, select the Mailchimp audience from which to read the contacts. You can change it later.
8. Click **Add**.

Once the Mailchimp data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the contacts read from Mailchimp.

### Import contacts into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the Mailchimp data source from which you want to import the contacts.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To import only specific contacts, apply a [filter](/filters) to refine your selection.
5. (Optional) To import only updated contacts (i.e., those modified since the last import), select the **Run incremental import** option. 
6. Define the mapping or use a transformation function to convert the contacts from Mailchimp into users in your workspace's data warehouse.
7. Click **Add**.

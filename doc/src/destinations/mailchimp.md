# Mailchimp data destination

The **Mailchimp** data destination allows you to add and update unified Meergo users in Mailchimp as contacts.

Mailchimp is an email marketing platform that helps businesses design and send marketing emails, manage mailing lists, and automate campaigns. It also offers tools for audience segmentation, performance tracking, and basic CRM functionalities to improve customer engagement.

### On this page

* [Add a Mailchimp data destination](#add-a-mailchimp-data-destination)

### Add a Mailchimp data destination

Before you can add a Mailchimp data destination, you need to create a private key in your Mailchimp account:

1. Log in to your <a href="https://mailchimp.com/" target="_blank">Mailchimp</a> account.
2. Click the account box at the top right and then click **Profile**.
3. On the **Profile** page, click **Extra > Registered apps**.
4. On the **Registered apps** page, click **Register An App**.
5. In the **App name** field, enter a name for the new app, for example, "Meergo data destination."
6. In the **Redirect URI** field, enter “https://your-meergo-domain/ui/oauth/authorize” where “your-meergo-domain” is the domain, and port if present, of your Meergo domain.
7. Fill in the remaining fields as desired.
8. Click **Create**.
9. Copy the **Client ID** and **Client Secret** field values.
10. Open the **config.yaml** configuration file of Meergo.
11. Under **connectorsOAuth > Mailchimp**, enter the client ID and client secret you copied earlier.
12. Restart the Meergo server.

> Mailchimp does not allow authentication via the "localhost" domain, so if you are using "localhost" as the Meergo domain, you should use "127.0.0.1" instead, at least when adding a Mailchimp data destination.

Now proceed to add a Mailchimp data destination:

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Sources** page, click **Add New source**.
3. Search for the **Mailchimp** destination; you can use the search bar at the top to assist you.
4. Next to the **Mailchimp** destination, click the **+** icon. The destination addition page will open.
5. Optional: In the **Name** field, enter a name for the destination to easily recognize it later.
6. In the **List** field, select the Mailchimp list to which write the contacts. You can change it later.
7. Click **Add**.

Once the Mailchimp data destination is added, the Actions page will be displayed, indicating the actions required to add and update contacts in Mailchimp.
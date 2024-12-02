{% extends "/layouts/doc.html" %}
{% macro Title string %}Mailchimp data destination{% end %}
{% Article %}

# Mixpanel data destination

The **Mixpanel** data destination allows you to dispatch events to Mixpanel.

Mixpanel is an analytics platform that tracks user interactions across digital platforms. It provides tools for event tracking, funnel analysis, retention monitoring, and cohort analysis, helping businesses optimize user experiences and make data-driven decisions.

### On this page

* [Add a Mixpanel data destination](#add-a-mixpanel-data-destination)

### Add a Mixpanel data destination

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
4. Next to the **Mailchimp** destination, click the **+** icon. A page will open on the Mailchimp site.
5. Follow the instructions provided by Mailchimp to authorize access to your account to write contacts. Once finished, you will return to the Meergo admin.
6. Optional: In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **List** field, select the Mailchimp list to which write the contacts. You can change it later.
8. Click **Add**.

Once the Mailchimp data destination is added, the Actions page will be displayed, indicating the actions required to add and update contacts in Mailchimp.
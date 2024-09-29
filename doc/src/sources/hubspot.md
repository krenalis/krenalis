# HubSpot data source

The **HubSpot** data source allows you to read contacts from HubSpot and then unify them as users in Meergo.

HubSpot is a cloud application that offers tools for customer relationship management (CRM), marketing, and sales. It helps businesses organize contacts, automate marketing campaigns, and improve customer support.

### On this page

* [Add a HubSpot data source](#add-a-hubspot-data-source)

### Add a HubSpot data source

Before you can add a HubSpot data source, you need to create a private app in your HubSpot account and configure the Meergo settings file with OAuth credentials:

1. Create a <a href="https://developers.hubspot.com/" target="_blank">HubSpot developer account</a> or log in to an existing developer account.
2. Open the **Apps** page.
3. Click on **Create app**.
4. Optional: Fill in the **Public app name** and **Description** fields to help you recognize the app later. The app does not need to be published on the HubSpot marketplace, so it will never be public.
5. Click on the **Auth** tab.
6. Click **Add new scope** and add the scope **crm.objects.contacts.read**. If you also intend to use this app for a HubSpot data destination, add the **crm.objects.contacts.write** scope as well.
7. Click **Create app** and confirm the creation.
8. Copy the client ID and the client secret.
9. Open the **config.yaml** configuration file in Meergo.
10. Under **connectorsOAuth > HubSpot**, enter the client ID and client secret you copied earlier.
11. Restart the Meergo server.

Now proceed to add a HubSpot data source:

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **HubSpot** source; you can use the search bar at the top to help you.
4. Next to the **HubSpot** source, click the **+** icon. A page will open on the HubSpot site.
5. Follow the instructions provided by HubSpot to authorize access to your account to read contacts. Once finished, you will return to the Meergo admin.
6. Optional: In the **Name** field, enter a name for the source to easily recognize it later.
7. Click **Add**.

Once the HubSpot data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the contacts read from HubSpot.

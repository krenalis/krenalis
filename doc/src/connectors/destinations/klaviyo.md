{% extends "/layouts/doc.html" %}
{% macro Title string %}Klaviyo (Destination){% end %}
{% Article %}

# Klaviyo (Destination)

The destination connector for Klaviyo allows you to add and update unified Meergo users in Klaviyo as profiles and allows to send the received events to Klaviyo.

Klaviyo is an email marketing and automation platform that enables businesses to create personalized and targeted marketing campaigns. It is especially used by e-commerce companies to analyze customer data, segment audiences, and improve communication through personalized emails and messages.

### On this page

* [Add destination connection for Klaviyo](#add-destination-connection-for-klaviyo)
* [Export users as Klaviyo profiles](#export-users-as-klaviyo-profiles)
* [What to do if events don't show up](#what-to-do-if-events-dont-show-up)

### Add destination connection for Klaviyo

Before you can add destination connection for Klaviyo, you need to create a private key in your Klaviyo account:

1. Log in to your <a href="https://www.klaviyo.com/" target="_blank">Klaviyo</a> account.
2. Click the account box at the bottom left and then click **Settings**.
3. On the **Settings** page, click **Account > API Keys**.
4. On the **API Keys** page, click **Create Private API Key**.
5. In the **Private API Key Name** field, enter a name for the new key, for example, “Meergo data destination.”
6. With the *Custom Key* option enable:
    * Under the API scope **Events**, select **Full Access**.
    * Under the API scope **Profiles**, select **Full Access**.
7. Click **Create**.
8. Copy your **Private Key**.

Now proceed to add destination connection for a:

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **Klaviyo**; you can use the search bar at the top or filter by category.
4. Click on the connector for **Klaviyo**r. A panel will open on the right with information about **Klaviyo**.
5. Click on **Add destination**. The `Add Klaviyo destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **Your Private Key** field, enter the previously copied private key.
8. Click **Add**.

Once the destination connection for Klaviyo is added, the **Actions** page will be displayed, indicating the actions required to add and update profiles and send events to Klaviyo.

### Export users as Klaviyo profiles

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the destination connection for Klaviyo where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To export only specific users, apply a [filter](/filters) to refine your selection.
5. Select the matching properties that define how users in your workspace correspond to Klaviyo profiles.
6. Choose what can be done with the users: **Create and update**, **Create only**, or **Update only**.
7. (Optional) If multiple Klaviyo profiles match a single user in Meergo, choose whether to update them anyway or skip the update.
8. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse to Klaviyo profiles.
9. Click **Add**.

### What to do if events don't show up

If the connector for Klaviyo on Meergo sends events, but you don't see them in your Klaviyo dashboard, try the following checks:

* Make sure that the email address sent with events are not considered test email address by Klaviyo, as they would otherwise be silently discarded. Klaviyo considers test email addresses those ending with `@example.com`, `@test.com`, and others not specified. See [Klaviyo’s documentation on this](https://developers.klaviyo.com/en/reference/events_api_overview#create-event).

* See the [**Receiving a 202?**](https://developers.klaviyo.com/en/reference/events_api_overview#receiving-a-202) section of Klaviyo's documentation, which includes various scenarios where events are considered successfully received but are not displayed in Klaviyo.

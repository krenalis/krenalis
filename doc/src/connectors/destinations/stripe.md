{% extends "/layouts/doc.html" %}
{% macro Title string %}Stripe (Destination){% end %}
{% Article %}

# Stripe (Destination)

The destination connector for Stripe allows you to add and update unified Meergo users in Stripe as customers.

Stripe is a payment processing platform that enables businesses to accept online payments and manage transactions. It offers tools for handling credit card payments, subscriptions, and billing, as well as APIs for integrating payment solutions into websites and apps.

### On this page

* [Add destination connection for Stripe](#add-destination-connection-for-stripe)
* [Export users as Stripe customers](#export-users-as-stripe-customers)

### Add destination connection for Stripe

Before you can add destination connection for Stripe, you need to create an API key in your Stripe account:

1. Log in to your <a href="https://stripe.com/" target="_blank">Stripe</a> account.
2. Click **Developers**.
3. On the **Developers** page, click **API keys**.
4. On the **API keys** page, click **Create restricted key**.
5. If the Stripe interface asks you **How ​​will you use this API key?**, you can choose the second option, **Building your own integration**.
6. On the **Create restricted API key** page, in the **Key name** field, enter a name for the key, for example, "Meergo data destination."
7. In the row **Customers** and first column **PERMISSIONS**, click both **Read** and **Write**.
8. Click **Create key**.
9. In the screen showing the keys, copy the token of the key you just created.

Now proceed to add destination connection for a:

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **Stripe**; you can use the search bar at the top or filter by category.
4. Click on the connector for **Stripe**. A panel will open on the right.
5. Click on **Add destination...**. The `Add destination connection for Stripe` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the **API Key** field, enter the previously copied key.
8. Click **Add**.

Once the destination connection for Stripe is added, the **Actions** page will be displayed, indicating the actions required to add and update customers in Stripe.

### Export users as Stripe customers

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the destination connection for Stripe where you want to export the users.
3. If there are no actions, click **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To export only specific users, apply a [filter](/filters) to refine your selection.
5. Select the matching properties that define how users in your workspace correspond to Stripe customers.
6. Choose what can be done with the users: **Create and update**, **Create only**, or **Update only**.
7. (Optional) If multiple Stripe customers match a single user in Meergo, choose whether to update them anyway or skip the update.
8. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse to Stripe customers.
9. Click **Add**.

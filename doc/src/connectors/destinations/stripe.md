{% extends "/layouts/doc.html" %}
{% macro Title string %}Stripe data destination{% end %}
{% Article %}

# Stripe data destination

The **Stripe** data destination allows you to add and update unified Meergo users in Stripe as customers.

Stripe is a payment processing platform that enables businesses to accept online payments and manage transactions. It offers tools for handling credit card payments, subscriptions, and billing, as well as APIs for integrating payment solutions into websites and apps.

### On this page

* [Add a Stripe data destination](#add-a-stripe-data-destination)
* [Export users as Stripe customers](#export-users-as-stripe-customers)

### Add a Stripe data destination

Before you can add a Stripe data destination, you need to create an API key in your Stripe account:

1. Log in to your <a href="https://stripe.com/" target="_blank">Stripe</a> account.
2. Click **Developers**.
3. On the **Developers** page, click **API keys**.
4. On the **API keys** page, click **Create restricted key**.
5. On the **Create restricted API key** page, in the **Key name** field, enter a name for the key, for example, "Meergo data destination."
6. In the row **Customers** and first column **PERMISSIONS**, click both **Read** and **Write**.
7. Click **Create key**.
8. On the **Your new API key** dialog window, copy the key.
9. Click **Done**.

Now proceed to add a Stripe data destination:

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destination**.
3. Search for the **Stripe** destination; you can use the search bar at the top to assist you.
4. Next to the **Stripe** destination, click the **+** icon. The destination addition page will open.
5. (Optional) In the **Name** field, enter a name for the destination to easily recognize it later.
6. In the **API Key** field, enter the previously copied key.
7. Click **Add**.

Once the Stripe data destination is added, the **Actions** page will be displayed, indicating the actions required to add and update customers in Stripe.

### Export users as Stripe customers

1. From the Meergo admin, go to **Connections > Destinations**.
2. Click on the Stripe data destination where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To export only specific users, apply a [filter](/filters) to refine your selection.
5. Select the matching properties that define how users in your workspace correspond to Stripe customers.
6. Choose what can be done with the users: **Create and update**, **Create only**, or **Update only**.
7. (Optional) If multiple Stripe customers match a single user in Meergo, choose whether to update them anyway or skip the update.
8. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse to Stripe customers.
9. Click **Add**.

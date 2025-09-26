{% extends "/layouts/doc.html" %}
{% macro Title string %}Stripe (Source){% end %}
{% Article %}

# Stripe (Source)

The source connector for Stripe allows you to read customers from Stripe and then unify them as users in Meergo.

Stripe is a payment processing platform that enables businesses to accept online payments and manage transactions. It offers tools for handling credit card payments, subscriptions, and billing, as well as APIs for integrating payment solutions into websites and apps.

### On this page

* [Add source connection for Stripe](#add-source-connection-for-stripe)
* [Import customers into the workspace's data warehouse](#import-customers-into-the-workspaces-data-warehouse)

### Add source connection for Stripe

Before you can add source connection for Stripe, you need to create an API key in your Stripe account:

1. Log in to your <a href="https://stripe.com/" target="_blank">Stripe</a> account.
2. Click **Developers**.
3. On the **Developers** page, click **API keys**.
4. On the **API keys** page, click **Create restricted key**.
5. If the Stripe interface asks you **How ​​will you use this API key?**, you can choose the second option, **Building your own integration**.
6. On the **Create restricted API key** page, in the **Key name** field, enter a name for the key, for example, "Meergo data source."
7. In the row **Customers** and first column **PERMISSIONS**, click **Read**.
8. Click **Create key**.
9. In the screen showing the keys, copy the token of the key you just created.

Now proceed to add source connection for a:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **Stripe**; you can use the search bar at the top or filter by category.
4. Click on the connector for **Stripe**. A panel will open on the right.
5. Click on **Add source...**. The `Add source connection for Stripe` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **API Key** field, enter the previously copied key.
8. Click **Add**.

Once the source connection for Stripe is added, the **Actions** page will be displayed. This page indicates what actions to perform with the customers read from Stripe.

### Import customers into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the source connection for Stripe from which you want to import the customers.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To import only specific customers, apply a [filter](/filters) to refine your selection.
5. (Optional) To import only updated customers (i.e., those modified since the last import), select the **Run incremental import** option.
6. Define the mapping or use a transformation function to convert the customers from Stripe into users in your workspace's data warehouse.
7. Click **Add**.

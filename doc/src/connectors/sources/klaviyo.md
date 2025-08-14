{% extends "/layouts/doc.html" %}
{% macro Title string %}Klaviyo data source{% end %}
{% Article %}

# Klaviyo data source

The **Klaviyo** data source allows you to read profiles from Klaviyo and then unify them as users in Meergo.

Klaviyo is an email marketing and automation platform that enables businesses to create personalized and targeted marketing campaigns. It is especially used by e-commerce companies to analyze customer data, segment audiences, and improve communication through personalized emails and messages.

### On this page

* [Add a Klaviyo data source](#add-a-klaviyo-data-source)
* [Import profiles into the workspace's data warehouse](#import-profiles-into-the-workspaces-data-warehouse)

### Add a Klaviyo data source

Before you can add a Klaviyo data source, you need to create a private key in your Klaviyo account:

1. Log in to your <a href="https://www.klaviyo.com/" target="_blank">Klaviyo</a> account.
2. Click the account box at the bottom left and then click **Settings**.
3. On the **Settings** page, click **Account > API Keys**.
4. On the **API Keys** page, click **Create Private API Key**.
5. In the **Private API Key Name** field, enter a name for the new key, for example, “Meergo data source.”
6. With the *Custom Key* option enabled, under the API scope **Profiles**, select **Read Access**.
7. Click **Create**.
8. Copy your **Private Key**.

Now proceed to add a Klaviyo data source:

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Klaviyo** source; you can use the search bar at the top or filter by category.
4. Click on the **Klaviyo** connector. A panel will open on the right with information about **Klaviyo**.
5. Click on **Add source**. The `Add Klaviyo source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the **Your Private Key** field, enter the previously copied private key.
8. Click **Add**.

Once the Klaviyo data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the profiles read from Klaviyo.

### Import profiles into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the Klaviyo data source from which you want to import the profiles.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. (Optional) To import only specific profiles, apply a [filter](/filters) to refine your selection.
5. (Optional) To import only updated profiles (i.e., those modified since the last import), select the **Run incremental import** option.
6. Define the mapping or use a transformation function to convert the profiles from Klaviyo into users in your workspace's data warehouse.
7. Click **Add**.

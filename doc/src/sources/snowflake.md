{% extends "/layouts/doc.html" %}
{% macro Title string %}Snowflake data source{% end %}
{% Article %}

# Snowflake data source

The **Snowflake** data source allows you to read users from a Snowflake database and unify them as users within Meergo.

Snowflake is a cloud-based data warehousing platform for storing and analyzing large volumes of data. It provides a scalable architecture for real-time analytics, making it ideal for business intelligence applications.

### On this page

* [Add a Snowflake data source](#add-a-snowflake-data-source)

### Add a Snowflake data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **Snowflake** source; you can use the search bar at the top to help you.
4. Next to the **Snowflake** source, click the **+** icon.
5. On the `Add Snowflake source connection` page, in the **Name** field, enter a name for the source to easily recognize it later.
6. In the remaining fields, provide the necessary information to access your Snowflake data warehouse:
    * **Account**: The account ID of the Snowflake warehouse.
    * **Username**: A username with read access to the tables.
    * **Password**: The password for the user.
    * **Database**: The name of the database.
    * **Schema**: The name of the schema.
    * **Warehouse**: The name of the warehouse.
    * **Role**: The role to be assigned to the user.
7. Optional: Click **Test connection** to check if the inserted data is correct.
8. Click **Add**.

Once the Snowflake data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the users read from Snowflake.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Snowflake data destination{% end %}
{% Article %}

# Snowflake data destination

The **Snowflake** data destination allows you to write the unified users into a Snowflake table and keep it synchronized.

Snowflake is a cloud-based data warehousing platform for storing and analyzing large volumes of data. It provides a scalable architecture for real-time analytics, making it ideal for business intelligence applications.

### On this page

* [Add a Snowflake data destination](#add-a-snowflake-data-destination)

### Add a Snowflake data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destinations**.
3. Search for the **Snowflake** destination; you can use the search bar at the top to help you.
4. Next to the **Snowflake** destination, click the **+** icon.
5. On the `Add Snowflake destination connection` page, in the **Name** field, enter a name for the destination to easily recognize it later.
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

Once the Snowflake data destination is added, the **Actions** page will be displayed, indicating the actions required to update the table.
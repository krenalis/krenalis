{% extends "/layouts/doc.html" %}
{% macro Title string %}Snowflake data destination{% end %}
{% Article %}

# Snowflake data destination

The **Snowflake** data destination allows you to write the unified users into a Snowflake table and keep it synchronized.

Snowflake is a cloud-based data warehousing platform for storing and analyzing large volumes of data. It provides a scalable architecture for real-time analytics, making it ideal for business intelligence applications.

### On this page

* [Add a Snowflake data destination](#add-a-snowflake-data-destination)
* [Export users to Snowflake](#export-users-to-snowflake)

### Add a Snowflake data destination

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destinations**.
3. Search for the **Snowflake** destination; you can use the search bar at the top or filter by category.
4. Click on the **Snowflake** connector. A panel will open on the right with information about **Snowflake**.
5. Click on **Add destination**. The `Add Snowflake destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the remaining fields, provide the necessary information to access your Snowflake data warehouse:
    * **Account**: The account ID of the Snowflake warehouse.
    * **Username**: A username with read access to the tables.
    * **Password**: The password for the user.
    * **Database**: The name of the database.
    * **Schema**: The name of the schema.
    * **Warehouse**: The name of the warehouse.
    * **Role**: The role to be assigned to the user.
8. (Optional) Click **Test connection** to check if the inserted data is correct.
9. Click **Add**.

Once the Snowflake data destination is added, the **Actions** page will be displayed.

### Export users to Snowflake

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the Snowflake data destination where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the name of the Snowflake table where users should be added or updated.
5. Click **Confirm** to proceed.
6. Specify the key column. This column will be used to identify and update existing rows.
7. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse into Snowflake rows.
8. Click **Add**.

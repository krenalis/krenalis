{% extends "/layouts/doc.html" %}
{% macro Title string %}ClickHouse data destination{% end %}
{% Article %}

# ClickHouse data destination

The **ClickHouse** data destination allows you to write the unified users into a ClickHouse table and keep it synchronized.

ClickHouse is an open-source, column-oriented database optimized for real-time analytics. It efficiently processes large volumes of data, providing high performance for complex queries, making it an excellent choice for business intelligence applications.

### On this page

* [Add a ClickHouse data destination](#add-a-clickhouse-data-destination)
* [Export users to ClickHouse](#export-users-to-clickhouse)

### Add a ClickHouse data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add new destinations**.
3. Search for the **ClickHouse** destination; you can use the search bar at the top to help you.
4. Next to the **ClickHouse** destination, click the **+** icon.
5. On the `Add ClickHouse destination connection` page, in the **Name** field, enter a name for the destination to easily recognize it later.
6. In the remaining fields, provide the necessary information to access your ClickHouse instance:
    * **Host**: The name of the host.
    * **Port**: The port for the Native protocol (TCP). The default is 9000.
    * **Username**: A username with read and write access to the table.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
7. (Optional) Click **Test connection** to check if the inserted data is correct.
8. Click **Add**.

Once the ClickHouse data destination is added, the **Actions** page will be displayed, indicating the actions required to update the table.

### Export users to ClickHouse

1. From the Meergo admin, go to **Connections > Destinations**.
2. Click on the ClickHouse data destination where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the name of the ClickHouse table where users should be added or updated.
5. Click **Confirm** to proceed.
6. Specify the key column. This column will be used to identify and update existing rows.
7. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse into ClickHouse rows.
8. Click **Add**.
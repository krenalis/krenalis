# ClickHouse data source

The **ClickHouse** data source allows you to read users from a ClickHouse database and unify them as users within Meergo.

ClickHouse is an open-source, column-oriented database optimized for real-time analytics. It efficiently processes large volumes of data, providing high performance for complex queries, making it an excellent choice for business intelligence applications.

### On this page

* [Add a ClickHouse data source](#add-a-clickhouse-data-source)

### Add a ClickHouse data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **ClickHouse** source; you can use the search bar at the top to help you.
4. Next to the **ClickHouse** source, click the **+** icon.
5. On the `Add ClickHouse source connection` page, in the **Name** field, enter a name for the source to easily recognize it later.
6. In the remaining fields, provide the necessary information to access your ClickHouse instance:
   * **Host**: The name of the host.
   * **Port**: The port for the Native protocol (TCP). The default is 9000.
   * **Username**: A username with read access to the tables.
   * **Password**: The password for the user.
   * **Database name**: The name of the database.
7. Optional: Click **Test connection** to check if the inserted data is correct.
8. Click **Add**.

Once the ClickHouse data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the users read from ClickHouse.

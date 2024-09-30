# PostgreSQL data source

The **PostgreSQL** data source allows you to read users from a PostgreSQL database and unify them as users within Meergo.

PostgreSQL is an advanced open-source relational database system known for its robustness and scalability. It supports various data types and features like transaction management and full-text search.

### On this page

* [Add a PostgreSQL data source](#add-a-postgresql-data-source)

### Add a PostgreSQL data source

1. From the Meergo admin, go to **Connections > Sources**.
2. On the **Sources** page, click **Add new source**.
3. Search for the **PostgreSQL** source; you can use the search bar at the top to help you.
4. Next to the **PostgreSQL** source, click the **+** icon.
5. On the `Add PostgreSQL source connection` page, in the **Name** field, enter a name for the source to easily recognize it later.
6. In the remaining fields, provide the necessary information to access your PostgreSQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 5432.
    * **Username**: A username with read access to the tables.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
7. Optional: Click **Test connection** to check if the inserted data is correct.
8. Click **Add**.

Once the PostgreSQL data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the users read from PostgreSQL.

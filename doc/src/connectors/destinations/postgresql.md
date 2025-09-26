{% extends "/layouts/doc.html" %}
{% macro Title string %}PostgreSQL (Destination){% end %}
{% Article %}

# PostgreSQL (Destination)

The destination connector for PostgreSQL allows you to write the unified users into a PostgreSQL table and keep it synchronized.

PostgreSQL is an advanced open-source relational database system known for its robustness and scalability. It supports various data types and features like transaction management and full-text search.

### On this page

* [Add destination connection for PostgreSQL](#add-destination-connection-for-postgresql)
* [Export users to PostgreSQL](#export-users-to-postgresql)

### Add destination connection for PostgreSQL

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. On the **Destinations** page, click **Add a new destination ⊕**.
3. Search **PostgreSQL**; you can use the search bar at the top or filter by category.
4. Click on the connector for **PostgreSQL**. A panel will open on the right with information about **PostgreSQL**.
5. Click on **Add destination**. The `Add PostgreSQL destination connection` page will appear.
6. In the **Name** field, enter a name for the destination to easily recognize it later.
7. In the remaining fields, provide the necessary information to access your PostgreSQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 5432.
    * **Username**: A username with read and write access to the table.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
8. (Optional) Click **Test connection** to check if the inserted data is correct.
9. Click **Add**.

Once the destination connection for PostgreSQL is added, the **Actions** page will be displayed.

### Export users to PostgreSQL

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the destination connection PostgreSQL where you want to export the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the name of the PostgreSQL table where users should be added or updated.
5. Click **Confirm** to proceed.
6. Specify the key column. This column will be used to identify and update existing rows.
7. Define the mapping or use a transformation function to convert the users in your workspace's data warehouse into PostgreSQL rows.
8. Click **Add**.

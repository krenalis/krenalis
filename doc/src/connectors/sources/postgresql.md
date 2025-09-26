{% extends "/layouts/doc.html" %}
{% macro Title string %}PostgreSQL (Source){% end %}
{% Article %}

# PostgreSQL (Source)

The source connector for PostgreSQL allows you to read users from a PostgreSQL database and unify them as users within Meergo.

PostgreSQL is an advanced open-source relational database system known for its robustness and scalability. It supports various data types and features like transaction management and full-text search.

### On this page

* [Add source connection for PostgreSQL](#add-source-connection-for-postgresql)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)
* [Do incremental imports in query](#do-incremental-imports-in-query)

### Add source connection for PostgreSQL

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search **PostgreSQL**; you can use the search bar at the top or filter by category.
4. Click on the connector for **PostgreSQL**. A panel will open on the right with information about **PostgreSQL**.
5. Click on **Add source**. The `Add PostgreSQL source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the remaining fields, provide the necessary information to access your PostgreSQL instance:
    * **Host**: The name of the host.
    * **Port**: The port. The default is 5432.
    * **Username**: A username with read access to the tables.
    * **Password**: The password for the user.
    * **Database name**: The name of the database.
8. (Optional) Click **Test connection** to check if the inserted data is correct.
9. Click **Add**.

Once the source connection for PostgreSQL is added, the **Actions** page will be displayed. This page indicates what actions to perform with the users read from PostgreSQL.

### Import users into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the source connection for PostgreSQL from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the SQL query to select the PostgreSQL rows to be imported as users.
5. (Optional) Click **Preview** to see a preview of the query results.
6. Click **Confirm** to execute the query and continue.
7. Choose the identity column. This column must uniquely identify each user.
8. (Optional) Select a last change time column. This column should contain the timestamp of the user's most recent update.
9. (Optional) To import only updated users (i.e., those modified since the last import), select the **Run incremental import** option.
10. Define the mapping or use a transformation function to convert the users from PostgreSQL into users in your workspace's data warehouse.
11. Click **Add**.

### Do incremental imports in query

If the incremental import is enabled, you must use the `last_change_time` placeholder in the query, as shown in the following example:

```
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= ${last_change_time}
ORDER BY updated_at
```

The column used in the `WHERE` statement must be the same column selected as the last change time column in the action, and the query must return the rows ordered by this column in ascending order. For example, if the last change time column is a datetime column and the last change time is `2025-01-30 16:12:25.837`, the executed query would be:

```
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= '2025-01-30 16:12:25.837'
ORDER BY updated_at
```

If incremental import is not selected, `${last_change_time}` will be `NULL`. To make the query work whether or not incremental import is enabled, you can write it as follows:

```
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= ${last_change_time} OR ${last_change_time} IS NULL 
ORDER BY updated_at
```

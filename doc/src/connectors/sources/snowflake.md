{% extends "/layouts/doc.html" %}
{% macro Title string %}Snowflake data source{% end %}
{% Article %}

# Snowflake data source

The **Snowflake** data source allows you to read users from a Snowflake database and unify them as users within Meergo.

Snowflake is a cloud-based data warehousing platform for storing and analyzing large volumes of data. It provides a scalable architecture for real-time analytics, making it ideal for business intelligence applications.

### On this page

* [Add a Snowflake data source](#add-a-snowflake-data-source)
* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)
* [Do incremental imports in query](#do-incremental-imports-in-query)

### Add a Snowflake data source

1. From the Meergo Admin console, go to **Connections > Sources**.
2. On the **Sources** page, click **Add a new source ⊕** .
3. Search for the **Snowflake** source; you can use the search bar at the top or filter by category.
4. Click on the **Snowflake** connector. A panel will open on the right with information about **Snowflake**.
5. Click on **Add source**. The `Add Snowflake source connection` page will appear.
6. In the **Name** field, enter a name for the source to easily recognize it later.
7. In the remaining fields, provide the necessary information to access your Snowflake data warehouse:
    * **Account Identifier**: The account ID of the Snowflake warehouse.
    * **User Name**: A username with read access to the tables.
    * **Password**: The password for the user.
    * **Role**: The role to be assigned to the user.
    * **Database**: The name of the database.
    * **Schema**: The name of the schema.
    * **Warehouse**: The name of the warehouse.
8. (Optional) Click **Test connection** to check if the inserted data is correct.
9. Click **Add**.

Once the Snowflake data source is added, the **Actions** page will be displayed. This page indicates what actions to perform with the users read from Snowflake.

### Import users into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the Snowflake data source from which you want to import the users.
3. If there are no actions, click  **Add**, otherwise click **Add new action ⊕**.
4. Enter the SQL query to select the Snowflake rows to be imported as users.
5. (Optional) Click **Preview** to see a preview of the query results.
6. Click **Confirm** to execute the query and continue.
7. Choose the identity column. This column must uniquely identify each user.
8. (Optional) Select a last change time column. This column should contain the timestamp of the user's most recent update.
9. (Optional) To import only updated users (i.e., those modified since the last import), select the **Run incremental import** option.
10. Define the mapping or use a transformation function to convert the users from Snowflake into users in your workspace's data warehouse.
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

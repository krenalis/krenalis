{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Collect users from databases{% end %}
{% Article %}

# Collect users from databases
## Learn how to collect users from databases.

Meergo makes it easy to collect user data directly from your databases by writing your own queries, map it seamlessly into the Customer Model schema, and load it into your data warehouse for a unified, consistent view of your customers.

Meergo currently supports reading user data from **PostgreSQL**, **MySQL**, **ClickHouse**, and **Snowflake**.

## Steps

### 1. Connect a database

1. Go to the **Sources** page of your Meergo workspace.
2. Click on **Add a new source ⊕** and click on the card corresponding to your database.
3. Click on **Add source...**.

**Enter the connection details for the database.** The specified database name and schema define the default context for queries but do not restrict access to other databases or schemas the user is authorized to reference. It is recommended to use a read-only user account, ideally limited to the databases and tables you want to be accessible in queries.

<!-- tabs id:settings Settings -->

#### PostgreSQL

| Field         | Description                               |
|---------------|-------------------------------------------|
| Host          | Hostname or IP address of the database.   |
| Port          | Port used to connect (default is `5432`). |
| Username      | Username for authentication.              |
| Password      | Password associated with the username.    |
| Database name | Name of the database.                     |
| Schema        | Name of the schema within the database.   |

#### MySQL

| Field         | Description                               |
|---------------|-------------------------------------------|
| Host          | Hostname or IP address of the database.   |
| Port          | Port used to connect (default is `3306`). |
| Username      | Username for authentication.              |
| Password      | Password associated with the username.    |
| Database name | Name of the database.                     |

#### ClickHouse

| Field         | Description                               |
|---------------|-------------------------------------------|
| Host          | Hostname or IP address of the database.   |
| Port          | Port used to connect (default is `9000`). |
| Username      | Username for authentication.              |
| Password      | Password associated with the username.    |
| Database name | Name of the database.                     |

#### Snowflake

| Field              | Description                                                              |
|--------------------|--------------------------------------------------------------------------|
| Account Identifier | Account ID of the Snowflake warehouse in the form `<orgname>-<account>`. |
| User Name          | Username used to authenticate with Snowflake.                            |
| Password           | Password associated with the provided user name.                         |
| Role               | Role that will be used for accessing the data in Snowflake.              |
| Database           | Name of the database.                                                    |
| Schema             | Name of the schema within the database.                                  |
| Warehouse          | Name of the virtual warehouse to execute the query.                      |

<!-- end tabs -->

Before clicking **Add**, you can test the connection by clicking **Test connection**. The connection you just created is a source connection. You can access it later by clicking **Sources** section in the sidebar.

### 2. Add an action to import users

On the connection page, click on **Add action...**.

{{ Screenshot("Add action", "/docs/unification/collect-users/add-action.png", "", 264) }}

### 3. Query

Enter the query to run on your database to return the user data to import.

{{ Screenshot("Query editor", "/docs/unification/collect-users/query-editor.png", "", 2172) }}

<div data-tabs="settings">
  💡See the
  <span data-tab="postgresql">[PostgreSQL (Source) documentation](/sources/postgresql)</span>
  <span data-tab="mysql">[MySQL (Source) documentation](/sources/mysql)</span>
  <span data-tab="clickhouse">[ClickHouse (Source) documentation](/sources/clickhouse)</span>
  <span data-tab="snowflake">[Snowflake (Source) documentation](/sources/snowflake)</span>
  for more details on writing queries for incremental imports.
</div>

Click **Preview** to show a preview of the query with the first rows.

Click **Confirm** when you're satisfied with the query. You can still modify it later if needed.

### 4. Identity columns

Select the columns that uniquely identify a user in the result rows and, if available, the column containing the user's last updated date. This step ensures that only unique users are imported and incremental updates work correctly.

{{ Screenshot("Identity columns", "/docs/unification/collect-users/database-identity-columns.png", "", 2172) }}

Select **Run incremental import** if you want subsequent imports to include only the rows updated after the last import.

### 5. Transformation

The **Transformation** section allows you to harmonize the query schema with your Customer Model schema. You can choose between visual mapping or advanced transformations using JavaScript or Python.

Its purpose is to assign values retrieved from the query execution to the properties of the Customer Model. You have full control over which properties to map, assigning only those that matter to your business context while leaving others unassigned when no corresponding values exist.

{{ Screenshot("Visual mapping", "/docs/unification/collect-users/database-visual-mapping.png", "", 2168) }}

For complete details on how transformations work for harmonization, see how to [harmonize data](/unification/harmonization).

### 6. Save your changes

When you're done, click **Add** (or **Save** if you're editing an existing action).

For a single connection, you can also create multiple actions that import different user sets from the same database.

## Continue reading

### Process collected users

Clean and standardize user data using visual mapping, JavaScript, or Python transformations.

{{ render "../_includes/manage-users-cards.html" }}

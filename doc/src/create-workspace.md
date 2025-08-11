{% extends "/layouts/doc.html" %}
{% macro Title string %}Create a workspace{% end %}
{% Article %}

# Create a workspace

Meergo is a **warehouse-native Customer Data Platform (CDP)**. This means that your customer data remains stored directly in **your own data warehouse** — not within the application itself.

In Meergo each **workspace** is linked to its own data warehouse. Workspaces are fully isolated from each other and do not share any data.

When creating a new workspace, you will be prompted to provide the connection details for the data warehouse to be linked. This must be an **empty database**, with no existing tables.

> 🔒 While you can update the connection credentials at any time, it is **not possible to switch to a different data warehouse** once it has been associated with a workspace.

Meergo currently supports PostgreSQL and Snowflake as data warehouse.

<ul class="grid-list">
  <li><a href="#postgresql"> PostgreSQL</a></li>
  <li><a href="#snowflake"> Snowflake</a></li>
</ul><br> 

> 💡 Note that, when running Meergo through Docker Compose, is automatically available a PostgreSQL data warehouse that runs locally, ready to use, and that requires no configuration. For a quick tryout of Meergo, this is the recommended option.

## PostgreSQL

The following fields are required to connect a PostgreSQL data warehouse:

| Field           | Description                                                          |
|-----------------|----------------------------------------------------------------------|
| `Host`          | Hostname or IP address of the database.                              |
| `Port`          | Port used to connect (default is `5432`).                            |
| `Username`      | Username for authentication.                                         |
| `Password`      | Password associated with the username.                               |
| `Database name` | Name of the empty database to be used as the data warehouse.         |
| `Schema`        | Name of the schema within the database where tables will be created. |

## Snowflake

The following fields are required to connect a Snowflake data warehouse:

| Field       | Description                                                                   |
|-------------|-------------------------------------------------------------------------------|
| `Account`   | Account ID of the Snowflake warehouse in the form `<orgname>-<account_name>`. |
| `Port`      | Port used to connect (default is `443` for HTTPS).                            |
| `Username`  | Username used to authenticate with Snowflake.                                 |
| `Password`  | Password associated with the provided username.                               |
| `Database`  | Name of the database to be used as the data warehouse.                        |
| `Schema`    | Name of the schema within the database where tables will be created.          |
| `Warehouse` | name of the virtual warehouse to execute queries.                             |
| `Role`      | Role that will be used for accessing the data in Snowflake.                   |

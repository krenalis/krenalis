{% extends "/layouts/doc.html" %}
{% macro Title string %}Database setup{% end %}
{% Article %}

# Database setup

Meergo relies on PostgreSQL for its internal database. Note that this is not the same as the data warehouse you will configure later — this database is used exclusively for Meergo's own operational data and internal management.

To initialize it, execute the SQL script [`database/initialization/1 - postgres.sql`](https://github.com/meergo/meergo/blob/main/database/initialization/1%20-%20postgres.sql), which will create the required schema and tables based on your configuration.

Make sure the database connection settings in specified with the environment variables match your PostgreSQL instance.

## Database initialization steps

First, create a PostgreSQL database that will be used by Meergo. In this example, we'll name the database `meergo`:

```bash
psql postgres -c "CREATE DATABASE meergo"
```

Next, initialize the database with the tables needed for Meergo to run:

```bash
curl -sf https://raw.githubusercontent.com/meergo/meergo/refs/heads/main/database/initialization/1%20-%20postgres.sql\?token\=GHSAT0AAAAAACG2OK4HUAQL2QOFNT7Y4ZWG2EUV2QA | psql meergo
```

If the database initialization was successful, running the command:

```bash
psql meergo -c "\d"
```

You should see something like this:

```plain
                     List of relations
 Schema |            Name             |   Type   |  Owner   
--------+-----------------------------+----------+----------
 public | access_keys                 | table    | user
 public | accounts                    | table    | user
 public | accounts_id_seq             | sequence | user
 public | actions                     | table    | user
 public | actions_errors              | table    | user
 public | actions_executions          | table    | user
 public | actions_executions_id_seq   | sequence | user
 public | actions_id_seq              | sequence | user
...
 public | event_write_keys            | table    | user
 public | members                     | table    | user
 public | members_id_seq              | sequence | user
 public | metadata                    | table    | user
 public | notifications               | table    | user
 public | organizations               | table    | user
 public | organizations_id_seq        | sequence | user
 public | user_schema_primary_sources | table    | user
 public | workspaces                  | table    | user
```

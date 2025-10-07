{% extends "/layouts/doc.html" %}
{% macro Title string %}Database setup{% end %}
{% Article %}

# Database setup

Meergo uses PostgreSQL as its internal database. This is separate from the data warehouse you will configure later: the internal database stores only Meergo's operational and system data, and never contains any customer information.

### 1. Create a database

Create a PostgreSQL database that will be used by Meergo. In this example, we'll name the database _meergo_:

```bash
psql postgres -c "CREATE DATABASE meergo"
```

You can choose any name you like for the database; _meergo_ is just an example. You'll need to specify the name you choose later in the Meergo [configuration](../configuration).

### 2. Initialize the database

Initialize the database with the tables needed for Meergo to run:

```bash
curl -sf https://raw.githubusercontent.com/meergo/meergo/refs/heads/main/database/initialization/1%20-%20postgres.sql\?token\=GHSAT0AAAAAACG2OK4HUAQL2QOFNT7Y4ZWG2EUV2QA | psql meergo
```

To verify that the database was initialized successfully, run:

```bash
psql meergo -c "\d"
```

You should see something like this:

```plain
                     List of relations
 Schema |            Name           |   Type   |  Owner   
--------+---------------------------+----------+----------
 public | access_keys               | table    | user
 public | accounts                  | table    | user
 public | accounts_id_seq           | sequence | user
 public | actions                   | table    | user
 public | actions_errors            | table    | user
 public | actions_executions        | table    | user
 public | actions_executions_id_seq | sequence | user
 public | actions_id_seq            | sequence | user
...
 public | event_write_keys          | table    | user
 public | members                   | table    | user
 public | members_id_seq            | sequence | user
 public | metadata                  | table    | user
 public | notifications             | table    | user
 public | organizations             | table    | user
 public | organizations_id_seq      | sequence | user
 public | primary_sources           | table    | user
 public | workspaces                | table    | user
```

### Next step

Now you are ready to proceed with the [configuration](../configuration).

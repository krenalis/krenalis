{% extends "/layouts/doc.html" %}
{% macro Title string %}Database setup{% end %}
{% Article %}

# Database setup

Meergo relies on PostgreSQL for its internal database. Note that this is not the same as the data warehouse you will configure later — this database is used exclusively for Meergo's own operational data and internal management.

To initialize it, execute the SQL script [`database/initialization/1 - postgres.sql`](https://github.com/meergo/meergo/blob/main/database/initialization/1%20-%20postgres.sql), which will create the required schema and tables based on your configuration.

Make sure the database connection settings in specified with the environment variables match your PostgreSQL instance.

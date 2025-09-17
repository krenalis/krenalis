# database

## About the [initialization](./initialization/) directory

The [initialization](./initialization/) directory contains SQL scripts that initialize Meergo's PostgreSQL database.

It serves two purposes:

1. It includes SQL files, alphabetically ordered, which are executed in that order by the PostgreSQL Docker image. This behavior [is documented here](https://hub.docker.com/_/postgres#initialization-scripts).
2. It includes the [`1 - postgres.sql`](./initialization/1%20-%20postgres.sql) file, which can also be run manually on a PostgreSQL database to prepare it for use with Meergo;

It’s therefore important to carefully check (1) which files are in the [initialization](./initialization/) directory and (2) their alphabetical order, since this determines the execution order used by PostgreSQL in Docker.

## About the [`updates.sql`](updates.sql) file

The [`updates.sql`](updates.sql) file contains various update queries for the PostgreSQL database and is used internally by Meergo developers.

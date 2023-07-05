# chichi

<img src="https://static.wikia.nocookie.net/dragonballaf/images/c/c3/Chichi_foto8.jpg/revision/latest?cb=20120616090846&path-prefix=it" width=260px/>

- [Before commit](#before-commit)
  - [Troubleshoot tests](#troubleshoot-tests)
  - [Short tests during development](#short-tests-during-development)
  - [Troubleshooting](#troubleshooting)
- [How to execute Chichi](#how-to-execute-chichi)
  - [1. Install React and other dependencies](#1-install-react-and-other-dependencies)
  - [2. Configure and add certificates](#2-configure-and-add-certificates)
  - [3. Compile the server](#3-compile-the-server)
  - [4. Populate the database and the warehouse](#4-populate-the-database-and-the-warehouse)
  - [5. Connect the data warehouse](#5-connect-the-data-warehouse)
  - [6. Initialize the warehouse](#6-initialize-the-warehouse)
  - [7. Reload the schemas](#7-reload-the-schemas)
  - [8. Run and open the browser](#8-run-and-open-the-browser)
- [The `CHICHI_DEBUG_UI` environment variable](#the-chichi_debug_ui-environment-variable)
- [Expose on the Internet (optional)](#expose-on-the-internet-optional)
- [How test events](#how-test-events)
- [Interact with Chichi using `chichi-cli`](#interact-with-chichi-using-chichi-cli)
- [APIs](#apis)

## Before commit

From the root of this repository, run:

```
go run commit/commit.go
```

### Troubleshoot tests

If you encounter problems running tests, you can run:

```
go run commit/commit.go -pkg -v
```

to execute tests on every package printing verbose output. This should help
troubleshooting bad tests.

### Short tests during development

For short tests during development you can also use the command:

```
go run commit/commit.go -short
```

Note: don't use the option `-short` before committing because it runs only a
subset of the tests.

### Troubleshooting

If one of the commands above returned an error in the form:

```
pattern ./...: main module (chichi) does not contain package chichi/connectors/mysql
```

that may mean that the file `go.work` at the top of this repository has not been
updated to use a given module.

## How to execute Chichi

### 1. Install React and other dependencies

```
cd admin
npm install
```

It is recommended to add the `/admin/node_modules/` directory your local `.gitignore` file.

### 2. Configure and add certificates

Add a configuration file `app.ini` (see `app.example.ini`) in the same directory of
the `chichi` executable, as well as a `cert.pem` and `key.pem` certificate files.

### 3. Compile the server

Within the root of this repository execute:

```bash
go build -tags osusergo,netgo -trimpath
```

### 4. Populate the database and the warehouse

Populate the database with the queries in:

* [database/PostgreSQL.sql](database/PostgreSQL.sql)

and populate the warehouse with the queries in:

* [database/warehouses/postgresql.sql](database/warehouses/postgresql.sql)

or

* [database/warehouses/clickhouse.sql](database/warehouses/clickhouse.sql)

### 5. Connect the data warehouse

(note that these steps requires [the chichi-cli executable](#interact-with-chichi-using-chichi-cli) installed and available on your system)

Connect the data warehouse with:

```
$ chichi-cli connect-warehouse PostgreSQL ./postgresql.json
```

or 

```
$ chichi-cli connect-warehouse ClickHouse ./clickhouse.json
```

where `./postgresql.json` is a JSON file containing the information to access the PostgreSQL data warehouse, like:

```json
{
    "Host": "localhost",
    "Port": 5432,
    "Username": "user",
    "Password": "***********",
    "Database": "warehouse",
    "Schema": "public"
}
```

and `./clickhouse.json` is a JSON file containing the information to access the ClickHouse data warehouse, like:

```json
{
    "Host": "localhost",
    "Port": 9000,
    "Username": "user",
    "Password": "***********",
    "Database": "warehouse"
}
```

### 6. Initialize the warehouse

Initialize the warehouse with:

```
$ chichi-cli init-warehouse
```

### 7. Reload the schemas

Reload the schemas with:

```
$ chichi-cli reload-schemas
```

### 8. Run and open the browser

Launch the server executing `chichi` (or `chichi.exe` on Windows) and visit https://localhost:9090/admin/.

## The `CHICHI_DEBUG_UI` environment variable

Set the `CHICHI_DEBUG_UI` to `true` to display clearer error messages in the UI.

## Expose on the Internet (optional)

1. Install [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/)
2. Check that it is installed correctly: `cloudflared --version`
3. Run cloudflared: `cloudflared tunnel --url https://localhost:9090`
4. Make a note of the URL listed in the standard output (example:  https://xxxxxxx.trycloudflare.com)
5. Open the URL in a browser

## How test events

1. Add a Website source connection with host `localhost:9090`.
2. Add an action with type "Receive events" and enable it.
3. Add the content of the `trace-events-script/snippet.js` file into the `trace-events-script/website-for-testing/index.html`.
4. In the pasted code, replace `123456789` with the ID of the website connection.
5. Visit https://localhost:9090/trace-events-script/website-for-testing/.

## Interact with Chichi using `chichi-cli`

Refer to the [documentation of the chichi-cli tool](chichi-cli/README.md).

## APIs

See [apis](apis) for a documentation of the APIs.

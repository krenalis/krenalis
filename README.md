
# chichi

<img src="https://static.wikia.nocookie.net/dragonballaf/images/c/c3/Chichi_foto8.jpg/revision/latest?cb=20120616090846&path-prefix=it" width=260px/>

- [Before commit](#before-commit)
  - [Troubleshoot tests](#troubleshoot-tests)
  - [Short tests during development](#short-tests-during-development)
  - [Troubleshooting](#troubleshooting)
- [How to execute Chichi for development](#how-to-execute-chichi-for-development)
  - [1. Install React and other dependencies](#1-install-react-and-other-dependencies)
  - [2. Configure and add certificates](#2-configure-and-add-certificates)
  - [3. Build the assets](#3-build-the-assets)
  - [4. Compile the server command in dev mode](#4-compile-the-server-command-in-dev-mode)
  - [5. Populate the database](#5-populate-the-database)
  - [6. Connect the data warehouse](#6-connect-the-data-warehouse)
  - [7. Initialize the warehouse](#7-initialize-the-warehouse)
  - [8. Run and open the browser](#8-run-and-open-the-browser)
  - [9. Add properties to the `users` / `user_identities` schemas](#9-add-properties-to-the-users--user_identities-schemas)
- [Enable telemetry (optional)](#enable-telemetry-optional)
  - [For the first time](#for-the-first-time)
  - [If you already have configured and enabled telemetry](#if-you-already-have-configured-and-enabled-telemetry)
- [Expose on the Internet (optional)](#expose-on-the-internet-optional)
- [How to test events (and eventually import user identities)](#how-to-test-events-and-eventually-import-user-identities)
- [Interact with Chichi using `chichi-cli`](#interact-with-chichi-using-chichi-cli)
- [APIs](#apis)

## Before commit

From the root of this repository, run:

```
go run ./commit
```

### Troubleshoot tests

To troubleshoot bad tests, for example if they block indefinitely, you can run:

```
go run ./commit -pkg -v
```

to execute tests on every package printing verbose output. This should help
locating the problem.

### Short tests during development

For short tests during development you can also use the command:

```
go run ./commit -short
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

## How to execute Chichi for development

### 1. Install React and other dependencies

```
cd assets
npm install
```

### 2. Configure and add certificates

Add a configuration file `config.yaml` (see `config.example.yaml`) in the same directory of
the `chichi` executable, as well as a `cert.pem` and `key.pem` certificate files.

### 3. Build the assets

Within the root of this repository execute:

```bash
go generate ./cmd/chichi
```

Note that the assets will be embedded into the executable. However, in development mode, the assets are rebuilt for each invocation of the UI.

### 4. Compile the server command in dev mode

Within the root of this repository execute:

```bash 
go build -tags dev,osusergo,netgo -trimpath ./cmd/chichi
```

### 5. Populate the database

Populate the Chichi's database with the queries in [database/PostgreSQL.sql](database/PostgreSQL.sql).

### 6. Connect the data warehouse

(note that these steps requires [the chichi-cli executable](#interact-with-chichi-using-chichi-cli) installed and available on your system)

Connect the data warehouse with:

```
chichi-cli connect-warehouse PostgreSQL ./postgresql.json
```

or 

```
chichi-cli connect-warehouse ClickHouse ./clickhouse.json
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

### 7. Initialize the warehouse

Initialize the warehouse with:

```
chichi-cli init-warehouse
```

### 8. Run and open the browser

Launch the server command executing `./chichi` (or `./chichi.exe` on Windows) and visit https://localhost:9090/ui/.

### 9. Add properties to the `users` / `user_identities` schemas

Within the root of the repository, run:

```
chichi-cli change-user-schema ./test/example_user_schema.json
```

## Enable telemetry (optional)

### For the first time

1. see the documentation in the [telemetry directory](./telemetry) to learn how
   to install and run tools needed for telemetry.
2. update your local configuration file `config.yaml` according to the file
   [config.example.yaml](config.example.yaml).

### If you already have configured and enabled telemetry

From the directory `telemetry` of this repository, run the following commands:

To start the **OpenTelemetry Collector**:

```bash
otelcol --config confs/otelcol.yaml
```

To start **Jaeger**:

```bash
jaeger-all-in-one --collector.otlp.enabled=0
```

To start **Prometheus**:

```bash
prometheus --config.file=confs/prometheus.yml --web.listen-address="0.0.0.0:9095"
```

## Expose on the Internet (optional)

1. Install [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/)
2. Check that it is installed correctly: `cloudflared --version`
3. Run cloudflared: `cloudflared tunnel --url https://localhost:9090`
4. Make a note of the URL listed in the standard output (example:  https://xxxxxxx.trycloudflare.com)
5. Open the URL in a browser

## How to test events (and eventually import user identities)

1. Add a JavaScript source connection with host `localhost:9090`.
2. Add an action with type "Import events" (and/or an action "Import users", depending on what you want to test) and enable it.
3. Copy the snippet in "Settings > Snippet" of the connection.
4. Paste the snippet into your website between &lt;head&gt; and &lt;/head&gt;. You can also save the following HTML code into a file (let's suppose `javascript-sdk/mywebsite/index.html`):
   <details>
    <summary>Minimal HTML5 page</summary>

    <pre>
    &lt;!DOCTYPE html&gt;
    &lt;html lang=&quot;en&quot;&gt;
    &lt;head&gt;
        &lt;meta charset=&quot;utf-8&quot;&gt;
        &lt;title&gt;Test website&lt;/title&gt;
        &lt;!-- Paste the snippet here  --&gt;
    &lt;/head&gt;
    &lt;body&gt;
        &lt;p&gt;Test website&lt;/p&gt;
    &lt;/body&gt;
    &lt;/html&gt;
    </pre>

    </details>
5. Build the JavaScript SDK:

    ```sh
    cd javascript-sdk
    npm install
    deno task build
    ```
6. Visit the URL pointing to the HTML file, for example https://localhost:9090/javascript-sdk/mywebsite/.

## Interact with Chichi using `chichi-cli`

Refer to the [documentation of the chichi-cli tool](chichi-cli/README.md).

## APIs

See [apis](apis) for a documentation of the APIs.

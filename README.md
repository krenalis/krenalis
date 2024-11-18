
# meergo

- [Before commit](#before-commit)
  - [Run the UI tests](#run-the-ui-tests)
  - [Troubleshoot tests](#troubleshoot-tests)
  - [Short tests during development](#short-tests-during-development)
  - [Troubleshooting](#troubleshooting)
- [How to execute Meergo for development](#how-to-execute-meergo-for-development)
  - [1. Install React and other dependencies](#1-install-react-and-other-dependencies)
  - [2. Configure and add certificates](#2-configure-and-add-certificates)
  - [3. Build the assets](#3-build-the-assets)
  - [4. Compile the server command in dev mode](#4-compile-the-server-command-in-dev-mode)
  - [5. Populate the database](#5-populate-the-database)
  - [6. Create the workspace, connect a warehouse and initialize it](#6-create-the-workspace-connect-a-warehouse-and-initialize-it)
  - [7. Run and open the browser](#7-run-and-open-the-browser)
  - [8. Add properties to the user schema](#8-add-properties-to-the-user-schema)
- [Expose on the Internet (optional)](#expose-on-the-internet-optional)
- [How to test events (and eventually import user identities)](#how-to-test-events-and-eventually-import-user-identities)
- [Interact with Meergo using `meergo-cli`](#interact-with-meergo-using-meergo-cli)
- [Docker](#docker)
  - [Building Meergo Image](#building-meergo-image)
  - [Running Meergo within a Container](#running-meergo-within-a-container)

## Before commit

1. From the root of this repository, run:

```
go run ./commit
```

2. Set these environment variables, where each of them must point to a JSON file with the credentials of an **empty, i.e. initializable, data warehouse**.

* `MEERGO_TEST_PATH_WAREHOUSE_POSTGRESQL`
* `MEERGO_TEST_PATH_WAREHOUSE_SNOWFLAKE`

3. From the root of this repository, run:

```
go test -run ^TestWarehousesIdentityResolution$ github.com/meergo/meergo/core/datastore -count 1 -v
```

4. Set these environment variables, where each of them must point to a JSON file with the credentials of a data warehouse: 

* `MEERGO_TEST_PATH_POSTGRESQL`
* `MEERGO_TEST_PATH_SNOWFLAKE`

5. From the root of this repository, run:

```
go test -run ^Test_Merge$ github.com/meergo/meergo/warehouses/... -count 1 -v
```

### Run the UI tests

1. Start Meergo.

2. Create a new workspace.

3. Apply the test schema to the warehouse of the new workspace, replacing the placeholder `<YOUR_WORKSPACE_ID>` with the id of the newly created workspace:
    ```
    cd meergo-cli
    go build
    meergo-cli change-user-schema --yes ../test/example_user_schema.json -w <YOUR_WORKSPACE_ID>
    ```

5. In the directory `assets/tests`, add a file `test-config.json` and copy inside it the contents of the file `test-config-example.json`. Then, fill the various keys of the config file based on your local environment. For instance:

    ```json
    {
        "baseURL": "https://localhost:9090",
        "workspaceID": 1234567890,
        "dbHost": "127.0.0.1",
        "dbPort": 5432,
        "dbUsername": "postgres",
        "dbPassword": "foopass",
        "dbName": "foodb"
    }
    ```
    **NOTE**: the database credentials in the config file are used to test the creation of a "PostgreSQL" connection and the creation of actions associated with it. For this reason, any PostgreSQL database credentials can be provided, as long as: 
    - the database can be modified by the tests without causing loss of relevant information
    - the database is capable of satisfying the query `SELECT email, first_name, last_name FROM users WHERE ${last_change_time} LIMIT ${limit}` used in the test that adds the "Import Users" action 
    - the database allows the use of `users` as the table in the test that adds the "Export Users" action
    
    For this purpose, you can create a new database and execute the following query to populate it so that it is compatible with the tests:
    ```sql
    CREATE TABLE users (
        email VARCHAR(300),
        first_name VARCHAR(300),
        last_name VARCHAR(300)
    );
    ```

5. Navigate to the `assets` directory and install all the dependencies:

    ```
    cd assets
    npm install
    npx playwright install chromium
    ```
  
6. From the same `assets` directory, run the tests directly from the command line:

    ```
    npx playwright test
    ```
    or launch the Playwright UI to run and debug the tests visually:
    ```
    npx playwright test --ui
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
pattern ./...: main module (meergo) does not contain package meergo/connectors/mysql
```

that may mean that the file `go.work` at the top of this repository has not been
updated to use a given module.

## How to execute Meergo for development

### 1. Install React and other dependencies

```
cd assets
npm install
```

### 2. Configure and add certificates

Add a configuration file `config.yaml` (see `config.example.yaml`) in the same directory of
the `meergo` executable, as well as a `cert.pem` and `key.pem` certificate files.

### 3. Build the assets

Within the root of this repository execute:

```bash
go generate ./cmd/meergo
```

Note that the assets will be embedded into the executable. However, in development mode, the assets are rebuilt for each invocation of the UI.

### 4. Compile the server command in dev mode

Within the root of this repository execute:

```bash 
go build -tags dev,osusergo,netgo -trimpath ./cmd/meergo
```

### 5. Populate the database

Populate the Meergo's database with the queries in [database/PostgreSQL.sql](database/PostgreSQL.sql).

### 6. Create the workspace, connect a warehouse and initialize it

(note that these steps requires [the meergo-cli executable](#interact-with-meergo-using-meergo-cli) installed and available on your system)

Create the workspace, connect a PostgreSQL warehouse and initialize it with:

```
meergo-cli create-workspace PostgreSQL ./postgresql.json
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

> NOTE: It is possible to specify other data warehouses besides PostgreSQL. For more details, see `meergo-cli create-workspace --help`.

### 7. Run and open the browser

Launch the server command executing `./meergo` (or `./meergo.exe` on Windows) and visit https://localhost:9090/ui/.

### 8. Add properties to the user schema

Within the root of the repository, run:

```
meergo-cli change-user-schema ./test/example_user_schema.json -w <workspace ID>
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

## Interact with Meergo using `meergo-cli`

Refer to the [documentation of the meergo-cli tool](meergo-cli/README.md).

## Docker

### Building Meergo Image

1. Cd the root of this repository
   
2. Run:
   
    ```bash
    docker build -t meergo:dev . --progress=plain
    ```

### Running Meergo within a Container

**Note about the network**: the network is the same as the host system (`--net host`), so Meergo responds to and makes network requests to the same addresses it would if it were running outside of a container. This also includes the address of the PostgreSQL server that Meergo connects to and the addresses of the admin UI.

1. Cd the root of this repository
   
2. Run this command, replacing the paths on the left of `:` as needed (and leaving the paths on the right, `/bin/config.yaml`, etc... as they are):

    ```bash
    docker run -it \
        -v ./cmd/meergo/config.yaml:/bin/config.yaml \
        -v ./cmd/meergo/cert.pem:/bin/cert.pem \
        -v ./cmd/meergo/key.pem:/bin/key.pem \
        --net host \
        meergo:dev
    ```
3. Visit Meergo at the address specified in `config.yaml` (for example [https://localhost:9090/ui/](https://localhost:9090/ui/))

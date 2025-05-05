# Developers 🛠️

This file contains information useful to Meergo developers.

<h2>Table of contents</h2>

- [Before Pushing Commits to `main`](#before-pushing-commits-to-main)
- [How to run tests using GitHub Action](#how-to-run-tests-using-github-action)
- [Expose and see Meergo metrics](#expose-and-see-meergo-metrics)
- [Local Testing Cookbook](#local-testing-cookbook)
  - [Testing Snowflake](#testing-snowflake)
  - [Altering the tests configuration](#altering-the-tests-configuration)
- [How to execute Meergo for development](#how-to-execute-meergo-for-development)
  - [1. Install React and other dependencies](#1-install-react-and-other-dependencies)
  - [2. Configure and add certificates](#2-configure-and-add-certificates)
  - [3. Build the assets](#3-build-the-assets)
  - [4. Compile the server command in dev mode](#4-compile-the-server-command-in-dev-mode)
  - [5. Populate the database](#5-populate-the-database)
  - [7. Run and open the browser](#7-run-and-open-the-browser)
- [Expose on the Internet (optional)](#expose-on-the-internet-optional)
- [How to test events (and eventually import user identities)](#how-to-test-events-and-eventually-import-user-identities)
- [Docker](#docker)
  - [Building Meergo Image](#building-meergo-image)
  - [Running Meergo within a Container](#running-meergo-within-a-container)


## Before Pushing Commits to `main`

Before pushing commits to the `main` branch of Meergo, from the root of this repository, run:

```
go run ./commit
```

Note that this command does not print anything and may take 10 to 15 minutes to execute.

For various options and other ways to test, see the command documentation by running:

```
go run ./commit --help
```

**Tests Dependencies**. Refer to the file [.github/workflows/main.yml](.github/workflows/main.yml) for the list of dependencies required to run the tests and their respective supported versions.

## How to run tests using GitHub Action

1. Go to https://github.com/meergo/meergo/actions/workflows/main.yml
2. Click on the button "Run workflow"
3. Choose the branch on which you want to run the tests
4. Click on "Run workflow"

> ⌛ Note that this may take some time, even something on the order of about ten minutes.

## Expose and see Meergo metrics

1. **Enable metrics** by setting to `true` the `Enabled` constant in file `metrics/metrics.go`
2. Build and run Meergo

Now metrics are exposed at:

https://localhost:9090/debug/vars

## Local Testing Cookbook

Here are some guides to run various local tests and handle various situations that may arise. These are not mandatory but may be useful in some situations.

### Testing Snowflake

1. Set this environment variable, which must point to a JSON file with the credentials of an **empty, i.e. initializable**, Snowflake data warehouse:

    ```
    MEERGO_TEST_PATH_WAREHOUSE_SNOWFLAKE
    ```

2. From the root of this repository, run:

    ```
    go test -run ^TestWarehousesIdentityResolution$ github.com/meergo/meergo/core/datastore -count 1 -v
    ```

3. Set this environment variable, which must point to a JSON file with the credentials of a Snowflake data warehouse:

    ```
    MEERGO_TEST_PATH_SNOWFLAKE
    ```

4. From the root of this repository, run:

    ```
    go test -run ^Test_Merge$ github.com/meergo/meergo/warehouses/... -count 1 -v
    ```

### Altering the tests configuration

The tests inside `/test/` are already configured by default when the repository is clean, and they can be run as they are; however, in certain circumstances, it may become necessary to modify the test configuration, perhaps to meet a specific configuration of the system that runs them. Below are the documented environment variables that affect the tests:

| Variable                   | Description                                                                | Default                  |
|----------------------------|----------------------------------------------------------------------------|--------------------------|
| `MEERGO_TESTS_ADDR`        | The host and port on which Meergo is started                               | `127.0.0.1:9091`         |
| `MEERGO_TESTS_PYTHON_PATH` | The path to the Python executable for running the transformation functions | It depends on the system |

## How to execute Meergo for development

### 1. Install React and other dependencies

```
cd assets
npm install
```

### 2. Configure and add certificates

Set environment variables necessary to run Meergo (you can add a configuration file `.env` (see `meergo.example.env`) in the same directory of the `meergo` executable), as well as a `cert.pem` and `key.pem` certificate files.

### 3. Build the assets

Within the root of this repository execute:

```bash
go generate ./cmd/meergo
```

Note that the assets will be embedded into the executable. However, in development mode, the assets are rebuilt for each invocation of the admin.

### 4. Compile the server command in dev mode

Within the root of this repository execute:

```bash
go build -tags dev,osusergo,netgo -trimpath ./cmd/meergo
```

### 5. Populate the database

Populate the Meergo's database with the queries in [database/PostgreSQL.sql](database/PostgreSQL.sql).

### 7. Run and open the browser

Launch the server command executing `./meergo` (or `./meergo.exe` on Windows) and visit https://localhost:9090/admin/.

## Expose on the Internet (optional)

1. Install [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/)
2. Check that it is installed correctly: `cloudflared --version`
3. Run cloudflared: `cloudflared tunnel --url https://localhost:9090`
4. Make a note of the URL listed in the standard output (example: https://xxxxxxx.trycloudflare.com)
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

## Docker

The documentation on how to start Meergo inside Docker is available inside the [Getting started](doc/src/getting-started.md) documentation page, so it will not be repeated here.

Only more technical information is kept here.

### Building Meergo Image

1. Cd the root of this repository
2. Run:

    ```bash
    docker build -t meergo:dev . --progress=plain
    ```

### Running Meergo within a Container

**Note about the network**: the network is the same as the host system (`--net host`), so Meergo responds to and makes network requests to the same addresses it would if it were running outside of a container. This also includes the address of the PostgreSQL server that Meergo connects to and the addresses of the admin.

1. Cd the root of this repository
2. Run this command, replacing the paths for `--env-file` and on the left of `:` as needed (and leaving the paths on the right, `./cmd/meergo/cert.pem`, etc... as they are):

    ```bash
    docker run -it \
        --env-file ./cmd/meergo/.env \
        -v ./cmd/meergo/cert.pem:/bin/cert.pem \
        -v ./cmd/meergo/key.pem:/bin/key.pem \
        --net host \
        meergo:dev
    ```

3. Visit Meergo at the address shown on the console

# Developers 🛠️

This file contains information useful to Meergo developers.

<h2>Table of contents</h2>

- [Before Pushing Commits to `main`](#before-pushing-commits-to-main)
- [How to run tests using GitHub Action](#how-to-run-tests-using-github-action)
- [Telemetry](#telemetry)
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
- [Docker](#docker)
  - [Building Meergo Image](#building-meergo-image)
  - [Running Meergo within a Container](#running-meergo-within-a-container)


## Before Pushing Commits to `main`

> Before proceeding, **make sure you have all the necessary dependencies** installed locally. For the complete list of them, please refer to the file [.github/workflows/main.yml](.github/workflows/main.yml).

Before pushing commits to the `main` branch of Meergo, from the root of this repository, run:

```
go run ./commit
```

Note that this command does not print anything and may take 10 to 15 minutes to execute.

To run the tests without testing the connectors, run:

```
go run commit/commit.go --no-connector-tests
```

For various options and other ways to test, see the command documentation by running:

```
go run ./commit --help
```

## How to run tests using GitHub Action

1. Go to https://github.com/meergo/meergo/actions/workflows/main.yml
2. Click on the button "Run workflow"
3. Choose the branch on which you want to run the tests
4. Click on "Run workflow"

> ⌛ Note that this may take some time, even something on the order of about ten minutes.

## Telemetry

Telemetry can be enabled at various levels, depending on the value of the environment variable `MEERGO_TELEMETRY_LEVEL`:

| Value for `MEERGO_TELEMETRY_LEVEL` | Data sent to Sentry                    | Data sent to chichi.open2b.net |
|------------------------------------|----------------------------------------|--------------------------------|
| `none`                             | *(none)*                               | *(none)*                       |
| `errors`                           | Go server panics, Admin console errors | *(none)*                       |
| `stats`                            | *(none)*                               | State changes                  |
| `all` or empty string/not set      | Go server panics, Admin console errors | State changes                  |

The `MEERGO_TELEMETRY_LEVEL` environment variable is the only thing that can enable or disable telemetry. Anything else (how Meergo is compiled, build flags, availability of Debug IDs, etc...) does not impact the sending of data to Sentry and/or chichi.open2b.net.

Also, note that:

* **Personal data is never sent**. All error and statistics data sent to Sentry and chichi.open2b.net contain no personal information. For example, this is why only panic errors are eventually sent to Sentry, and not slog errors—because panics have been verified not to include personal data, while this cannot currently be guaranteed for slog.

* **Admin stack traces are available only under certain conditions**. The ability to see stack traces of Admin console errors in Sentry only exists if (1) Meergo is running in production mode (i.e., non-dev mode) and (2) the Meergo assets are unchanged from any commit in the repository, for which the GitHub Action sent source maps with Debug IDs to sentry. So, in any other case, the errors displayed on Sentry may not show a correct stack trace.

* **Admin errors are sent to Sentry through a server tunnel**. This avoids the problem of adblockers blocking data from being sent directly to Sentry. This does not cause any inconvenience to the user, as they can disable telemetry at any time through the environment variable.

* **About chichi.open2b.net**. chichi.open2b.net is an instance of Meergo that receives data sent from various Meergo instances, with the purpose of obtaining anonymous statistics on Meergo usage.

## Expose and see Meergo metrics

> Note that the concept of "Meergo metrics" is completely different from telemetry. "Meergo metrics" simply refers to an endpoint exposed by Meergo (disabled by default) to monitor certain internal software values.

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

Note that the assets will be embedded into the executable; however, in development mode (i.e. when using the `dev` flag compiling Meergo, see the section below), the assets are rebuilt for each invocation of the Admin console.

### 4. Compile the server command in dev mode

Within the root of this repository execute:

```bash
go build -tags dev,osusergo,netgo -trimpath ./cmd/meergo
```

(please note the `dev` flag, which is specific to Meergo)

### 5. Populate the database

Populate the Meergo's database with the queries in [database/initialization/1 - postgres.sql](database/initialization/1%20-%20postgres.sql).

### 7. Run and open the browser

Launch the server command executing `./meergo` (or `./meergo.exe` on Windows) and visit https://localhost:9090/admin/.

## Expose on the Internet (optional)

1. Install [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/)
2. Check that it is installed correctly: `cloudflared --version`
3. Run cloudflared: `cloudflared tunnel --url https://localhost:9090`
4. Make a note of the URL listed in the standard output (example: https://xxxxxxx.trycloudflare.com)
5. Open the URL in a browser

## Docker

The documentation on how to start Meergo inside Docker is available inside the [Getting started](doc/src/getting-started.md) documentation page, so it will not be repeated here.

Only more technical information is kept here.

### Building Meergo Image

1. Cd the root of this repository
2. Run:

    ```bash
    docker build -t gianlucamondini/prova-meergo:dev . --progress=plain
    ```

### Running Meergo within a Container

**Note about the network**: the network is the same as the host system (`--net host`), so Meergo responds to and makes network requests to the same addresses it would if it were running outside of a container. This also includes the address of the PostgreSQL server that Meergo connects to and the addresses of the Admin console.

1. Cd the root of this repository
2. Run this command, replacing the paths for `--env-file` and on the left of `:` as needed (and leaving the paths on the right, `./cmd/meergo/cert.pem`, etc... as they are):

    ```bash
    docker run -it \
        --env-file ./cmd/meergo/.env \
        -v ./cmd/meergo/cert.pem:/bin/cert.pem \
        -v ./cmd/meergo/key.pem:/bin/key.pem \
        --net host \
        gianlucamondini/prova-meergo:dev
    ```

3. Visit Meergo at the address shown on the console

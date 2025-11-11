# Developers 🛠️

This file contains information useful to **Meergo developers**.

## Before pushing commits to `main`

> Before proceeding, **make sure you have all the necessary dependencies** installed locally. For the complete list of them, please refer to the file [.github/workflows/go-run-commit.yml](.github/workflows/go-run-commit.yml).

Before pushing commits to the `main` branch of Meergo, from the root of this repository, run:

```
go run ./commit
```

⏱️ This may take 10 to 15 minutes to execute.

To run the tests **without testing the connectors**, run:

```
go run commit/commit.go --no-connector-tests
```

For various options and other ways to test, see the command documentation by running:

```
go run ./commit --help
```

## Run tests on GitHub (using GitHub Action)

1. Go to https://github.com/meergo/meergo/actions/workflows/go-run-commit.yml
2. Click on the button **Run workflow**
3. Choose the branch on which you want to run the tests
4. Click on **Run workflow**

⏱️ This may take 10 to 15 minutes to execute.

## Telemetry

👨🏻‍💻 The telemetry documentation kept here relates only to the more technical/implementational aspects.

### The MEERGO_TELEMETRY_LEVEL env variable

Telemetry can be enabled at various levels, depending on the value of the environment variable `MEERGO_TELEMETRY_LEVEL`:

| Value for `MEERGO_TELEMETRY_LEVEL` | Data sent to Sentry                    | Data sent to chichi.open2b.net |
|------------------------------------|----------------------------------------|--------------------------------|
| `none`                             | *(none)*                               | *(none)*                       |
| `errors`                           | Go server panics, Admin console errors | *(none)*                       |
| `stats`                            | *(none)*                               | State changes                  |
| `all` or empty string/not set      | Go server panics, Admin console errors | State changes                  |

The `MEERGO_TELEMETRY_LEVEL` environment variable is the only thing that can enable or disable telemetry. Anything else (how Meergo is compiled, build flags, availability of Debug IDs, etc...) does not impact the sending of data to Sentry and/or chichi.open2b.net.

### Principles and considerations

* **Personal data is never sent**. All error and statistics data sent to Sentry and chichi.open2b.net contain no personal information. For example, this is why only panic errors are eventually sent to Sentry, and not slog errors—because panics have been verified not to include personal data, while this cannot currently be guaranteed for slog.

* **Admin stack traces are available only under certain conditions**. The ability to see stack traces of Admin console errors in Sentry only exists if (1) Meergo is running in production mode (i.e., non-dev mode) and (2) the Meergo assets are unchanged from any commit in the repository, for which the GitHub Action sent source maps with Debug IDs to sentry. So, in any other case, the errors displayed on Sentry may not show a correct stack trace.

* **Admin errors are sent to Sentry through a server tunnel**. This avoids the problem of adblockers blocking data from being sent directly to Sentry. This does not cause any inconvenience to the user, as they can disable telemetry at any time through the environment variable.

* **About chichi.open2b.net**. chichi.open2b.net is an instance of Meergo that receives data sent from various Meergo instances, with the purpose of obtaining anonymous statistics on Meergo usage.

## Expose and see Meergo metrics

> ⚠️ Note that this type of metric is deprecated in favor of Prometheus metrics. See https://github.com/meergo/meergo/issues/1840.

> Note that the concept of "Meergo metrics" is completely different from telemetry. "Meergo metrics" simply refers to an endpoint exposed by Meergo (disabled by default) to monitor certain internal software values.

1. **Enable metrics** by setting to `true` the `Enabled` constant in file `metrics/metrics.go`
2. Build and run Meergo

Now metrics are exposed at:

https://localhost:2022/debug/vars

## Local Testing Cookbook

Here are some guides to run various local tests and handle various situations that may arise. These are not mandatory but may be useful in some situations.

### Testing Snowflake

1. Set this environment variable, which must contain the absolute path to a JSON file with the credentials of an **empty (i.e., initializable)** Snowflake data warehouse:

    ```
    MEERGO_TEST_PATH_WAREHOUSE_SNOWFLAKE
    ```

    <details style="margin: 1em 0;">
      <summary>Sample <code>MEERGO_TEST_PATH_WAREHOUSE_SNOWFLAKE</code> file</summary>

      ```json
      {
        "Account": "",
        "Username": "",
        "Password": "",
        "Role": "",
        "Database": "",
        "Schema": "PUBLIC",
        "Warehouse": ""
      }
      ```
    </details>

2. From the root of this repository, run:

    ```
    go test -run ^TestWarehousesIdentityResolution$ github.com/meergo/meergo/core/internal/datastore -count 1 -v
    ```

3. Set this environment variable, which must point to a JSON file with the credentials of a Snowflake data warehouse:

    ```
    MEERGO_TEST_PATH_SNOWFLAKE
    ```

    <details style="margin: 1em 0;">
      <summary>Sample <code>MEERGO_TEST_PATH_SNOWFLAKE</code> file</summary>

      ```json
      {
        "Account": "",
        "Username": "",
        "Password": "",
        "Role": "",
        "Database": "",
        "Schema": "PUBLIC",
        "Warehouse": ""
      }
      ```
    </details>

4. From the root of this repository, run:

    ```
    go test -run ^Test_Merge$ github.com/meergo/meergo/warehouses/... -count 1 -v
    ```

### Altering the tests configuration

The tests inside `/test/` are already configured by default when the repository is clean, and they can be run as they are; however, in certain circumstances, it may become necessary to modify the test configuration, perhaps to meet a specific configuration of the system that runs them. Below are the documented environment variables that affect the tests:

| Variable                   | Description                                                                | Default                  |
|----------------------------|----------------------------------------------------------------------------|--------------------------|
| `MEERGO_TESTS_ADDR`        | The host and port on which Meergo is started                               | `127.0.0.1:2023`         |
| `MEERGO_TESTS_PYTHON_PATH` | The path to the Python executable for running the transformation functions | It depends on the system |

## Docker

### Building Meergo Image

The Meergo Docker image is built using [the GitHub Action](https://github.com/meergo/meergo/actions/workflows/publish-docker-image.yml).

### Running Meergo within a Container

**🌐 Note about the network**: the network is the same as the host system (`--net host`), so Meergo responds to and makes network requests to the same addresses it would if it were running outside of a container. This also includes the address of the PostgreSQL server that Meergo connects to and the addresses of the Admin console.

1. Cd the root of this repository
2. Run this command, replacing the paths for `--env-file` and on the left of `:` as needed (and leaving the paths on the right, `./cmd/meergo/cert.pem`, etc... as they are):

    ```bash
    docker run -it \
        --env-file ./cmd/meergo/.env \
        -v ./cmd/meergo/cert.pem:/bin/cert.pem \
        -v ./cmd/meergo/key.pem:/bin/key.pem \
        --net host \
        meergocdp/meergo:v0.8.5
    ```

3. Visit Meergo at the address shown on the console

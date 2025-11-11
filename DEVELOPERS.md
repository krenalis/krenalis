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

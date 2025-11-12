# Developers 🛠️

This file contains information useful to **Meergo developers**.

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

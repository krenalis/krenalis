# Developers 🛠️

This file contains information useful to **Meergo developers**.

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

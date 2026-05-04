# snowflaketester/cmd

Script to **create a temporary Snowflake schema for manual testing** (eg. workspace creation, import, IR execution, etc.).

## Environment Variables

This script is configured via environment variables. See the documentation for the `test/snowflaketester` package.

For a quick reference, the env vars are:

- `KRENALIS_SNOWFLAKE_TESTER_ACCOUNT`
- `KRENALIS_SNOWFLAKE_TESTER_DATABASE`
- `KRENALIS_SNOWFLAKE_TESTER_PASSWORD`
- `KRENALIS_SNOWFLAKE_TESTER_ROLE`
- `KRENALIS_SNOWFLAKE_TESTER_USER`
- `KRENALIS_SNOWFLAKE_TESTER_WAREHOUSE`

## How to Run

In the directory of this README, simply run:

```
go run .
```

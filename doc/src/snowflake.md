# Snowflake

## Supported types

The table below provides a list of supported types in a Snowflake data warehouse along with their corresponding property types:

| Column Type     | Property Type  |
|-----------------|----------------|
| `NUMBER(p,s)`   | `Decimal(p,s)` |
| `FLOAT`         | `Float(64)`    |
| `VARCHAR`       | `Text`         |
| `BOOLEAN`       | `Boolean`      |
| `DATE`          | `Date`         |
| `TIME`          | `Time`         |
| `TIMESTAMP_NTZ` | `DateTime`     |
| `VARIANT`       | `JSON`         |
| `ARRAY`         | `Array(JSON)`  |

Alias types are also supported:

* `DECIMAL`, `DEC`, `NUMERIC`, `INT`, `INTEGER`, `BIGINT`, `SMALLINT`, `TINYINT`, `BYTEINT`, as aliases of `NUMBER`.

* `FLOAT4`, `FLOAT8`, `DOUBLE`, `DOUBLE PRECISION`, `REAL`, as aliases of `FLOAT`.

* `CHAR`, `CHARACTER`, `STRING`, `TEXT`, as aliases of `VARCHAR`.

* `DATETIME`, `TIMESTAMP`, as aliases of `TIMESTAMP_NTZ`.

## NULL columns

Properties of columns declared as `NULL` are nullable.

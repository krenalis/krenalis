# Snowflake Driver

## Supported Types

> This section **may be obsolete**. See [this issue](https://github.com/meergo/meergo/issues/575).

The table below provides a list of supported property types when using a Snowflake data warehouse, along with the corresponding column types that are generated on the data warehouse.

| Property Type  | Column Type     |
| -------------- | --------------- |
| `Decimal(p,s)` | `NUMBER(p,s)`   |
| `Float(64)`    | `FLOAT`         |
| `Text`         | `VARCHAR`       |
| `Boolean`      | `BOOLEAN`       |
| `Date`         | `DATE`          |
| `Time`         | `TIME`          |
| `DateTime`     | `TIMESTAMP_NTZ` |
| `JSON`         | `VARIANT`       |
| `Array(JSON)`  | `ARRAY`         |

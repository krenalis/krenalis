# PostgreSQL

## Supported types

The table below provides a list of supported types in a PostgreSQL data warehouse along with their corresponding property types:

| Column Type                   | Property Type     |
|-------------------------------|-------------------|
| `smallint`                    | `Int(16)`         |
| `integer`                     | `Int(32)`         |
| `bigint`                      | `Int(64)`         |
| `decimal(p,s)` [^1]           | `Decimal(p,s)`    |
| `numeric(p,s)` [^1]           | `Decimal(p,s)`    |
| `real`                        | `Float(32)`       |
| `double precision`            | `Float(64)`       |
| `smallserial`                 | `Int(16)`         |
| `serial`                      | `Int(32)`         |
| `bigserial`                   | `Int(64)`         |
| `varchar`                     | `Text`            |
| `char`                        | `Text`            |
| `bpchar`                      | `Text`            |
| `timestamp without time zone` | `DateTime`        |
| `date`                        | `Date`            |
| `time without time zone`      | `Time`            |
| _Enumerated types_            | `Text`            |
| `uuid`                        | `UUID`            |
| `json`                        | `JSON`            |
| `jsonb`                       | `JSON`            |
| `T[]` [^2]                    | `Array(T's type)` |

[^1] `decimal(p,s)` and `numeric(p,s)` are supported if `p` is in range [1, 76] and `s` is in range [0, 37].

[^2] `T[]` is supported if `T` is supported.

## Requirements

Columns may not have unique constraints, and non-hidden columns must not have check constraints. In such cases, an upsert operation may fail.

## NULL columns

Properties of columns declared as `NULL` are nullable.

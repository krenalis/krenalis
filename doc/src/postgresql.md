# PostgreSQL

## Data Types

The table below provides a list of supported property types when using a PostgreSQL data warehouse, along with the corresponding column types that are generated on the data warehouse.

| Property Type       | Column Type                   |
|---------------------|-------------------------------|
| `Bool`              | `boolean`                     |
| `Int(16)`           | `smallint`                    |
| `Int(32)`           | `integer`                     |
| `Int(64)`           | `bigint`                      |
| `Decimal(p,s)`      | `decimal(p,s)`                |
| `Float(32)`         | `real`                        |
| `Float(64)`         | `double precision`            |
| `Text`              | `varchar`                     |
| `DateTime`          | `timestamp without time zone` |
| `Date`              | `date`                        |
| `Time`              | `time without time zone`      |
| `UUID`              | `uuid`                        |
| `JSON`              | `jsonb`                       |
| `Array(T)` [^array] | `T[]`                         |

[^array]: where `T` is not `Array`

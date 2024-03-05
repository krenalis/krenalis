# PostgreSQL

## Supported types

The table below provides a list of supported property types when using a PostgreSQL data warehouse, along with the corresponding column types that are generated on the data warehouse.

| Property Type            | Column Type                   |
|--------------------------|-------------------------------|
| `Bool`                   | `boolean`                     |
| `Int(16)` [^1]           | `smallint`                    |
| `Int(32)` [^1]           | `integer`                     |
| `Int(64)` [^1]           | `bigint`                      |
| `Decimal(p,s)` [^1] [^2] | `decimal(p,s)`                |
| `Float(32)` [^1][^3]     | `real`                        |
| `Float(64)` [^1][^3]     | `double precision`            |
| `Text` [^4]              | `varchar`                     |
| `DateTime`               | `timestamp without time zone` |
| `Date`                   | `date`                        |
| `Time`                   | `time without time zone`      |
| `UUID`                   | `uuid`                        |
| `JSON`                   | `jsonb`                       |
| `Array(T)` [^5]          | `T[]`                         |

[^1]: Numeric types with limited ranges are not supported. For more details, see the issue [#578](https://github.com/open2b/chichi/issues/578).

[^2]: `Decimal(p,s)` is supported if `p` is in range [1, 76] and `s` is in range [0, 37].

[^3]: Only non-real `Float` types are supported, as Postgres floating-point types allow `Infinity`, `-Infinity` and `NaN`

[^4]: `Text` types with regexp, bytes length or values are not supported.

[^5]: `Array(T)` is supported if `T` is supported and if `T` is not array.

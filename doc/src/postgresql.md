# PostgreSQL

## Supported types

The table below provides a list of supported property types when using a PostgreSQL data warehouse, along with the corresponding column types that are generated on the data warehouse.

| Property Type                              | Column Type                   |
|--------------------------------------------|-------------------------------|
| `Bool`                                     | `boolean`                     |
| `Int(16)` [^numrange]                      | `smallint`                    |
| `Int(32)` [^numrange]                      | `integer`                     |
| `Int(64)` [^numrange]                      | `bigint`                      |
| `Decimal(p,s)` [^numrange] [^decimalrange] | `decimal(p,s)`                |
| `Float(32)` [^numrange] [^float]           | `real`                        |
| `Float(64)` [^numrange] [^float]           | `double precision`            |
| `Text` [^text]                             | `varchar`                     |
| `DateTime`                                 | `timestamp without time zone` |
| `Date`                                     | `date`                        |
| `Time`                                     | `time without time zone`      |
| `UUID`                                     | `uuid`                        |
| `JSON`                                     | `jsonb`                       |
| `Array(T)` [^array]                        | `T[]`                         |

[^numrange]: Numeric types with limited ranges are not supported. For more details, see the issue [#578](https://github.com/meergo/meergo/issues/578).

[^decimalrange]: `Decimal(p,s)` is supported if `p` is in range [1, 76] and `s` is in range [0, 37].

[^float]: Only non-real `Float` types are supported, as Postgres floating-point types allow `Infinity`, `-Infinity` and `NaN`

[^text]: `Text` types with regexp, bytes length or values are not supported.

[^array]: `Array(T)` is supported if `T` is supported and if `T` is not array.

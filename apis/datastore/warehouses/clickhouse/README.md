# ClickHouse Driver

> This page **may be obsolete**. See [this issue](https://github.com/meergo/meergo/issues/575).

## Supported types

The table below provides a list of supported types in a ClickHouse data warehouse along with their corresponding property types:

| Column Type              | Property Type  |
|--------------------------|----------------|
| `UInt8`                  | `Uint8`        |
| `UInt16`                 | `Uint16`       |
| `UInt32`                 | `Uint32`       |
| `UInt64`                 | `Uint64`       |
| `Int8`                   | `Int8`         |
| `Int16`                  | `Int16`        |
| `Int32`                  | `Int32`        |
| `Int64`                  | `Int64`        |
| `Float32`                | `Float32`      |
| `Float64`                | `Float64`      |
| `Decimal(p,s)` [^1]      | `Decimal(p,s)` |
| `String`                 | `String`       |
| `FixedString`            | `String`       |
| `Date`                   | `Date`         |
| `Date32`                 | `Int32`        |
| `DateTime`               | `DateTime`     |
| `DateTime64`             | `DateTime`     |
| `JSON`                   | `JSON`         |
| `UUID`                   | `UUID`         |
| `Enum`                   | `Text`         |
| `Enum8`                  | `Text`         |
| `Enum16`                 | `Text`         |
| `LowCardinality(T)` [^2] | _`T`'s type_   |
| `Array`                  | `Array`        |
| `Map(key, value)` [^3]   | `Map`          |
| `Nullable(T)` [^4]       | _`T`'s type_   |
| `IPv4`                   | `Inet`         |
| `IPv6`                   | `Inet`         |

[^1]: Scale `s` must be in range [0, 37]. 

[^2]: `LowCardinality(T)` is supported if `T` is supported.

[^3]: `Map(key,value)` is supported only for `String` keys.

[^4]: `Nullable(T)` is supported if `T` is supported. Properties of `Nullable(T)` columns are nullable.

### Aliases

Alias types are also supported:

* `TINYINT`, `BOOL`, `BOOLEAN`, `INT1`,  as aliases of `Int8`.
* `SMALLINT`, `INT2`,  as aliases of `Int16`.
* `INT`, `INT4`, `INTEGER`, as aliases of `Int32`.
* `FLOAT` as alias of `Float32`.
* `DOUBLE` as alias of `Float64`.
* `LONGTEXT`, `MEDIUMTEXT`, `TINYTEXT`, `TEXT`, `LONGBLOB`, `MEDIUMBLOB`, `TINYBLOB`, `BLOB`, `VARCHAR`, `CHAR`, as aliases of `String`.

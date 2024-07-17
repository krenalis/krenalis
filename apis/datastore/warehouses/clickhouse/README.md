# ClickHouse Driver

## Supported types

> This section **may be obsolete**. See [this issue](https://github.com/meergo/meergo/issues/575).

The table below provides a list of supported property types when using a ClickHouse data warehouse, along with the corresponding column types that are generated on the data warehouse.

| Property Type  | Column Type              |
|----------------|--------------------------|
| `Uint8`        | `UInt8`                  |
| `Uint16`       | `UInt16`                 |
| `Uint32`       | `UInt32`                 |
| `Uint64`       | `UInt64`                 |
| `Int8`         | `Int8`                   |
| `Int16`        | `Int16`                  |
| `Int32`        | `Int32`                  |
| `Int64`        | `Int64`                  |
| `Float32`      | `Float32`                |
| `Float64`      | `Float64`                |
| `Decimal(p,s)` | `Decimal(p,s)` [^1]      |
| `String`       | `String`                 |
| `String`       | `FixedString`            |
| `Date`         | `Date`                   |
| `Int32`        | `Date32`                 |
| `DateTime`     | `DateTime`               |
| `DateTime`     | `DateTime64`             |
| `JSON`         | `JSON`                   |
| `UUID`         | `UUID`                   |
| `Text`         | `Enum`                   |
| `Text`         | `Enum8`                  |
| `Text`         | `Enum16`                 |
| _`T`'s type_   | `LowCardinality(T)` [^2] |
| `Array`        | `Array`                  |
| `Map`          | `Map(key, value)` [^3]   |
| _`T`'s type_   | `Nullable(T)` [^4]       |
| `Inet`         | `IPv4`                   |
| `Inet`         | `IPv6`                   |

[^1]: Scale `s` must be in range [0, 37]. 

[^2]: `LowCardinality(T)` is supported if `T` is supported.

[^3]: `Map(key,value)` is supported only for `String` keys.

[^4]: `Nullable(T)` is supported if `T` is supported. Properties of `Nullable(T)` columns are nullable.
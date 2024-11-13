
# Connectors Data Types

| Kind      | Go              | json.Unmarshal               | ClickHouse           | PostgreSQL  | MySQL     | Snowflake   | Parquet      | JSON            | CSV    | Excel  |
|-----------|-----------------|------------------------------|----------------------|-------------|-----------|-------------|--------------|-----------------|--------|--------|
| Boolean   | bool            | bool                         | bool                 | bool        | -         | bool        | bool         | -               | -      | -      |
| Int(8)    | int             | float64, json.Number         | int8                 | -           | int64     | -           |              | -               | -      | -      |
| Int(16)   | int             | float64, json.Number         | int16                | int64       | int64     | -           |              | -               | -      | -      |
| Int(24)   | int             | float64, json.Number         | -                    | -           | int64     | -           | -            | -               | -      | -      |
| Int(32)   | int             | float64, json.Number         | int32                | int64       | int64     | -           | int32        | -               | -      | -      |
| Int(64)   | int             | float64, json.Number         | int64                | int64       | int64     | -           | int64        | -               | -      | -      |
| Uint(8)   | uint            | float64, json.Number         | uint8                | -           | int64     | -           |              | -               | -      | -      |
| Uint(16)  | uint            | float64, json.Number         | uint16               | -           | int64     | -           |              | -               | -      | -      |
| Uint(24)  | uint            | float64, json.Number         | -                    | -           | int64     | -           | -            | -               | -      | -      |
| Uint(32)  | uint            | float64, json.Number         | uint32               | -           | int64     | -           |              | -               | -      | -      |
| Uint(64)  | uint            | float64, json.Number         | uint64               | -           | uint64    | -           |              | -               | -      | -      |
| Float(32) | float64         | float64, json.Number         | float32              | float64     | float32   | -           | float32      | -               | -      | -      |
| Float(64) | float64         | float64, json.Number         | float64              | float64     | float64   | float64     | float64      | -               | -      | -      |
| Decimal   | decimal.Decimal | string, float64, json.Number | decimal.Decimal [^1] | string      | []byte    | string      | int32, int64 | -               | -      | -      |
| DateTime  | time.Time       | string, float64, json.Number | time.Time            | time.Time   | time.Time | time.Time   | time.Time    | -               | -      | -      |
| Date      | time.Time       | string                       | time.Time [^2]       | time.Time   | time.Time | time.Time   |              | -               | -      | -      |
| Time      | time.Time       | string                       | -                    | string      | []byte    | time.Time   |              | -               | -      | -      |
| Year      | int             | float64, json.Number         | -                    | -           | int64     | -           | -            | -               | -      | -      |
| UUID      | string          | string                       | string               | string      | -         | -           |              | -               | -      | -      |
| JSON      | json.Value      | JSON types [^4]              | - [^3]               | string      | []byte    | string [^6] |              | JSON types [^4] | -      | -      |
| Inet      | string          | string                       | net.IP               | string [^8] | -         | -           | -            | -               | -      | -      |
| Text      | string          | string                       | string               | string      | []byte    | string      | []byte       | -               | string | string |
| Array(T)  | []any           | []any                        | []T                  | []T [^5]    | -         | string [^7] | -            | -               | -      | -      |
| Object    | map[string]any  | map[string]any               | -                    | -           | -         | -           | -            | -               | -      | -      |
| Map(T)    | map[string]any  | map[string]any               | map[string]T         | -           | -         | string [^7] | -            | -               | -      | -      |


[^1]: The [github.com/shopspring/decimal.Decimal](https://pkg.go.dev/github.com/shopspring/decimal#Decimal) type.

[^2]: The ClickHouse driver, for the `Date32` type, returns a `time.Time` value not corresponding to the stored value.

[^3]: The `JSON` type in ClickHouse is [experimental](https://github.com/ClickHouse/ClickHouse/issues/68428).

[^4]: JSON types: `json.RawMessage`, `bool`, `string`, `json.Number`, `float64`, `map[string]any`, and `[]any`. `nil` represents a `nil` value, not the JSON `null`.

[^5]: For the connector, the support for Arrays is not implemented neither for the `Query` method nor for the `Upsert` method, but for the latter it may be implemented by changing the `quoteTable` function.

[^6]: As Snowflake `VARIANT` type.

[^7]: Only supports `Array(JSON)` and `Map(JSON)` as Snowflake `ARRAY` and `OBJECT` types.

[^8]: The returned IP address also includes the netmask bits, as in `"127.0.0.1/32"`.

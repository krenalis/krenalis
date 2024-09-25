
# Connectors Data Types

| Kind      | Go              | json.Unmarshal               | ClickHouse      | PostgreSQL  | MySQL     | Snowflake   | Parquet      | JSON            | CSV    | Excel  |
|-----------|-----------------|------------------------------|-----------------|-------------|-----------|-------------|--------------|-----------------|--------|--------|
| Boolean   | bool            | bool                         | bool            | bool        | -         | bool        | bool         | -               | -      | -      |
| Int(8)    | int             | float64, json.Number         | int8            | -           | int64     | -           |              | -               | -      | -      |
| Int(16)   | int             | float64, json.Number         | int16           | int64       | int64     | -           |              | -               | -      | -      |
| Int(24)   | int             | float64, json.Number         | -               | -           | int64     | -           | -            | -               | -      | -      |
| Int(32)   | int             | float64, json.Number         | int32           | int64       | int64     | -           | int32        | -               | -      | -      |
| Int(64)   | int             | float64, json.Number         | int64           | int64       | int64     | -           | int64        | -               | -      | -      |
| Uint(8)   | uint            | float64, json.Number         | uint8           | -           | int64     | -           |              | -               | -      | -      |
| Uint(16)  | uint            | float64, json.Number         | uint16          | -           | int64     | -           |              | -               | -      | -      |
| Uint(24)  | uint            | float64, json.Number         | -               | -           | int64     | -           | -            | -               | -      | -      |
| Uint(32)  | uint            | float64, json.Number         | uint32          | -           | int64     | -           |              | -               | -      | -      |
| Uint(64)  | uint            | float64, json.Number         | uint64          | -           | uint64    | -           |              | -               | -      | -      |
| Float(32) | float64         | float64, json.Number         | float32         | float64     | float32   | -           | float32      | -               | -      | -      |
| Float(64) | float64         | float64, json.Number         | float64         | float64     | float64   | float64     | float64      | -               | -      | -      |
| Decimal   | decimal.Decimal | string, float64, json.Number | decimal.Decimal | string      | []byte    | string      | int32, int64 | -               | -      | -      |
| DateTime  | time.Time       | string, float64, json.Number | time.Time       | time.Time   | time.Time | time.Time   | time.Time    | -               | -      | -      |
| Date      | time.Time       | string                       | time.Time [^1]  | time.Time   | time.Time | time.Time   |              | -               | -      | -      |
| Time      | time.Time       | string                       | -               | string      | []byte    | time.Time   |              | -               | -      | -      |
| Year      | int             | float64, json.Number         | -               | -           | int64     | -           | -            | -               | -      | -      |
| UUID      | string          | string                       | string          | string      | -         | -           |              | -               | -      | -      |
| JSON      | JSON types [^3] | JSON types [^3]              | - [^2]          | []byte [^7] | []byte    | string [^5] |              | JSON types [^3] | -      | -      |
| Inet      | string          | string                       | net.IP          | string      | -         | -           | -            | -               | -      | -      |
| Text      | string          | string                       | string          | string      | []byte    | string      | []byte       | -               | string | string |
| Array(T)  | []any           | []any                        | []T             | []T [^4]    | -         | string [^6] | -            | -               | -      | -      |
| Object    | map[string]any  | map[string]any               | -               | -           | -         | -           | -            | -               | -      | -      |
| Map(T)    | map[string]any  | map[string]any               | map[string]T    | -           | -         | string [^6] | -            | -               | -      | -      |

[^1]: the ClickHouse driver, for the `Date32` type, returns a `time.Time` value not corresponding to the stored value.

[^2]: the `JSON` type in ClickHouse is [experimental](https://github.com/ClickHouse/ClickHouse/issues/68428).

[^3]: JSON types: `json.RawMessage`, `bool`, `string`, `json.Number`, `float64`, `map[string]any`, and `[]any`. `nil` represents a `nil` value, not the JSON `null`.

[^4]: For the connector, the support for Arrays is not implemented neither for the `Query` method nor for the `Upsert` method, but for the latter it may be implemented by changing the `quoteTable` function.

[^5]: As Snowflake `VARIANT` type.

[^6]: Only supports `Array(JSON)` and `Map(JSON)` as Snowflake `ARRAY` and `OBJECT` types.

[^7]: When using the packages `database/sql` and `github.com/jackc/pgx` as drivers, the Go type of the returned values is `[]byte` for both `json` and `jsonb`.

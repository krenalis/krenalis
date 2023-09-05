
# Data Types

| Physical type | Go              | json.Unmarshal                | ClickHouse      | PostgreSQL | MySQL      | Snowflake   | Parquet      | JSON            | CSV    | Excel  |
|---------------|-----------------|-------------------------------|-----------------|------------|------------|-------------|--------------|-----------------|--------|--------|
| Boolean       | bool            | bool                          | bool            | bool       | -          | bool        | bool         | -               | -      | -      |
| Int           | int             | float64, json.Number          | int32           | int64      | []byte     | -           | int32        | -               | -      | -      |
| Int8          | int             | float64, json.Number          | int8            | -          | []byte     | -           |              | -               | -      | -      |
| Int16         | int             | float64, json.Number          | int16           | int64      | []byte     | -           |              | -               | -      | -      |
| Int24         | int             | float64, json.Number          | -               | -          | []byte     | -           | -            | -               | -      | -      |
| Int64         | int             | float64, json.Number          | int64           | int64      | []byte     | -           | int64        | -               | -      | -      |
| UInt          | uint            | float64, json.Number          | uint32          | -          | []byte     | -           |              | -               | -      | -      |
| UInt8         | uint            | float64, json.Number          | uint8           | -          | []byte     | -           |              | -               | -      | -      |
| UInt16        | uint            | float64, json.Number          | uint16          | -          | []byte     | -           |              | -               | -      | -      |
| UInt24        | uint            | float64, json.Number          | -               | -          | []byte     | -           | -            | -               | -      | -      |
| UInt64        | uint            | float64, json.Number          | uint64          | -          | []byte     | -           |              | -               | -      | -      |
| Float         | float64         | float64, json.Number          | float64         | float64    | []byte     | float64     | float64      | -               | -      | -      |
| Float32       | float64         | float64, json.Number          | float32         | float64    | []byte     | -           | float32      | -               | -      | -      |
| Decimal       | decimal.Decimal | string, float64, json.Number  | decimal.Decimal | string     | []byte     | string      | int32, int64 | -               | -      | -      |
| DateTime      | time.Time       | string, float64, json.Number  | time.Time       | time.Time  | time.Time  | time.Time   | time.Time    | -               | -      | -      |
| Date          | time.Time       | string                        | time.Time [^2]  | time.Time  | time.Time  | time.Time   |              | -               | -      | -      |
| Time          | time.Time       | string                        | -               | string     | []byte     | time.Time   |              | -               | -      | -      |
| Year          | int             | float64, json.Number          | -               | -          | []byte     | -           | -            | -               | -      | -      |
| UUID          | string          | string                        | string          | string     | -          | -           |              | -               | -      | -      |
| JSON          | JSON types [^4] | JSON types [^4]               | - [^3]          | []byte     | - [^1]     | string [^6] |              | JSON types [^4] | -      | -      |
| Inet          | string          | string                        | net.IP          | string     | -          | -           | -            | -               | -      | -      |
| Text          | string          | string                        | string          | string     | []byte     | string      | []byte       | -               | string | string |
| Array(T)      | []any           | []any                         | []T             | []T [^5]   | -          | string [^7] | -            | -               | -      | -      |
| Object        | map[string]any  | map[string]any                | -               | -          | -          | -           | -            | -               | -      | -      |
| Map(T)        | map[string]any  | map[string]any                | map[string]T    | -          | -          | string [^7] | -            | -               | -      | -      |

[^1]: Even by declaring a column as type `json`, the MySQL driver returns the type `VARCHAR` instead of `JSON`.

[^2]: the ClickHouse driver, for the `Date32` type, returns a `time.Time` value not corresponding to the stored value.

[^3]: the `JSON` type in ClickHouse is experimental.

[^4]: JSON types: `json.RawMessage`, `bool`, `string`, `json.Number`, `float64`, `map[string]any`, and `[]any`. `nil` represents a `nil` value, not the JSON `null`.

[^5]: Only the data warehouse driver supports arrays. Arrays of numeric, array, and composite types are not supported yet.

[^6]: As Snowflake `VARIANT` type.

[^7]: Only supports `Array(JSON)` and `Map(JSON)` as Snowflake `ARRAY` and `OBJECT` types.

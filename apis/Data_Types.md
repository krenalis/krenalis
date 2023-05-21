
# Data Types

| Physical type | Go              | json.Unmarshal               | ClickHouse      | PostgreSQL | MySQL      | Parquet      | CSV    | Excel  |
|---------------|-----------------|------------------------------|-----------------|------------|------------|--------------|--------|--------|
| Boolean       | bool            | bool                         | bool            | bool       | -          | bool         | -      | -      |
| Int           | int             | float64, json.Number         | int32           | int64      | []byte     | int32        | -      | -      |
| Int8          | int             | float64, json.Number         | int8            | int64      | []byte     |              | -      | -      |
| Int16         | int             | float64, json.Number         | int16           | int64      | []byte     |              | -      | -      |
| Int24         | int             | float64, json.Number         | -               | -          | []byte     | -            | -      | -      |
| Int64         | int             | float64, json.Number         | int64           | int64      | []byte     | int64        | -      | -      |
| UInt          | uint            | float64, json.Number         | uint32          | -          | []byte     |              | -      | -      |
| UInt8         | uint            | float64, json.Number         | uint8           | -          | []byte     |              | -      | -      |
| UInt16        | uint            | float64, json.Number         | uint16          | -          | []byte     |              | -      | -      |
| UInt24        | uint            | float64, json.Number         | -               | -          | []byte     | -            | -      | -      |
| UInt64        | uint            | float64, json.Number         | uint64          | -          | []byte     |              | -      | -      |
| Float         | float64         | float64, json.Number         | float64         | float64    | []byte     | float64      | -      | -      |
| Float32       | float64         | float64, json.Number         | float32         | float64    | []byte     | float32      | -      | -      |
| Decimal       | decimal.Decimal | string, float64, json.Number | decimal.Decimal | string     | []byte     | int32, int64 | -      | -      |
| DateTime      | time.Time       | string, float64, json.Number | time.Time       | time.Time  | time.Time  | time.Time    | -      | -      |
| Date          | time.Time       | string                       | time.Time [^2]  | time.Time  | time.Time  |              | -      | -      |
| Time          | time.Time       | string                       | -               | string     | []byte     |              | -      | -      |
| Year          | int             | float64, json.Number         | -               | int64      | []byte     | -            | -      | -      |
| UUID          | string          | string                       | string          | string     | -          |              | -      | -      |
| JSON          | json.RawMessage | json.RawMessage, any [^4]    | - [^3]          | []byte     | - [^1]     |              | -      | -      |
| Inet          | string          | string                       | net.IP          | string     | -          | -            | -      | -      |
| Text          | string          | string                       | string          | string     | []byte     | []byte       | string | string |
| Array(T)      | []any           | []any                        | []T             | -          | -          | -            | -      | -      |
| Object        | map[string]any  | map[string]any               | -               | -          | -          | -            | -      | -      |
| Map(T)        | map[string]any  | map[string]any               | map[string]T    | -          | -          | -            | -      | -      |

[^1]: Even by declaring a column as type `json`, the MySQL driver returns the type `VARCHAR` instead of `JSON`.

[^2]: the ClickHouse driver, for the `Date32` type, returns a `time.Time` value not corresponding to the stored value.

[^3]: the `JSON` type in ClickHouse is experimental.

[^4]: `json.RawMessage` values represent JSON code, any other value will be marshalled into JSON. `nil` represents a `nil` value and not the JSON `null`.

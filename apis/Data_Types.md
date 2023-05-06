
# Data Types

| Physical type  | Go                 | json.Unmarshal               | ClickHouse      | PostgreSQL | MySQL      | Parquet      | CSV    | Excel  |
|----------------|--------------------|------------------------------|-----------------|------------|------------|--------------|--------|--------|
| Boolean        | bool               | bool                         | - [^3]          | bool       | -          | bool         | -      | -      |
| Int            | int                | float64, json.Number         | int32           | int64      | []byte     | int32        | -      | -      |
| Int8           | int                | float64, json.Number         | int8            | int64      | []byte     |              | -      | -      |
| Int16          | int                | float64, json.Number         | int16           | int64      | []byte     |              | -      | -      |
| Int24          | int                | float64, json.Number         | -               | -          | []byte     | -            | -      | -      |
| Int64          | int                | float64, json.Number         | int64           | int64      | []byte     | int64        | -      | -      |
| UInt           | uint               | float64, json.Number         | uint32          | -          | []byte     |              | -      | -      |
| UInt8          | uint               | float64, json.Number         | uint8           | -          | []byte     |              | -      | -      |
| UInt16         | uint               | float64, json.Number         | uint16          | -          | []byte     |              | -      | -      |
| UInt24         | uint               | float64, json.Number         | -               | -          | []byte     | -            | -      | -      |
| UInt64         | uint               | float64, json.Number         | uint64          | -          | []byte     |              | -      | -      |
| Float          | float64            | float64, json.Number         | float64         | float64    | []byte     | float64      | -      | -      |
| Float32        | float64            | float64, json.Number         | float32         | float64    | []byte     | float32      | -      | -      |
| Decimal        | decimal.Decimal    | string, float64, json.Number | decimal.Decimal | string     | []byte     | int32, int64 | -      | -      |
| DateTime       | connector.DateTime | string, float64, json.Number | time.Time       | time.Time  | time.Time  | time.Time    | -      | -      |
| Date           | connector.Date     | string                       | time.Time [^2]  | time.Time  | time.Time  |              | -      | -      |
| Time           | connector.Time     | string, float64, json.Number | -               | -          | []byte     |              | -      | -      |
| Year           | int                | float64, json.Number         | -               | int64      | []byte     | -            | -      | -      |
| UUID           | string             | string                       | string          | string     | -          |              | -      | -      |
| JSON           | json.RawMessage    | string, json.RawMessage      | - [^4]          | []byte     | - [^1]     |              | -      | -      |
| Inet           | string             | string                       | - [^3]          | string     | -          | -            | -      | -      |
| Text           | string             | string                       | string          | string     | []byte     | []byte       | string | string |
| Array          | []any              | []any                        | -               | -          | -          | -            | -      | -      |
| Object         | map[string]any     | map[string]any               | -               | -          | -          | -            | -      | -      |
| Map            | map[string]any     | map[string]any               | -               | -          | -          | -            | -      | -      |

[^1]: Even by declaring a column as type `json`, the MySQL driver returns the type `VARCHAR` instead of `JSON`.

[^2]: the ClickHouse driver, for the `Date32` type, returns a `time.Time` value not corresponding to the stored value.

[^3]: the ClickHouse driver, for `Bool`, `IPv4` and `IPv6` types, does not support the `sql.Scanner` interface.

[^4]: the `JSON` type in ClickHouse is experimental.
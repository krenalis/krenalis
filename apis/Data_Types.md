
# Data Types

| Physical type  | Go              | json.Unmarshal               | PostgreSQL | MySQL      | Parquet      | CSV    | Excel  |
|----------------|-----------------|------------------------------|------------|------------|--------------|--------|--------|
| Boolean        | bool            | bool                         | bool       | -          | bool         | -      | -      |
| Int            | int             | float64, json.Number         | int64      | []byte     | int32        | -      | -      |
| Int8           | int             | float64, json.Number         | int64      | []byte     |              | -      | -      |
| Int16          | int             | float64, json.Number         | int64      | []byte     |              | -      | -      |
| Int24          | int             | float64, json.Number         | int64      | []byte     | -            | -      | -      |
| Int64          | int             | float64, json.Number         | int64      | []byte     | int64        | -      | -      |
| UInt           | uint            | float64, json.Number         | -          | []byte     |              | -      | -      |
| UInt8          | uint            | float64, json.Number         | -          | []byte     |              | -      | -      |
| UInt16         | uint            | float64, json.Number         | -          | []byte     |              | -      | -      |
| UInt24         | uint            | float64, json.Number         | -          | []byte     | -            | -      | -      |
| UInt64         | uint            | float64, json.Number         | -          | []byte     |              | -      | -      |
| Float          | float64         | float64, json.Number         | float64    | []byte     | float64      | -      | -      |
| Float32        | float64         | float64, json.Number         | float64    | []byte     | float32      | -      | -      |
| Decimal        | decimal.Decimal | string, float64, json.Number | string     | []byte     | int32, int64 | -      | -      |
| DateTime       | time.Time       | string, float64, json.Number | time.Time  | time.Time  | time.Time    | -      | -      |
| Date           | time.Time       | string, float64, json.Number | time.Time  | time.Time  |              | -      | -      |
| Time           | time.Time       | string, float64, json.Number | -          | []byte     |              | -      | -      |
| Year           | int             | float64, json.Number         | int64      | []byte     | -            | -      | -      |
| UUID           | uuid.UUID       | string                       | string     | -          |              | -      | -      |
| JSON           | json.RawMessage | string, json.RawMessage      | []byte     | - [^1]     |              | -      | -      |
| Inet           | netip.Addr      | string                       | string     | -          | -            | -      | -      |
| Text           | string          | string                       | string     | []byte     | []byte       | string | string |
| Array(T)       | []T             | []any                        | -          | -          | -            | -      | -      |
| Object         | map[string]any  | map[string]any               | -          | -          | -            | -      | -      |
| Map(T)         | map[string]T    | map[string]any               | -          | -          | -            | -      | -      |

[^1]: Even by declaring a column as type `json`, the MySQL driver returns the type `VARCHAR` instead of `JSON`.

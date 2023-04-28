
# Data Types

| Physical type  | Go              | PostgreSQL | MySQL      | Parquet
|----------------|-----------------|------------|------------|-----------
| Boolean        | bool            | bool       | -          | bool
| Int            | int             | int64      | []byte     | int32
| Int8           | int             | int64      | []byte     |
| Int16          | int             | int64      | []byte     |
| Int24          | int             | int64      | []byte     | -
| Int64          | int             | int64      | []byte     | int64
| UInt           | uint            | -          | []byte     |
| UInt8          | uint            | -          | []byte     |
| UInt16         | uint            | -          | []byte     |
| UInt24         | uint            | -          | []byte     | -
| UInt64         | uint            | -          | []byte     |
| Float          | float64         | float64    | []byte     | float64
| Float32        | float64         | float64    | []byte     | float32
| Decimal        | decimal.Decimal | string     | []byte     | int32, int64
| DateTime       | time.Time       | time.Time  | time.Time  | time.Time
| Date           | time.Time       | time.Time  | time.Time  |
| Time           | time.Time       | -          | []byte     |
| Year           | int             | int64      | []byte     | -
| UUID           | uuid.UUID       | string     | -          |
| JSON           | json.RawMessage | []byte     | - [^1]     |
| Inet           | netip.Addr      | string     | -          | -
| Text           | string          | string     | []byte     | []byte
| Array(T)       | []T             | -          | -          | -
| Object         | map[string]any  | -          | -          | -
| Map(T)         | map[string]T    | -          | -          | -

[^1]: Even by declaring a column as type `json`, the MySQL driver returns the type `VARCHAR` instead of `JSON`.


# Data Types

| Physical type  | Go              | PostgreSQL | MySQL      | Parquet      | json.Unmarshal
|----------------|-----------------|------------|------------|--------------|------------------------------
| Boolean        | bool            | bool       | -          | bool         | bool                
| Int            | int             | int64      | []byte     | int32        | float64, json.Number
| Int8           | int             | int64      | []byte     |              | float64, json.Number
| Int16          | int             | int64      | []byte     |              | float64, json.Number
| Int24          | int             | int64      | []byte     | -            | float64, json.Number
| Int64          | int             | int64      | []byte     | int64        | float64, json.Number
| UInt           | uint            | -          | []byte     |              | float64, json.Number
| UInt8          | uint            | -          | []byte     |              | float64, json.Number
| UInt16         | uint            | -          | []byte     |              | float64, json.Number
| UInt24         | uint            | -          | []byte     | -            | float64, json.Number
| UInt64         | uint            | -          | []byte     |              | float64, json.Number
| Float          | float64         | float64    | []byte     | float64      | float64, json.Number
| Float32        | float64         | float64    | []byte     | float32      | float64, json.Number
| Decimal        | decimal.Decimal | string     | []byte     | int32, int64 | string, float64, json.Number
| DateTime       | time.Time       | time.Time  | time.Time  | time.Time    | string, float64, json.Number 
| Date           | time.Time       | time.Time  | time.Time  |              | string, float64, json.Number
| Time           | time.Time       | -          | []byte     |              | string, float64, json.Number
| Year           | int             | int64      | []byte     | -            | float64, json.Number
| UUID           | uuid.UUID       | string     | -          |              | string
| JSON           | json.RawMessage | []byte     | - [^1]     |              | string, json.RawMessage
| Inet           | netip.Addr      | string     | -          | -            | string
| Text           | string          | string     | []byte     | []byte       | string
| Array(T)       | []T             | -          | -          | -            | []any
| Object         | map[string]any  | -          | -          | -            | map[string]any
| Map(T)         | map[string]T    | -          | -          | -            | map[string]any

[^1]: Even by declaring a column as type `json`, the MySQL driver returns the type `VARCHAR` instead of `JSON`.

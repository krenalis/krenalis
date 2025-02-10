
# Connectors Data Types

| Kind      | Go              | json.Unmarshal               | ClickHouse           | PostgreSQL  | MySQL     | Snowflake   | Parquet      | JSON            | CSV    | Excel  |
|-----------|-----------------|------------------------------|----------------------|-------------|-----------|-------------|--------------|-----------------|--------|--------|
| boolean   | bool            | bool                         | bool                 | bool        | -         | bool        | bool         | -               | -      | -      |
| int(8)    | int             | float64, json.Number         | int8                 | -           | int64     | -           |              | -               | -      | -      |
| int(16)   | int             | float64, json.Number         | int16                | int64       | int64     | -           |              | -               | -      | -      |
| int(24)   | int             | float64, json.Number         | -                    | -           | int64     | -           | -            | -               | -      | -      |
| int(32)   | int             | float64, json.Number         | int32                | int64       | int64     | -           | int32        | -               | -      | -      |
| int(64)   | int             | float64, json.Number         | int64                | int64       | int64     | -           | int64        | -               | -      | -      |
| uint(8)   | uint            | float64, json.Number         | uint8                | -           | int64     | -           |              | -               | -      | -      |
| uint(16)  | uint            | float64, json.Number         | uint16               | -           | int64     | -           |              | -               | -      | -      |
| uint(24)  | uint            | float64, json.Number         | -                    | -           | int64     | -           | -            | -               | -      | -      |
| uint(32)  | uint            | float64, json.Number         | uint32               | -           | int64     | -           |              | -               | -      | -      |
| uint(64)  | uint            | float64, json.Number         | uint64               | -           | uint64    | -           |              | -               | -      | -      |
| float(32) | float64         | float64, json.Number         | float32              | float64     | float32   | -           | float32      | -               | -      | -      |
| float(64) | float64         | float64, json.Number         | float64              | float64     | float64   | float64     | float64      | -               | -      | -      |
| decimal   | decimal.Decimal | string, float64, json.Number | decimal.Decimal [^1] | string      | []byte    | string      | int32, int64 | -               | -      | -      |
| datetime  | time.Time       | string, float64, json.Number | time.Time            | time.Time   | time.Time | time.Time   | time.Time    | -               | -      | -      |
| date      | time.Time       | string                       | time.Time [^2]       | time.Time   | time.Time | time.Time   |              | -               | -      | -      |
| time      | time.Time       | string                       | -                    | string      | []byte    | time.Time   |              | -               | -      | -      |
| year      | int             | float64, json.Number         | -                    | -           | int64     | -           | -            | -               | -      | -      |
| uuid      | string          | string                       | string               | string      | -         | -           |              | -               | -      | -      |
| json      | json.Value      | JSON types [^4]              | - [^3]               | string      | []byte    | string [^6] |              | JSON types [^4] | -      | -      |
| inet      | string          | string                       | net.IP               | string [^8] | -         | -           | -            | -               | -      | -      |
| text      | string          | string                       | string               | string      | []byte    | string      | []byte       | -               | string | string |
| array(T)  | []any           | []any                        | []T                  | []T [^5]    | -         | string [^7] | -            | -               | -      | -      |
| object    | map[string]any  | map[string]any               | -                    | -           | -         | -           | -            | -               | -      | -      |
| map(T)    | map[string]any  | map[string]any               | map[string]T         | -           | -         | string [^7] | -            | -               | -      | -      |


[^1]: The [github.com/shopspring/decimal.Decimal](https://pkg.go.dev/github.com/shopspring/decimal#Decimal) type.

[^2]: The ClickHouse driver, for the `Date32` type, returns a `time.Time` value not corresponding to the stored value.

[^3]: The `JSON` type in ClickHouse is [experimental](https://github.com/ClickHouse/ClickHouse/issues/68428).

[^4]: JSON types: `json.RawMessage`, `bool`, `string`, `json.Number`, `float64`, `map[string]any`, and `[]any`. `nil` represents a `nil` value, not the JSON `null`.

[^5]: For the connector, the support for arrays is not implemented neither for the `Query` method nor for the `Upsert` method, but for the latter it may be implemented by changing the `quoteTable` function.

[^6]: As Snowflake `VARIANT` type.

[^7]: Only supports `array(json)` and `map(json)` as Snowflake `ARRAY` and `OBJECT` types.

[^8]: The returned IP address also includes the netmask bits, as in `"127.0.0.1/32"`.

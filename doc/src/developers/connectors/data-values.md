# Data Values

When representing user, group, and event data in a connector, these are mapped to specific Go types based on the data type. For example, in an export, a `DateTime` value is represented in Go as `time.Time`, and a `Text` value is represented as a `string`.

## Data Import

During an import, connectors return data to import, such as user information. This data must use specific types according to the data type. Key scenarios for handling data in an app connector include:

- [`Records`](./app/users-and-groups.md#read-records) method of apps returns properties of users or groups.
- `ReceiveWebhook` method of apps may return properties of users or groups.
- [`Query`](database.md#query-method) method of databases returns the resulting rows from a query.
- [`Read`](file.md#read-method) method of files takes a `RecordWriter` whose methods return the read records.

Connectors can directly return values deserialized from JSON using the [`json.Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal) function, because the types returned by this function are supported.

The table below shows, for each data type, which Go type a connector can return. Note that in the "Examples" column, the symbol "→" shows how a value returned by a connector is interpreted.

| Data Type                                                           | Go Types                                                                                                                                                                                                                                                                  | Examples                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
|---------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Boolean` [^1]                                                      | ◦`bool`<br/>◦`string`                                                                                                                                                                                                                                                     | `true`<br/>`"true"` → `true`, `"false"` → `false`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `Int(n)`<br/>`Uint(n)`<br/>`Float(n)`<br/>`Decimal(p,s)`<br/>`Year` | ◦`int`<br/>◦`int8`<br/>◦`int16`<br/>◦`int32`<br/>◦`int64`<br/>◦`uint`<br/>◦`uint8`<br/>◦`uint16`<br/>◦`uint32`<br/>◦`uint64`<br/>◦`float32`<br/>◦`float64`<br/>◦[`decimal.Decimal`](https://pkg.go.dev/github.com/shopspring/decimal#Decimal)<br/>◦`string`<br/>◦`[]byte` | `12`  // this and the following examples are for the `Int(n)` data type<br/>`int8(23)` → 23<br/>`int16(512)` → 512<br/>`int32(-802)` → -802<br/>`int64(-1061274994)` → -1061274994<br/>`uint(3095322305)` → 3095322305<br/>`uint8(250)` → 250<br/>`uint16(12000)` → 12000<br/>`uint32(6038461)` → 6038461<br/>`uint64(612)` → 612<br/>`float32(51.0)` → 51.0<br/>`1048553.0` → 1048553<br/>`decimal.RequireFromString("932811")` → `932811`<br/>`"932811"` → `932811`<br/>`[]byte("932811")` → `932811`                                        |
| `DateTime`                                                          | ◦[`time.Time`](https://pkg.go.dev/time#Time)<br/>◦`string`<br/>◦`float64`                                                                                                                                                                                                 | `time.Date(2024, 4, 11, 16, 48, 26, 377105982, time.UTC)`<br/>`"2024-04-11 15:59:05"`<br/>`1712851145.0``                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `Date` [^2]                                                         | ◦[`time.Time`](https://pkg.go.dev/time#Time)<br/>◦`string`                                                                                                                                                                                                                | `time.Date(2024, 4, 11, 0, 0, 0, 0, time.UTC)`<br/>`"2027-09-15"`                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `Time` [^3]                                                         | ◦[`time.Time`](https://pkg.go.dev/time#Time)<br/>◦`string`<br/>◦`[]byte`                                                                                                                                                                                                  | `time.Date(1970, 1, 1, 16, 38, 50, 204813055, time.UTC)`<br/>`"16:38:50.204813055"`<br/>`[]byte("23:09:26.047126396")` → `time.Date(1970, 1, 1, 23, 9, 26, 47126396, time.UTC)`                                                                                                                                                                                                                                                                                                                                                                |
| `UUID` [^4]                                                         | `string`                                                                                                                                                                                                                                                                  | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
| `JSON` [^5] [^6]                                                    | ◦`json.Value`<br/>◦[`json.Marshaler`](https://pkg.go.dev/encoding/json#Marshaler)<br/>◦`string`<br/>◦`[]byte`                                                                                                                                                             | ```json.Value(`{"p": 561}`)```<br/>◦[`json.RawMessage`](https://pkg.go.dev/encoding/json#RawMessage)(``` `{"p": 561}` ```) → ```json.Value(`{"p": 561}`)```<br/>`"[1, 2, 3]"` → ```json.Value(`[1, 2, 3]`)```<br/>```[]byte(`{"points": 5}`)``` → ```json.Value(`{"points": 5}`)```                                                                                                                                                                                                                                                            |
| `Inet`                                                              | ◦`string`<br/>◦[`net.IP`](https://pkg.go.dev/net#IP)                                                                                                                                                                                                                      | `"192.168.1.1"`<br/>`net.ParseIP("192.168.1.1")` → `"192.168.1.1"`                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `Text`                                                              | ◦`string`<br/>◦`[]byte`                                                                                                                                                                                                                                                   | `"Emily Johnson"`<br/>`[]byte("Emily Johnson") → "Emily Johnson"`                                                                                                                                                                                                                                                                                                                                                                                                                                                                              |
| `Array(T)` [^7]                                                     | ◦`[]any`<br/>◦`[]T`<br/>◦`string`                                                                                                                                                                                                                                         | `[]any{618, 262, 791}`<br/>`[]int{19, -67, 8}` → `[]any{19, -67, 8}`<br/>`"[19,-67,8]"` → `[]any{19,  -67,  8}`                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `Object`                                                            | `map[string]any`                                                                                                                                                                                                                                                          | `map[string]any{"plan": "Premium"}`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            | 
| `Map(T)` [^8]                                                       | ◦`map[string]any`<br/>◦`map[string]T`<br/>◦`string`                                                                                                                                                                                                                       | `map[string]any{"1": "a", "2": "b", "others": "c"}`<br/>`map[string]int{"a": 1, "b": 2}` → `map[string]any{"a": 1, "b": 2}`<br/>`"{"a":1,"b":2}"` → `map[string]any{"a": 1, "b": 2}`                                                                                                                                                                                                                                                                                                                                                           |

[^1]: If the returned type is `string`, the only allowed values are `"true"` and `"false"`.   

[^2]: If the returned type is `time.Time`, the time part is not relevant and is not considered.

[^3]: If the returned type is `time.Time`, the date part is not relevant and is not considered.

[^4]: The value must be in the UUID textual representation "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" as defined by RFC 9562.
 
[^5]: `nil` represents a `nil` value, not the JSON `null`. A JSON `null` is represented by `json.Value("null")`.

[^6]: `json.Value`, `string`, and  `[]byte` values must contain valid JSON; the `MarshalJSON` method of the `json.Marshaler` interface must return valid JSON.

[^7]: If the returned type is `string`, the value must represent a JSON Array.

[^8]: If the returned type is `string`, the value must represent a JSON Object.

### Time Layouts

`DateTime`, `Date`, and `Time` values can also be returned as `string`, not just [`time.Time`](https://pkg.go.dev/time#Time), and `Time` values can be also returned as `[]byte`. The format should follow ISO 8601, but alternative formats can be specified when registering the connector, using the layout supported by the [time.Parse](https://pkg.go.dev/time#Parse) function for each of these data types.

For example, when registering an app connector:

```go
meergo.RegisterApp(meergo.AppInfo{
    ...
    TimeLayouts: meergo.TimeLayouts{
        DateTime: "2006-01-02T15:04:05.999Z",
        Date:     "2006-01-02",
        Time:     "15:04:05.999Z",
    }
    ...
}, New)
```

#### Unix Epoch

For the `DateTime` field of `TimeLayouts`, you can also specify special layout strings such as `"unix"`, `"unixmilli"`, `"unixmicro"`, or `"unixnano"`.

When one of these layouts is used, the values returned as `string` or `float64` are interpreted respectively as the number of seconds, milliseconds, microseconds, and nanoseconds since the Unix epoch (January 1, 1970, at 00:00:00 UTC).

## Data Export

During an export, when a connector (whether app, database, or file) receives data to export, such as user information, it receives Go types specific to the data types. This involves methods such as:

- [`EventRequest`](app/dispatch-events.md#dispatching-an-event) method of apps takes extra event information.
- [`Upsert`](app/users-and-groups.md#update-and-create-records) method of apps takes properties of a user or group.
- [`Upsert`](database.md#upsert-method) method of databases takes rows to be added or updated.
- [`Write`](file.md#write-method) method of files takes a `RecordReader` whose `Record` method returns the next record to write.

The following table shows, for each data type, what Go type a connector should expect when it receives a value. In the "Examples" column, the symbol "→" illustrates how a connector should interpret a received value.

| Data Type       | Go Types                                                                      | Example                                                   |
|-----------------|-------------------------------------------------------------------------------|-----------------------------------------------------------|
| `Boolean`       | `bool`                                                                        | `true`                                                    |
| `Int(n)`        | `int`                                                                         | `-583`                                                    |
| `Uint(n)`       | `uint`                                                                        | `1499`                                                    |
| `Float(n)`      | `float64`                                                                     | `21.907305`                                               |
| `Decimal(p,s)`  | [`decimal.Decimal`](https://pkg.go.dev/github.com/shopspring/decimal#Decimal) | `decimal.RequireFromString("-123.45")`                    |
| `DateTime` [^9] | [`time.Time`](https://pkg.go.dev/time#Time)                                   | `time.Date(2024, 4, 11, 16, 48, 26, 377105982, time.UTC)` |
| `Date` [^10]    | [`time.Time`](https://pkg.go.dev/time#Time)                                   | `time.Date(2024, 4, 11, 0, 0, 0, 0, time.UTC)`            |
| `Time` [^11]    | [`time.Time`](https://pkg.go.dev/time#Time)                                   | `time.Date(1970, 1, 1, 16, 48, 26, 377105982, time.UTC)`  |
| `Year`          | `int`                                                                         | `2024`                                                    |
| `UUID`          | `string`                                                                      | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`                  |
| `JSON` [^12]    | `json.Value`                                                                  | ```json.Value(`{"p": 561}`)```                            |
| `Inet`          | `string`                                                                      | `"192.168.1.1"`                                           |
| `Text`          | `string`                                                                      | `"Emily Johnson"`                                         |
| `Array(T)`      | `[]any`                                                                       | `[]any{618, 262, 791}`                                    |
| `Object`        | `map[string]any`                                                              | `map[string]any{"plan": "Premium"}`                       | 
| `Map(T)`        | `map[string]any`                                                              | `map[string]any{"1": "a", "2": "b", "others": "c"}`       |


[^9]: The location is always `time.UTC`.

[^10]: The time part is always zero and the location is always `time.UTC`.

[^11]: The date is always set to January 1, 1970, and the location is always `time.UTC`.

[^12]: `nil` represents a `nil` value, not the JSON `null`. A JSON `null` is represented by `json.Value("null")`.

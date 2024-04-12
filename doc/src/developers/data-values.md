# Data Values

When representing user, group, and event data in a connector, these are mapped to specific Go types based on the data type. For example, in an export, a `DateTime` value is represented in Go as `time.Time`, and a `Text` value is represented as a `string`.

## Data Import from Apps

During an import, app connectors return data to import, such as user information. This data must use specific types according to the data type. Key scenarios for handling data in an app connector include:

- [`Records`](./app/users-and-groups.md#read-records) method of apps returns properties of users or groups.
- `ReceiveWebhook` method of apps may return properties of users or groups.

Connectors can directly return values deserialized from JSON using the [`json.Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal) function, because the types returned by this function are supported.

The table below shows, for each data type, which Go type an app connector can return. Note that in the "Examples" column, the symbol "→" indicates how a value returned by a connector is interpreted.   

| Data Type      | Go Types                                                                                                                                                                                                           | Examples                                                                                                                                                           |
|----------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Boolean` [^1] | ◦ `bool`<br/>◦ `string`                                                                                                                                                                                            | `true`<br/>`"true"` → `true`, `"false"` → `false`                                                                                                                  |
| `Int(n)`       | ◦ `int`<br/>◦ `float64`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)                                                                                                                             | `12`<br/>`-51.6` → `-51`<br/>`json.Number("9.55")` → `9`                                                                                                           |
| `Uint(n)`      | ◦ `uint`<br/>◦ `float64`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)                                                                                                                            | `385`<br/>`18.25` → `18`<br/>`json.Number("1605")` → `1605`                                                                                                        |
| `Float(n)`     | ◦ `float64`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)                                                                                                                                         | `2.13`<br/>`json.Number("18.39027")` → `18.39027`                                                                                                                  |
| `Decimal(p,s)` | ◦ [`decimal.Decimal`](https://pkg.go.dev/github.com/shopspring/decimal#Decimal)<br/>◦ `string`<br/>◦ `float64`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)                                      | `decimal.RequireFromString("67.19")`<br/>`"73.278"`<br/>`73.278`<br/>`json.Number("73.278")`                                                                       |
| `DateTime`     | ◦ [`time.Time`](https://pkg.go.dev/time#Time)<br/>◦ `string`<br/>◦ `float64`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)                                                                        | `time.Date(2024, 04, 11, 16, 48, 26, 377105982, time.UTC)`<br/>`"2024-04-11 15:59:05"`<br/>`1712851145.0`<br/>`json.Number("1712851145")`                          |
| `Date` [^2]    | ◦ [`time.Time`](https://pkg.go.dev/time#Time)<br/>◦ `string`                                                                                                                                                       | `time.Date(2024, 04, 11, 0, 0, 0, 0, time.UTC)`<br/>`"2027-09-15"`                                                                                                 |
| `Time` [^3]    | ◦ [`time.Time`](https://pkg.go.dev/time#Time)<br/>◦ `string`                                                                                                                                                       | `time.Date(1970, 01, 01, 16, 38, 50, 204813055, time.UTC)`<br/>`"16:38:50.204813055"`                                                                              |
| `Year`         | ◦ `int`<br/>◦ `float64`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)                                                                                                                             | `1984`<br/>`2024.0` → `2024`<br/>`json.Number("2027")` → `2027`                                                                                                    | 
| `UUID`         | `string`                                                                                                                                                                                                           | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`                                                                                                                           |
| `JSON` [^4]    | ◦ [`json.RawMessage`](https://pkg.go.dev/encoding/json#RawMessage)<br/>◦ `bool`<br/>◦ `string`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)<br/>◦ `float64`<br/>◦ `map[string]any`<br/>◦ `[]any` | ```json.RawMessage(`{"points":561}`)```<br/>`true`<br/>`"gold"`<br/>`json.Number("23")`<br/>`377.402`<br/>`map[string]any{"points": 561}`<br/>`[]any{"gold", 855}` |
| `Inet`         | `string`                                                                                                                                                                                                           | `"192.168.1.1"`                                                                                                                                                    |
| `Text`         | `string`                                                                                                                                                                                                           | `"Emily Johnson"`                                                                                                                                                  |
| `Array(T)`     | `[]any`                                                                                                                                                                                                            | `[]any{618, 262, 791}`                                                                                                                                             |
| `Object`       | `map[string]any`                                                                                                                                                                                                   | `map[string]any{"plan": "Premium"}`                                                                                                                                | 
| `Map(T)`       | `map[string]any`                                                                                                                                                                                                   | `map[string]any{"1": "a", "2": "b", "others": "c"}`                                                                                                                |

[^1]: If the returned type is `string`, the only allowed values are `"true"` and `"false"`.   

[^2]: If the returned type is `time.Time`, the time part is not relevant and is not considered.

[^3]: If the returned type is `time.Time`, the date part is not relevant and is not considered.

[^4]: `nil` represents a `nil` value, not the JSON `null`. A JSON `null` is represented by `json.RawMessage("null")`.


#### DateTime, Date, and Time Layouts

`DateTime`, `Date`, and `Time` values can also be returned as `string`, not just [`time.Time`](https://pkg.go.dev/time#Time). The format should follow ISO 8601, but alternative formats can be specified when registering the app connector, using the layout supported by the [time.Parse](https://pkg.go.dev/time#Parse) function for each of these data types.

```go
	chichi.RegisterApp(chichi.AppInfo{
		...
		DateTimeLayout: "2006-01-02T15:04:05.999Z",
		DateLayout:     "2006-01-02",
		TimeLayout:     "15:04:05.999Z",
		...
	}, New)
```

For `DateTime` values can also be returned as the number of seconds, milliseconds, microseconds, and nanoseconds since the Unix epoch (January 1, 1970, at 00:00:00 UTC). This can be specified by using the special `"unix"`, `"unixmilli"`, `"unixmicro"`, or `"unixnano"` layout. The numbers returned can have type `float64` or [`json.Number`](https://pkg.go.dev/encoding/json#Number).


## Data Import from Databases and Files

During an import, database and file connectors return rows and records to import. They must use specific types according to their data type. This involves methods such as:

- [`Query`](database.md#query-method) method of databases returns the resulting rows from a query.
- [`Read`](file.md#read-method) method of files takes a `RecordWriter` whose methods return the read records.

The following table outlines, for each data type, which Go types a database and file connector can return:

| Data Type       | Go Types                                                                                                                                  | Examples                                                                                                                                                                                                                                                                          |
|-----------------|-------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Boolean`       | `bool`                                                                                                                                    | `true`                                                                                                                                                                                                                                                                            |
| `Int(n)`        | ◦ `int8`<br/>◦ `int16`<br/>◦ `int32`<br/>◦ `int64`<br/>◦ `[]byte`                                                                         | `int8(5)` → `5`<br/>`int16(-519)` → `-519`<br/>`int32(1562094)` → `1562094`<br/>`int64(-7027501743)` → `-7027501743`<br/>`[]byte("-803")` → `-803`                                                                                                                                |
| `Int(n)`        | ◦ `uint8`<br/>◦ `uint16`<br/>◦ `uint32`<br/>◦ `uint64`<br/>◦ `[]byte`                                                                     | `uint8(13)` → `13`<br/>`int16(875)` → `875`<br/>`int32(306372451)` → `306372451`<br/>`int64(8281537816)` → `8281537816`<br/>`[]byte("1794")` → `1794`                                                                                                                             |
| `Float(n)`      | ◦ `float64`<br/>◦ `float32`<br/>◦ `[]byte`                                                                                                | `18.390270271` → `18.390270271`<br/>`float32(-2.13)` → `-2.13`<br/>`[]byte("703.255")` → `703.255`                                                                                                                                                                                |
| `Decimal(p,s)`  | ◦ [`decimal.Decimal`](https://pkg.go.dev/github.com/shopspring/decimal#Decimal)<br/>◦ `int32`<br/>◦ `int64`<br/>◦ `string`<br/>◦ `[]byte` | `decimal.RequireFromString("12.301")`<br/>`int32(152)` → `decimal.RequireFromInt(152)`<br/>`int64(90481736143)` → `decimal.RequireFromInt(90481736143)`<br/>`"-705.99"` → `decimal.RequireFromString("-705.99")`<br/>`[]byte("4409253")` → `decimal.RequireFromString("4409253")` |
| `DateTime`      | [`time.Time`](https://pkg.go.dev/time#Time)                                                                                               | `time.Date(2024, 04, 11, 16, 48, 26, 377105982, time.UTC)`                                                                                                                                                                                                                        |
| `Date` [^5]     | [`time.Time`](https://pkg.go.dev/time#Time)                                                                                               | `time.Date(2024, 04, 11, 0, 0, 0, 0, time.UTC)`                                                                                                                                                                                                                                   |
| `Time` [^6]     | ◦ [`time.Time`](https://pkg.go.dev/time#Time)<br/>◦ `string`<br/>◦ `[]byte`                                                               | `time.Date(1970, 01, 01, 16, 38, 50, 204813055, time.UTC)`<br/>`"16:38:50.204813055"` → `time.Date(1970, 01, 01, 16, 38, 50, 204813055, time.UTC)`<br/>`[]byte("23:09:26.047126396")` → `time.Date(1970, 01, 01, 23, 09, 26, 47126396, time.UTC)`                                 |
| `Year`          | ◦ `int`<br/>◦ `int64`<br/>◦ `[]byte`                                                                                                      | `1984`<br/>`int64(2024)` → `2024`<br/>`[]byte("2027")` → `2027`                                                                                                                                                                                                                   | 
| `UUID`          | `string`                                                                                                                                  | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`                                                                                                                                                                                                                                          |
| `JSON`          | `[]byte`                                                                                                                                  | ```[]byte(`{"points":5}`)``` → ```json.RawMessage(`{"points":5}`)```                                                                                                                                                                                                              |
| `Inet`          | ◦ `string`<br/>◦ [`net.IP`](https://pkg.go.dev/net#IP)                                                                                    | `"192.168.1.1"`<br/>`net.ParseIP("192.168.1.1")` → `"192.168.1.1"`                                                                                                                                                                                                                |
| `Text`          | ◦ `string`<br/>◦ `[]byte`                                                                                                                 | `"Emily Johnson"`<br/>`[]byte("Emily Johnson") → "Emily Johnson"`                                                                                                                                                                                                                 |
| `Array(T)` [^7] | ◦ `[]T`<br/>◦ `string`                                                                                                                    | `[]int{19,  -67,  8}` → `[]any{19,  -67,  8}`<br/>`"[19,  -67,  8]"` → `[]any{19,  -67,  8}`                                                                                                                                                                                      |
| `Map(T)` [^8]   | ◦ `map[string]T`<br/>◦ `string`                                                                                                           | `map[string]int{"a":1, "b":2}` → `map[string]any{"a":1, "b":2}`<br/>`"{"a": 1, "b": 2}"` → `map[string]any{"a": 1, "b": 2}`                                                                                                                                                       |                                                                                                                                                                                                                                |

[^5]: The time part is not relevant and is not considered.

[^6]: The date part is not relevant and is not considered.

[^7]: If the returned type is `string`, the value must represent a JSON Array.

[^8]: If the returned type is `string`, the value must represent a JSON Object.


## Data Export

During an export, when a connector (whether app, database, or file) receives data to export, such as user information, it receives Go types specific to the data types. This involves methods such as:

- [`Create`](./app/users-and-groups.md#create-records) and [`Update`](./app/users-and-groups.md#update-records) methods of apps take properties of a user or group.
- [`EventRequest`](./app/dispatch-events.md#dispatching-an-event) method of apps takes extra event information.
- [`Upsert`](database.md#upsert-method) method of databases takes rows to be added or updated.
- [`Write`](file.md#write-method) method of files takes a `RecordReader` whose `Record` method returns the next record to write.

The following table shows, for each data type, what Go type a connector should expect when it receives a value. For example, if a `DateTime` value is expected, then it will receive a `time.Time` type. If a `JSON` value is expected, then it will receive one of the types listed for `JSON`.

| Data Type       | Go Types                                                                                                                                                                                                           | Example                                                                                                                                                            |
|-----------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Boolean`       | `bool`                                                                                                                                                                                                             | `true`                                                                                                                                                             |
| `Int(n)`        | `int`                                                                                                                                                                                                              | `-583`                                                                                                                                                             |
| `Uint(n)`       | `uint`                                                                                                                                                                                                             | `1499`                                                                                                                                                             |
| `Float(n)`      | `float64`                                                                                                                                                                                                          | `21.907305`                                                                                                                                                        |
| `Decimal(p,s)`  | [`decimal.Decimal`](https://pkg.go.dev/github.com/shopspring/decimal#Decimal)                                                                                                                                      | `decimal.RequireFromString("-123.45")`                                                                                                                             |
| `DateTime` [^9] | [`time.Time`](https://pkg.go.dev/time#Time)                                                                                                                                                                        | `time.Date(2024, 04, 11, 16, 48, 26, 377105982, time.UTC)`                                                                                                         |
| `Date` [^10]    | [`time.Time`](https://pkg.go.dev/time#Time)                                                                                                                                                                        | `time.Date(2024, 04, 11, 0, 0, 0, 0, time.UTC)`                                                                                                                    |
| `Time` [^11]    | [`time.Time`](https://pkg.go.dev/time#Time)                                                                                                                                                                        | `time.Date(1970, 1, 1, 16, 48, 26, 377105982, time.UTC)`                                                                                                           |
| `Year`          | `int`                                                                                                                                                                                                              | `2024`                                                                                                                                                             |
| `UUID`          | `string`                                                                                                                                                                                                           | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`                                                                                                                           |
| `JSON` [^12]    | ◦ [`json.RawMessage`](https://pkg.go.dev/encoding/json#RawMessage)<br/>◦ `bool`<br/>◦ `string`<br/>◦ [`json.Number`](https://pkg.go.dev/encoding/json#Number)<br/>◦ `float64`<br/>◦ `map[string]any`<br/>◦ `[]any` | ```json.RawMessage(`{"points":561}`)```<br/>`true`<br/>`"gold"`<br/>`json.Number("23")`<br/>`377.402`<br/>`map[string]any{"points": 561}`<br/>`[]any{"gold", 855}` |
| `Inet`          | `string`                                                                                                                                                                                                           | `"192.168.1.1"`                                                                                                                                                    |
| `Text`          | `string`                                                                                                                                                                                                           | `"Emily Johnson"`                                                                                                                                                  |
| `Array(T)`      | `[]any`                                                                                                                                                                                                            | `[]any{618, 262, 791}`                                                                                                                                             |
| `Object`        | `map[string]any`                                                                                                                                                                                                   | `map[string]any{"plan": "Premium"}`                                                                                                                                | 
| `Map(T)`        | `map[string]any`                                                                                                                                                                                                   | `map[string]any{"1": "a", "2": "b", "others": "c"}`                                                                                                                |


[^9]: The location is always `time.UTC`.

[^10]: The time part is always zero and the location is always `time.UTC`.

[^11]: The date is always set to January 1, 1970, and the location is always `time.UTC`.

[^12]: `nil` represents a `nil` value, not the JSON `null`. A JSON `null` is represented by `json.RawMessage("null")`.

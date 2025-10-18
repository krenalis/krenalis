{% extends "/layouts/doc.html" %}
{% macro Title string %}Data Values{% end %}
{% Article %}

# Data values

When representing user and event data in a connector, these are mapped to specific Go types based on the data type. For example, in an export, a `datetime` value is represented in Go as `time.Time`, and a `text` value is represented as a `string`.

## Import

During an import, connectors return data to import, such as user information. This data must use specific types according to the data type. Key scenarios for handling data in an app connector include:

- [`Records`](apps/users#read-records) method of apps returns properties of users.
{#- `ReceiveWebhook` method of apps may return properties of users.#}
- [`Query`](databases#query-method) method of databases returns the resulting rows from a query.
- [`Read`](files#read-method) method of files takes a `RecordWriter` whose methods return the read records.

Connectors can directly return values deserialized from JSON using the [`json.Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal) function, because the types returned by this function are supported.

The table below shows, for each data type, which Go type a connector can return. Note that in the "Examples" column, the symbol "→" shows how a value returned by a connector is interpreted.

| Meergo type                                                                                         | Go type                                                                                                                                                                                                           | Example                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
|-----------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `boolean`                                                                                           | ◦`bool`<br/>◦`string`[^1]                                                                                                                                                                                         | `true`<br/>`"true"` → `true`, `"false"` → `false`                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `int(n)`<br/>`uint(n)`<br/>`float(n)`<br/>`decimal(p,s)`<br/>`year`                                 | ◦`int`<br/>◦`int8`<br/>◦`int16`<br/>◦`int32`<br/>◦`int64`<br/>◦`uint`<br/>◦`uint8`<br/>◦`uint16`<br/>◦`uint32`<br/>◦`uint64`<br/>◦`float32`<br/>◦`float64`<br/>◦`decimal.Decimal`<br/>◦`string`[^2]<br/>◦`[]byte` | `12`  // this and the following examples are for the `int(n)` data type<br/>`int8(23)` → 23<br/>`int16(512)` → 512<br/>`int32(-802)` → -802<br/>`int64(-1061274994)` → -1061274994<br/>`uint(3095322305)` → 3095322305<br/>`uint8(250)` → 250<br/>`uint16(12000)` → 12000<br/>`uint32(6038461)` → 6038461<br/>`uint64(612)` → 612<br/>`float32(51.0)` → 51.0<br/>`1048553.0` → 1048553<br/>`decimal.MustInt(932811)` → `932811`<br/>`"932811"` → `932811`<br/>`[]byte("932811")` → `932811` |
| `datetime`<div style="font-size:.85em; padding-left:6px;">(See [Time Layouts](#time-layouts))</div> | ◦[`time.Time`](https://pkg.go.dev/time#Time)<br/>◦`string`[^2]<br/>◦`float64`                                                                                                                                     | `time.Date(2024, 4, 11, 16, 48, 26, 377105982, time.UTC)`<br/>`"2024-04-11 15:59:05"`<br/>`1712851145.0`                                                                                                                                                                                                                                                                                                                                                                                    |
| `date`<div style="font-size:.85em; padding-left:6px;">(See [Time Layouts](#time-layouts))</div>     | ◦[`time.Time`](https://pkg.go.dev/time#Time)[^3]<br/>◦`string`[^2]                                                                                                                                                | `time.Date(2024, 4, 11, 0, 0, 0, 0, time.UTC)`<br/>`"2027-09-15"`                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `time`<div style="font-size:.85em; padding-left:6px;">(See [Time Layouts](#time-layouts))</div>     | ◦[`time.Time`](https://pkg.go.dev/time#Time)[^4]<br/>◦`string`[^2]<br/>◦`[]byte`                                                                                                                                  | `time.Date(1970, 1, 1, 16, 38, 50, 204813055, time.UTC)`<br/>`"16:38:50.204813055"`<br/>`[]byte("23:09:26.047126396")` → `time.Date(1970, 1, 1, 23, 9, 26, 47126396, time.UTC)`                                                                                                                                                                                                                                                                                                             |
| `uuid`[^5]                                                                                          | ◦`string`[^2]<br/>◦`[]byte`                                                                                                                                                                                       | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`<br/>`[]byte{211, 124, 89, 213, 136, 127, 68, 248, 143, 250, 126, 36, 49, 79, 71, 62}`                                                                                                                                                                                                                                                                                                                                                              |
| `json`[^6]                                                                                          | ◦`json.Value`<br/>◦[`json.Marshaler`](https://pkg.go.dev/encoding/json#Marshaler)[^json_marshaler]<br/>◦`string`[^2]<br/>◦`[]byte`                                                                                | ```json.Value(`{"p": 561}`)```<br/>[`json.RawMessage`](https://pkg.go.dev/encoding/json#RawMessage)(``` `{"p": 561}` ```) → ```json.Value(`{"p": 561}`)```<br/>`"[1, 2, 3]"` → ```json.Value(`[1, 2, 3]`)```<br/>```[]byte(`{"points": 5}`)``` → ```json.Value(`{"points": 5}`)```                                                                                                                                                                                                          |
| `inet`                                                                                              | ◦`string`[^2]<br/>◦[`net.IP`](https://pkg.go.dev/net#IP)<br/>◦[`netip.Addr`](https://pkg.go.dev/netip#Addr)                                                                                                       | `"192.168.1.1"`<br/>`net.ParseIP("192.168.1.1")`<br/>`netip.MustParseAddr("192.168.1.1")` → `"192.168.1.1"`                                                                                                                                                                                                                                                                                                                                                                                 |
| `text`                                                                                              | ◦`string`[^2]<br/>◦`[]byte`                                                                                                                                                                                       | `"Emily Johnson"`<br/>`[]byte("Emily Johnson") → "Emily Johnson"`                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `array(T)`                                                                                          | ◦`[]any`<br/>◦`[]T`<br/>◦`string`[^7]                                                                                                                                                                             | `[]any{618, 262, 791}`<br/>`[]int{19, -67, 8}` → `[]any{19, -67, 8}`<br/>`"[19,-67,8]"` → `[]any{19,  -67,  8}`                                                                                                                                                                                                                                                                                                                                                                             |
| `object`                                                                                            | `map[string]any`                                                                                                                                                                                                  | `map[string]any{"plan": "Premium"}`                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `map(T)`                                                                                            | ◦`map[string]any`<br/>◦`map[string]T`<br/>◦`string`[^8]                                                                                                                                                           | `map[string]any{"1": "a", "2": "b", "others": "c"}`<br/>`map[string]int{"a": 1, "b": 2}` → `map[string]any{"a": 1, "b": 2}`<br/>`"{"a":1,"b":2}"` → `map[string]any{"a": 1, "b": 2}`                                                                                                                                                                                                                                                                                                        |

[^1]: If the returned type is `string`, the only allowed values are `"true"` and `"false"`.

[^2]: An empty string is interpreted as `nil` for *nullable* `decimal`, `datetime`, `date`, `time`, `uuid`, `json`, and `inet` types, as well as for *nullable* `text` types restricted to specific values that do not include the empty string.

[^3]: The time part is not relevant and is not considered.

[^4]: The date part is not relevant and is not considered.

[^5]: The value must be in the UUID textual representation "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" as defined by RFC 9562.

[^6]: `json.Value`, `string`, and  `[]byte` values must contain valid JSON; the `MarshalJSON` method of the `json.Marshaler` interface must return valid JSON.

[^7]: The string must represent a JSON Array.

[^8]: The string must represent a JSON Object.

[^json_marshaler]: If the value is `nil`, the `MarshalJSON` method is still called, and it is expected to return the JSON `null`.

### Time layouts

By default, values of type `datetime`, `date`, and `time` returned as `string` are parsed using the ISO 8601 format. The same applies to values of type `time` returned as `[]byte`.

When registering the app connector, you can specify a different format if the values are not in ISO 8601, using any layout supported by the [time.Parse](https://pkg.go.dev/time#Parse) function:

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

#### Unix epoch

For the `TimeLayouts.DateTime` field of `meergo.AppInfo`, you can also use special layout strings: `"unix"`, `"unixmilli"`, `"unixmicro"`, or `"unixnano"`. With these layouts, values returned as `string` or `float64` are interpreted as the number of seconds, milliseconds, microseconds, or nanoseconds since the Unix epoch (January 1, 1970, 00:00:00 UTC).

## Export

During an export, when a connector (whether app, database, or file) receives data to export, such as user information, it receives Go types specific to the data types. This involves methods such as:

- [`PreviewSendEvents`](apps/send-events#send-events-1) method of apps takes extra event information.
- [`SendEvents`](apps/send-events#send-events-1) method of apps takes extra event information.
- [`Upsert`](apps/users#updating-and-creating-records) method of apps takes properties of a user.
- [`Merge`](databases#merge-method) method of databases takes rows to be added or updated.
- [`Write`](files#write-method) method of files takes a `RecordReader` whose `Record` method returns the next record to write.

The following table shows, for each data type, what Go type a connector should expect when it receives a value. In the "Examples" column, the symbol "→" illustrates how a connector should interpret a received value.

| Meergo type    | Go type                                          | Example                                                   |
|----------------|--------------------------------------------------|-----------------------------------------------------------|
| `boolean`      | `bool`                                           | `true`                                                    |
| `int(n)`       | `int`                                            | `-583`                                                    |
| `uint(n)`      | `uint`                                           | `1499`                                                    |
| `float(n)`     | `float64`                                        | `21.907305`                                               |
| `decimal(p,s)` | `decimal.Decimal`                                | `decimal.MustParse("-123.45")`                            |
| `datetime`     | [`time.Time`](https://pkg.go.dev/time#Time)[^9]  | `time.Date(2024, 4, 11, 16, 48, 26, 377105982, time.UTC)` |
| `date`         | [`time.Time`](https://pkg.go.dev/time#Time)[^10] | `time.Date(2024, 4, 11, 0, 0, 0, 0, time.UTC)`            |
| `time`         | [`time.Time`](https://pkg.go.dev/time#Time)[^11] | `time.Date(1970, 1, 1, 16, 48, 26, 377105982, time.UTC)`  |
| `year`         | `int`                                            | `2024`                                                    |
| `uuid`         | `string`                                         | `"0296345b-23f6-48b4-9547-ff83fb637d0e"`                  |
| `json`         | `json.Value`[^12]                                | ```json.Value(`{"p": 561}`)```                            |
| `inet`         | `string`                                         | `"192.168.1.1"`                                           |
| `text`         | `string`                                         | `"Emily Johnson"`                                         |
| `array(T)`     | `[]any`                                          | `[]any{618, 262, 791}`                                    |
| `object`       | `map[string]any`                                 | `map[string]any{"plan": "Premium"}`                       |
| `map(T)`       | `map[string]any`                                 | `map[string]any{"1": "a", "2": "b", "others": "c"}`       |


[^9]: The location is always `time.UTC`.

[^10]: The time part is always zero and the location is always `time.UTC`.

[^11]: The date is always set to January 1, 1970, and the location is always `time.UTC`.

[^12]: `nil` represents a `nil` value, not the JSON `null`. A JSON `null` is represented by `json.Value("null")`.

# Go values for property types

A value `v` of a concrete property type `t` has the Go representation specified below. All ranges are inclusive. The same rules apply recursively to array elements, object properties and map values.

## Kinds

| Kind                 | Go type           | Allowed values                                                                                                                                                                                                                              |
|----------------------|-------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `StringKind`         | `string`          | Any valid UTF-8 string. The type may restrict the set of allowed strings by enumeration, regular expression, maximum byte length or maximum number of Unicode code points.                                                                  |
| `BooleanKind`        | `bool`            | `false` or `true`.                                                                                                                                                                                                                          |
| `IntKind` (signed)   | `int`             | Integers in `t.IntRange()`. The Go type is `int` for every supported width (8, 16, 24, 32 and 64 bits).                                                                                                                                     |
| `IntKind` (unsigned) | `uint`            | Integers in `t.UnsignedRange()`. The Go type is `uint` for the same five widths.                                                                                                                                                            |
| `FloatKind`          | `float64`         | A finite value must be representable at the declared width (32 or 64 bits) and lie in `t.FloatRange()`. NaN is permitted when `t.IsReal()` is false. Each infinity is permitted when it lies in `t.FloatRange()` and `t.IsReal()` is false. |
| `DecimalKind`        | `decimal.Decimal` | Values exactly representable at `t.Precision()` and `t.Scale()` and within `t.DecimalRange()`. Precision is 1–76 and scale is 0–37, up to the precision.                                                                                    |
| `DateTimeKind`       | `time.Time`       | An instant with `v.Location() == time.UTC` and `types.MinYear <= v.Year() && v.Year() <= types.MaxYear` (years 1–9999).                                                                                                                     |
| `DateKind`           | `time.Time`       | A calendar date in years 1–9999 with location `time.UTC` and hour, minute, second and nanosecond all zero.                                                                                                                                  |
| `TimeKind`           | `time.Time`       | A time of day with nanosecond precision, location `time.UTC` and date 1970-01-01.                                                                                                                                                           |
| `UUIDKind`           | `string`          | A lowercase, hyphenated UUID in the form `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`, as returned by `types.NormalizeUUID`.                                                                                                                      |
| `JSONKind`           | `json.Value`      | One valid JSON value, including the JSON literal `null`.                                                                                                                                                                                    |
| `IPKind`             | `string`          | An IPv4 or IPv6 address in the form returned by `types.NormalizeIP`. IPv4-mapped IPv6 addresses are represented as the IPv4 address they map, such as `1.2.3.4`, and IPv6 addresses omit the zone.                                          |
| `ArrayKind`          | `[]any`           | Each element is a value of `t.Elem()`. Length is between `t.MinElements()` and `t.MaxElements()`. If `t.Unique()`, elements must be distinct according to `types.FirstDuplicate`.                                                           |
| `ObjectKind`         | `map[string]any`  | Each key names a declared property. Its value follows that property's type and nullability. Required keys depend on the operation.                                                                                                          |
| `MapKind`            | `map[string]any`  | Keys are valid UTF-8 strings and every value is a value of the same element type, `t.Elem()`.                                                                                                                                               |

`InvalidKind` has no values. Generic types are used only where their use is explicitly documented for that context. A generic type has the same representation as its resolved concrete type. The package names `decimal` and `json` refer to Krenalis's [tools/decimal](../decimal) and [tools/json](../json).

## Semantics

A type may have at most one of the semantics below. Its kind determines the Go representation, and all constraints on the underlying type still apply. With `NoSemantics`, the kind and its constraints fully determine the allowed values.

| Semantics (Go constant and JSON name)  | Underlying type and options                                                                                   | Allowed values                                                                                                                       |
|----------------------------------------|---------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------|
| `CountrySemantics` (`country`)         | Unconstrained `StringKind`. Requires `format` of `alpha-2` or `alpha-3`.                                      | An uppercase ISO 3166 country code in the declared format, such as `"IT"` or `"ITA"`. Current and formerly assigned codes are valid. |
| `DurationSemantics` (`duration`)       | `IntKind`, `DecimalKind`, or `FloatKind` with `real`. Requires a `unit` accepted by `Unit.IsDurationUnit`.    | A numeric duration expressed in the declared unit.                                                                                   |
| `EmailSemantics` (`email`)             | `StringKind`.                                                                                                 | A string representing an email address. The types package applies only the declared string constraints.                              |
| `MeasurementSemantics` (`measurement`) | `IntKind`, `DecimalKind`, or `FloatKind` with `real`. Requires a `unit` accepted by `Unit.IsMeasurementUnit`. | A numeric measurement expressed in the declared unit.                                                                                |
| `MoneySemantics` (`money`)             | `DecimalKind`. Optional ISO 4217 `currency` code.                                                             | A monetary amount in the declared currency, if specified.                                                                            |
| `PercentageSemantics` (`percentage`)   | `DecimalKind`.                                                                                                | A percentage in decimal form within the underlying decimal range. For example, `1` represents 100% and `0.25` represents 25%.        |
| `PhoneSemantics` (`phone`)             | Unconstrained `StringKind`.                                                                                   | A canonical E.164 phone number accepted by `types.IsPhone`.                                                                          |
| `URLSemantics` (`url`)                 | `StringKind`.                                                                                                 | A string representing a web URL. The types package applies only the declared string constraints.                                     |
| `YearSemantics` (`year`)               | `IntKind`.                                                                                                    | A year within the underlying signed or unsigned integer range.                                                                       |

For `country` and `phone`, an unconstrained string has no `maxBytes`, `maxLength`, `pattern` or `values` constraint.

## Absent properties and null values

An absent object property has no map entry. A map entry with the Go value `nil` represents an explicit null and is valid only for a nullable property. The JSON literal `null` is a value of `JSONKind`, represented by `json.Value("null")`. It is valid even for a required, non-nullable JSON property.

Array elements and map entries follow their element type. For example, `[]any{json.Value("null")}` is a value of `array(json)`, and `[]any{int(7)}` is a value of `array(int(32))`.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Data Types{% end %}
{% Article %}

# Data types

When defining data types and schemas in a connector, use the functions from the `types` package:

```go
"github.com/meergo/meergo/core/types"
```

This package provides functions to construct data types to use in connectors. For example, `types.Boolean()` returns a `Type` value representing the boolean type, and `types.Array(types.Int(32))` returns a `Type` value representing an array of 32-bit signed integers.

In connectors, data types need to be defined for:

- User, group, and event type schemas for [app connectors](./connectors/app).
- Query result column types for [database connectors](./connectors/database).
- Table column types for [database connectors](./connectors/database).
- File column types for [file connectors](./connectors/file).

## How to construct data types

Below are the data types and how to construct them using the `types` package functions.

- [text](#text) - text
- [boolean](#boolean) - boolean
- [int(n)](#intn) - signed integer
- [uint(n)](#uintn) - unsigned integer
- [float(n)](#floatn) - floating point number
- [decimal(p,s)](#decimalps) - decimal number
- [datetime](#datetime) - date and time
- [date](#date) - date
- [time](#time) - time
- [year](#year) - year
- [uuid](#uuid) - UUID
- [json](#json) - JSON
- [inet](#inet) - IP4 or IP6 address
- [array(T)](#arrayt) - array of T
- [object](#object) - object
- [map(T)](#mapt) - map of T

### text

Represents UTF-8 encoded text.

```go
types.Text()
```

Text can be limited by allowed values, a regular expression, or maximum lengths in bytes and characters, for example:

```go
// Text limited to specific values.
types.Text().WithValues("Hearts", "Diamonds", "Clubs", "Spades")

// Text matching a regular expression.
types.Text().WithRegexp(regexp.MustCompile(`\d+`))

// Text with a maximum length of 1000 bytes.
types.Text().WithByteLen(1000)

// Text with a maximum length of 2 characters.
types.Text().WithCharLen(2)
```

You can combine maximum byte and character lengths:

```go
// Text with a maximum of 25 bytes and 20 characters.
types.Text().WithByteLen(25).WithCharLen(20)
```

### boolean

Represents a boolean value.

```go
types.Boolean()
```

### int(n)

A signed integer with n bytes, where n can be 8, 16, 24, 32, or 64.

```go
types.Int(n)
```

You can limit integers to a minimum and maximum range, for example:

```go
// 8-bit signed integer with range [-20, 20].
types.Int(8).WithIntRange(-20, 20)

// 32-bit signed integer with range [0, 100].
types.Int(32).WithIntRange(0, 100)
```

### uint(n)

An unsigned integer with n bytes, where n can be 8, 16, 24, 32, or 64.

```go
types.Uint(n)
```

You can also set a minimum and maximum range for unsigned integers, for example:

```go
// 16-bit unsigned integer with range [0, 100].
types.Uint(16).WithUintRange(0, 100)

// 32-bit unsigned integer with range [1000, 2000].
types.Uint(32).WithUintRange(1000, 2000)
```

### float(n)

A floating-point number with n bytes, n can be 32, or 64. Includes `+Inf`, `-Inf`, and `NaN` values.

```go
types.Float(n)
```

Floats can have a minimum and maximum value and can be restricted to real numbers only, for example:

```go
// 32-bit float with range [-20.5, 8].
types.Float(32).WithFloatRange(-20.5, 8)

// 64-bit float with range [0, 56.481782].
types.Float(64).WithFloatRange(0, 56.481782)

// 64-bit float, excluding +Inf, -Inf, and NaN.
types.Float(64).AsReal()
```

### decimal(p,s)

A decimal number with precision `p` and scale `s`, where `p` ranges from 1 to 76, `s` from 0 to 37, and `s` is less than or equal to `p`.

```go
types.Decimal(p, s)
```

Decimals can also have a minimum and maximum value range, for example:

```go
import "github.com/meergo/meergo/core/decimal"

...

min := decimal.MustInt(-10.5)
max := decimal.MustInt(8.25)

// Decimal with precision 5 and scale 2, range [-10.5, 8.25].
types.Decimal(5, 2).WithDecimalRange(min, max)
```

### datetime

Represents a date and time within the range [1, 9999] with nanosecond precision and no timezone.

```go
types.DateTime()
```

### date

Represents a date within the range [1, 9999].

```go
types.Date()
```

### time

Represents a time of day with nanosecond precision and no timezone.

```go
types.Time()
```

### year

Represents a year within the range [1, 9999].

```go
types.Year()
```

### uuid

Represents a UUID.

```go
types.UUID()
```

### json

Represents JSON data.

```go
types.JSON()
```

### inet

Represents an IP4 or IP6 address.

```go
types.Inet()
```

### array(T)

Represents an array of elements of type `T`.

```go
types.Array(T)
```

For example:

```go
// Array of text strings.
types.Array(types.Text())

// Array of 16-bit signed integers.
types.Array(types.Int(16))

// Array of UUID arrays.
types.Array(types.Array(types.UUID()))
```

Arrays can be limited in the minimum and maximum number of elements:

```go
// Array of 32-bit unsigned integers with at least 1 element.
types.Array(types.Int(32)).WithMinElements(1)

// Array with a maximum of 10 decimal values.
types.Array(types.Decimal(10, 2)).WithMaxElements(10)

// Array with 5 to 15 text values.
types.Array(types.Text()).WithMinElements(5).WithMaxElements(15)
```

Arrays can also be constrained to have unique values for their elements, except for arrays of `json`, `array`, `map`, and `object`:

```go
// Array of 64-bit signed integers with unique values.
types.Array(types.Uint64()).WithUnique()

// Array of UUIDs with unique values.
types.Array(types.UUID()).WithUnique()
```

### object

Represents an object with specified properties.

```go
types.Object([]types.Property{...})
```

For example:

```go
types.Object([]types.Property{
    {Name: "first_name", Type: types.Text().WithCharLen(30)},
    {Name: "last_name", Type: types.Text().WithCharLen(30)},
    {Name: "birth_date", Type: types.Year()},
})
```

You can also use the `types.ObjectOf` function to construct an `object`. Unlike `types.Object`, it does not panic if a property is invalid but returns an error instead:

```go
typ, err := types.ObjectOf([]types.Property{...})
if err != nil {
    ...
}
```

#### Properties

An object property is defined as follows:

```go
type Property struct {
    Name           string
    Prefilled      string
    Type           Type
    CreateRequired bool
    UpdateRequired bool
    ReadOptional   bool
    Nullable       bool
    Description    string
}
```

* `Name`: The name of the property. It must start with a letter `[A-Za-z_]` and can only contain alphanumeric characters and underscores `[A-Za-z0-9_]` after that. To check if a name is valid, use the `types.IsValidPropertyName` function.
* `Prefilled`: A prefilled value to use in transformation mappings for events sent to applications. It pre-fills the input with the expression that evaluates to the property's value.
* `Type`: The type of the property, which can be any [data type](#how-to-construct-data-types).
* `CreateRequired`: Indicates whether the property is required for creation.
* `UpdateRequired`: Indicates whether the property is required for the update.
* `ReadOptional`: Indicates whether the property may not be present when reading.
* `Nullable`: Indicates whether the property can be null. In Go, this means it can be `nil`. In JavaScript, it can be `null`, and in Python, it can be `None`.
* `Description`: A description providing additional information about the property's usage.

### map(T)

Represents a map with `text` keys and values of type `T`.

```go
types.Map(T)
```

For example:

```go
// Map with 64-bit signed integers as values.
types.Map(types.Int(64))

// Map with arrays of texts as values.
types.Map(types.Array(types.Text()))
```

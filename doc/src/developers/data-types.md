# Data Types

When defining data types and schemas in a connector, use the functions from the `types` package:

```go
"github.com/open2b/chichi/types"
```

This package provides functions to construct data types to use in connectors. For example, `types.Boolean()` returns a `Type` value representing the boolean type, and `types.Array(types.Int(32))` returns a `Type` value representing an array of 32-bit signed integers.

In connectors, data types need to be defined for:

- User, group, and event type schemas for [app connectors](./app.md).
- Query result column types for [database connectors](./database.md).
- Table column types for [database connectors](./database.md).
- File column types for [file connectors](./file.md).

## How to Construct Data Types

Below are the data types and how to construct them using the `types` package functions.

- [Boolean](#boolean) - boolean
- [Int(n)](#intn) - signed integer
- [Uint(n)](#uintn) - unsigned integer
- [Float(n)](#floatn) - floating point number
- [Decimal(p,s)](#decimalps) - decimal number
- [DateTime](#datetime) - date and time
- [Date](#date) - date
- [Time](#time) - time
- [Year](#year) - year
- [UUID](#uuid) - UUID
- [JSON](#json) - JSON
- [Inet](#inet) - IP4 or IP6 address
- [Text](#text) - text
- [Array(T)](#arrayt) - array of T
- [Object](#object) - object
- [Map(T)](#mapt) - map of T

### Boolean

Represents a boolean value.

```go
types.Boolean()
```

### Int(n)

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

### Uint(n)

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

### Float(n)

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

### Decimal(p,s)

A decimal number with precision `p` and scale `s`, where `p` ranges from 1 to 76, `s` from 0 to 37, and `s` is less than or equal to `p`.

```go
types.Decimal(p, s)
```

Decimals can also have a minimum and maximum value range, for example:

```go
import "github.com/shopspring/decimal"

...

min := decimal.NewFromInt(-10.5)
max := decimal.NewFromInt(8.25)

// Decimal with precision 5 and scale 2, range [-10.5, 8.25].
types.Decimal(5, 2).WithDecimalRange(min, max)
```

### DateTime

Represents a date and time within the range [1, 9999] with nanosecond precision and no timezone.

```go
types.DateTime()
```

### Date

Represents a date within the range [1, 9999].

```go
types.Date()
```



### Time

Represents a time of day with nanosecond precision and no timezone.

```go
types.Time()
```

### Year

Represents a year within the range [1, 9999].

```go
types.Year()
```

### UUID

Represents a UUID.

```go
types.UUID()
```

### JSON

Represents JSON data.

```go
types.JSON()
```

### Inet

Represents an IP4 or IP6 address.

```go
types.Inet()
```

### Text

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

### Array(T)

Represents an array of items of type `T`.

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

Arrays can be limited in the minimum and maximum number of items:

```go
// Array of 32-bit unsigned integers with at least 1 item.
types.Array(types.Int(32)).WithMinItems(1)

// Array with a maximum of 10 decimal values.
types.Array(types.Decimal(10, 2)).WithMaxItems(10)

// Array with 5 to 15 text values.
types.Array(types.Text()).WithMinItems(5).WithMaxItems(15)
```

### Object

Represents an object with specified properties.

```go
types.Object([]types.Property{...})
```

For example:

```go
types.Object([]types.Property{
	{Name: "first_name", Label: "First name", Type: types.Text().WithCharLen(30)},
    {Name: "last_name", Label: "Last name", Type: types.Text().WithCharLen(30)},
    {Name: "birth_date", Label: "Date of birth", Type: types.Year()},
})
```

> Properties are described in detail in the dedicated Properties section. (TODO)

You can also use the `types.ObjectOf` function to construct an `Object`. Unlike `types.Object`, it does not panic if a property is invalid but returns an error instead:

```go
typ, err := types.ObjectOf([]types.Property{...})
if err != nil {
	...
}
```

### Map(T)

Represents a map with `Text` keys and values of type `T`.

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

## How to Represents Data Values
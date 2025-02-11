{% extends "/layouts/doc.html" %}
{% macro Title string %}Schemas{% end %}
{% Article %}

# Schemas

Schemas are used by multiple endpoints to define the structure of users, events, and the source and destination schemas for apps, databases, and files.

Below is an example of a user schema with the properties: `email`, `date_of_birth` e `ip_addresses`. 

```json
{
  "kind": "object",
  "properties": [
    {
      "name": "email",
      "type": {
        "kind": "string",
        "charLen": 120
      },
      "readOptional": true
    },
    {
      "name": "date_of_birth",
      "type": { "kind": "date" },
      "readOptional": true
    },
    {
      "name": "ip_addresses",
      "type": {
        "kind": "array",
        "elementType": { "kind": "inet" }
      },
      "readOptional": true
    }
  ]
}
```

Note that a schema has the same representation as an [object type](#object). 

## Data types

Below are the data types and their representations in JSON format:

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
- [text](#text) - text
- [array(T)](#arrayt) - array of T
- [object](#object) - object
- [map(T)](#mapt) - map of T

### boolean

Represents a boolean value.

```json
{
  "kind": "boolean"
}
```

Values are `true` and `false`.

### int(n)

A signed integer with n bytes, where n can be 8, 16, 24, 32, or 64.

```json
{
  "kind": "int",
  "bitSize": 32
}
```

You can limit integers to a minimum and maximum range, for example the following defines an 8-bit integer with range [-20, 20]:

```json
{
  "kind": "int",
  "bitSize": 8,
  "minimum": -20,
  "maximum": 20
}
```

Examples of values are `12`, `-470`, `7308561`.

### uint(n)

An unsigned integer with n bytes, where n can be 8, 16, 24, 32, or 64.

```json
{
  "kind": "uint",
  "bitSize": 32
}
```

You can also set a minimum and maximum range for unsigned integers, for example the following defines a 32-bit integer with range [1000, 2000]:

```json
{
  "kind": "uint",
  "bitSize": 32,
  "minimum": 1000,
  "maximum": 2000
}
```

Examples of values are `63`, `0`, `947165402`.

### float(n)

A floating-point number with n bytes, n can be 32, or 64. Includes +Inf, -Inf, and NaN values.

```json
{
  "kind": "float",
  "bitSize": 64
}
```

Floats can have a minimum and maximum value and can be restricted to real numbers only.

For example a 32-bit float with range [-20.5, 8]:

```json
{
  "kind": "float",
  "bitSize": 32,
  "minimum": -20.5,
  "maximum": 8
}
```

For example a 64-bit float with range [-0, 56.481782]:

```json
{
  "kind": "float",
  "bitSize": 64,
  "minimum": 0,
  "maximum": 56.481782
}
```

For example a 64-bit float, excluding +Inf, -Inf, and NaN:

```json
{
  "kind": "float",
  "bitSize": 64,
  "real": true
}
```

Examples of values are `1.6892`, `-0.002516`, `441.015`.

### decimal(p,s)

A decimal number with a precision p and a scale s, where precision ranges from 1 to 76, scale from 0 to 37, and scale is less than or equal to precision.

```json
{
  "kind": "decimal",
  "precision": 10,
  "scale": 2
}
```

Decimals can also have a minimum and maximum value range, for example the following is a decimal with precision 5 and scale 2, range [-10.5, 8.25].

```json
{
  "kind": "decimal",
  "precision": 5,
  "scale": 2,
  "minimum": -10.5,
  "maximum": 8.25
}
```

Examples of values are `5.12`, `893`, `1258.068`.

### datetime

Represents a date and time within the range [1, 9999] with nanosecond precision and no timezone. Values are presented in ISO 8601 format.

```json
{
  "kind": "datetime"
}
```

Examples of values are `"2025-02-11T16:19:14.820364510Z"`, `"1975-12-05T09:55:12.048522068Z"`.

### date

Represents a date within the range [1, 9999]. Values are presented in ISO 8601 format.

```json
{
  "kind": "date"
}
```

Examples of values are `"2025-02-11"`, `"1975-12-05"`.

### time

Represents a time of day with nanosecond precision and no timezone. Values are presented in ISO 8601 format.

```json
{
  "kind": "time"
}
```

Examples of values are `"16:19:14.820364510"`, `"09:55:12.048522068"`.

### year

Represents a year within the range [1, 9999].

```json
{
  "kind": "year"
}
```

Examples of values are `2025`, `1975`.

### uuid

Represents a UUID.

```json
{
  "kind": "uuid"
}
```

Examples of values are `"ae3a0552-eff9-4456-8a2e-b94c64d03359"`, `"bbe0eda7-b672-4cfb-9285-1128681e00cd"`.

### json

Represents JSON data.

```json
{
  "kind": "json"
}
```

Examples of values are `"null"`, `"5"` `"\"hello"\"`, `"{\"name\":\"John\"}"`, `"true"`, `"[34,12,0,6]"`.

### inet

Represents an IP4 or IP6 address.

```json
{
  "kind": "inet"
}
```

Examples of values are `"192.0.2.1"`, `"2001:db8::1"`.

### text

Represents UTF-8 encoded text.

```json
{
  "kind": "text"
}
```

Text can be limited by allowed values, a regular expression, or maximum lengths in bytes and characters.

For example a text limited to specific values: 

```json
{
  "kind": "text",
  "values": [ "Hearts", "Diamonds", "Clubs", "Spades" ] 
}
```

For example a text matching a regular expression: 

```json
{
  "kind": "text",
  "regexp": "\\d+" 
}
```

Regular expression syntax is the same as the [Go syntax](https://pkg.go.dev/regexp/syntax).

For example a text with a maximum length of 1000 bytes:

```json
{
  "kind": "text",
  "byteLen": 1000 
}
```

For example a text with a maximum length of 2 characters:

```json
{
  "kind": "text",
  "charLen": 2 
}
```

You can combine maximum byte and character lengths. For example a text with a maximum of 25 bytes and 20 characters: 


```json
{
  "kind": "text",
  "byteLen": 25,
  "charLen": 20 
}
```

Examples of values are `"Everett Hayes"`, `"(555) 123-4567"`.

### array(T)

Represents an array of elements of type T. For example an array of text:

```json
{
  "kind": "array",
  "elementType": { "kind": "text" }
}
```

Arrays can be limited in the minimum and maximum number of elements. For example an array of 32-bit unsigned integers with at least 1 element:

```json
{
  "kind": "array",
  "elementType": {
    "kind": "int",
    "bitSize": 32
  },
  "minElements": 1
}
```

For example an array with a maximum of 10 decimal values: 

```json
{
  "kind": "array",
  "elementType": {
    "kind": "decimal",
    "precision": 10,
    "scale": 2
  },
  "maxElements": 10
}
```

For example an array with 5 to 15 text values.

```json
{
  "kind": "array",
  "elementType": { "kind": "text" },
  "minElements": 5,
  "maxElements": 15
}
```

Arrays can also be constrained to have unique values for their elements, except for arrays of json, array, map, and object.

For example an array of 64-bit signed integers with unique values: 

```json
{
  "kind": "array",
  "elementType": {
    "kind": "uint",
    "bitSize": 64
  },
  "uniqueElements": true
}
```

For example an array of UUIDs with unique values: 

```json
{
  "kind": "array",
  "elementType": { "kind": "uuid" },
  "uniqueElements": true
}
```

Examples of values are `[8498, 204, 7531]`, `[{"name": "John"}, {"name": "Emily"}]`.

### object

Represents an object with specified properties. For example:

```json
{
  "kind": "object",
  "properties": [
    {
      "name": "first_name",
      "type": {
        "kind": "text",
        "charLen": 30
      }
    },
    {
      "name": "last_name",
      "type": {
        "kind": "text",
        "charLen": 30
      }
    },
    {
      "name": "birth_date",
      "type": { "kind": "date" }
    }
  ]
}
```

Examples of values are `{"first_name": "John", "last_name": "Hollis", "age": 34, active: true}`, `{"city": "New York", "street": "5th Avenue", "zip": 10001}`.

#### Properties

An object property has the following keys:

| Key               | Type      | Required | Default | Description                                                                                                                                                         |
|-------------------|-----------|----------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `name`            | `String`  |    ✓     |         | The name of the property. It must start with a letter `[A-Za-z_]` and can only contain alphanumeric characters and underscores `[A-Za-z0-9_]` after that.           |
| `placeholder`     | `Number`  |          | `""`    | A placeholder to use in transformation mappings for events sent to applications. It pre-fills the input with the expression that evaluates to the property's value. |
| `type`            | `Object`  |    ✓     |         | The type of the property, which can be any [data type](#data-types).                                                                                                |
| `createRequired`  | `Boolean` |          | `false` | Indicates whether the property is required during creation, i.e., whether a value for the property is required at the time of creation.                             |
| `updateRequired`  | `Boolean` |          | `false` | Indicates whether the property is required for updating, i.e., whether a value for the property is mandatory when updating an existing record.                      |
| `readOptional`    | `Boolean` |          | `false` | Indicates whether the property may not be present when reading, i.e., whether the property is optional and may not be included in the data when retrieved.          |
| `nullable`        | `Boolean` |          | `false` | Indicates whether the property can be `null`.                                                                                                                       |
| `description`     | `String`  |          | `""`    | A description providing additional information about the property's usage.                                                                                          |

### map(T)

Represents a map with text keys and values of type T. For example a map of text:

```json
{
  "kind": "map",
  "elementType": { "kind": "text" }
}
```

For example a map with 64-bit signed integers as values:

```json
{
  "kind": "map",
  "elementType": {
    "kind": "int",
    "bitSize": 64
  }
}
```

For example a map with arrays of texts as values: 

```json
{
  "kind": "map",
  "elementType": {
    "kind": "array",
    "elementType": { "kind": "text" }
  }
}
```

Examples of values are `{"score": "6205", "player": "Everett Hayes"}`, `{"width": 1920, "height": 1080}`.


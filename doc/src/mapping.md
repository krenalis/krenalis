{% extends "/layouts/doc.html" %}
{% macro Title string %}Mapping{% end %}
{% Article %}

# Mapping

Mapping serves as an effortlessly simple data transformation method, designed with a focus on simplicity. It doesn't demand additional dependencies or mastery of programming languages like JavaScript or Python

It's the quickest and most efficient method because it doesn't need to call external functions. Instead, it uses a fast expression engine built into Meergo.

However, for much more complex use cases, a more powerful language is recommended, such as JavaScript and Python.

## Map properties

For each output property, you can provide an expression whose evaluation provides its value. In the expression you can refer the input properties.

As an example, the following mapping maps the `firstName` and `lastName` properties from the input to the `first_name` and `last_name` properties of the output, respectively. The output property `email` remains unmapped.

```
┌─────────────────────────────────┐
│ firstName                       │ ->  first_name
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ lastName                        │ ->  last_name
└─────────────────────────────────┘
┌─────────────────────────────────┐
│                                 │ ->  email
└─────────────────────────────────┘
```

With this mapping, the input:
```
firstName: "Emma"
lastName: "Johnson"
email: "emma.johnson@example.com"
```
will become:
```
first_name: "Emma"
last_name: "Johnson"
```

If a property is not mapped, its value remains unchanged. If you want to map an output property only in some cases, use the [when](#when-function) function.

### Constants

You can use the constant `null` and constant values such as strings, numbers, and booleans:
```
┌─────────────────────────────────┐
│ null                            │ ->  birthdate
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ "on"                            │ ->  status
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ 50                              │ ->  score
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ true                            │ ->  active
└─────────────────────────────────┘
```

Strings can also be written with single quotes:
```
┌─────────────────────────────────┐
│ 'on'                            │ ->  status
└─────────────────────────────────┘
```

To include a single or double quote within a string, simply prefix the quote with a backslash:
```
┌─────────────────────────────────┐
│ 'O\'Connor'                     │ ->  last_name
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ "123 Main Street, \"Apt 4B\""   │ ->  street
└─────────────────────────────────┘
```

### Concatenation

Properties, strings, numbers and booleans can be concatenated by writing them one after the other.

```
┌─────────────────────────────────┐
│ firstName " " lastName          │ ->  full_name
└─────────────────────────────────┘
```

With this mapping, the input:
```
firstName: "Emma"
lastName: "Johnson"
```
will become:
```
full_name: "Emma Johnson"
```

### Sub-properties, map keys, and JSON object keys

You can reference sub-properties, map keys, and JSON object keys in expressions using either dot notation or square brackets:

```
┌─────────────────────────────────┐
│ address.city                    │ ->  city
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ properties["birth day"]         │ ->  birth_day
└─────────────────────────────────┘
```

Accessing a non-existent property causes a compile error. However, trying to access a map key or a JSON object key that doesn't exist will result in `null`. If the result of the map expression is `null`, but the output property cannot be `null`, the output property will be missing from the mapping result. For example, consider the `traits` property with the following JSON value:

```json
{
    "address": {
        "city": "Milan"
    },
    "phone": "+39 02 12345678"
}
```

Here, `traits.address` evaluates to the JSON value `{"city":"Milan"}`. However, `traits.name` evaluates to `null` since `name` is not a property of `traits`. Therefore, in the following mapping expression:

```
┌─────────────────────────────────┐
│ traits.name                     │ ->  first_name
└─────────────────────────────────┘
```

the `first_name` output property will be missing from the mapping result if it cannot be `null`. As a special case, if the output property is of type `json` and cannot be `null`, it will be present with the JSON null value.

Attempting to access a non-object JSON value as if it were an object results in an error, causing the entire mapping to fail. For example, the following mapping will return an error because `traits.phone` is not a JSON object:

```
┌─────────────────────────────────┐
│ traits.phone.mobile             │ ->  phoneNumber
└─────────────────────────────────┘
```

To avoid this error, append a `?` after the key. This prevents the error if the accessed JSON value is not an object and evaluates to `null`. The following mapping expression will be evaluated as `null`:

```
┌─────────────────────────────────┐
│ traits.phone.mobile?            │ ->  phoneNumber
└─────────────────────────────────┘
```

### Functions

In expressions, you can use some functions to do slightly more complex tasks.

For instance, consider a scenario where the input property `maritalStatus` is `null`, indicating an unknown marital status. You must map this property to the non-nullable output property `marital_status`. To address this situation, you could use the `coalesce` function, which returns the first non-null argument:
```
┌─────────────────────────────────────┐
│ coalesce(maritalStatus, "Unknown")  │ ->  marital_status
└─────────────────────────────────────┘
```

With this mapping, the input:
```
maritalStatus: null
```
will become:
```
marital_status: "Unknown"
```

Below is a list of available functions:

#### **and** function

The `and` function returns `true` only when all of its arguments are `true`; otherwise, it returns `false`. For example:
```
┌─────────────────────────────────┐
│ and(active, newsletterConsent)  │ ->  marketing_consent
└─────────────────────────────────┘
```

It returns `null`, if an argument is `null` and there are no `false` arguments. For example:
```
and(true, true)   -> true
and(true, false)  -> false
and(false, false) -> false
and(null, true)   -> null
and(null, false)  -> false
```

The arguments of the `and` function should have type `boolean`, and the result has type `boolean`.

#### **array** function

The `array` function returns an array with the passed arguments as elements. For example:
```
┌─────────────────────────────────┐
│ array(email, company.email)     │ ->  emails
└─────────────────────────────────┘
```

The result of the `array` function has type `array(json)`.

#### **coalesce** function

The `coalesce` function returns the first non-null argument, or `null` if all arguments are `null`.  For example:

```
┌───────────────────────────────────────────┐
│ coalesce(shippingAddress, billingAddress) │ ->  ship_address
└───────────────────────────────────────────┘
```

For example:
```
coalesce(null, 0)    -> 0
coalesce(null, null) -> null
```

The result of the `coalesce` function has type `json`.

#### **eq** function

The `eq` function takes two values and returns `true` if they are equal; otherwise, it returns `false`. For example:
```
┌─────────────────────────────────┐
│ eq(level, "VIP")                │ ->  vip_customer
└─────────────────────────────────┘
```

If an argument is `null`, the function returns `null`. For example:
```
eq(5, 5)      -> true
eq('a', 'b')  -> false
eq('a', 5)    -> false
eq('a', null) -> null
```

The result of the `eq` function has type `boolean`.

#### **if** function

The `if` function evaluates the first boolean argument. If it is `true`, the function returns the second argument. If the first argument is `false` or `null`, the function returns the third argument. For example, if `hasCode` is `true`, the following code evaluates to the value of `code`; otherwise, it evaluates to an empty `text` value.
```
┌─────────────────────────────────┐
│ if(hasCode, code, '')           │ ->  code
└─────────────────────────────────┘
```

For example:
```
if(true, 1, 2)      -> 1
if(false, 'a', 'b') -> 'b'
if(null, 7.5, 9.8)  -> 9.8
```

The `if` function can also be called with only two arguments. In this case, if the first argument is not `true`, the result is `null`.
```
┌─────────────────────────────────┐
│ if(hasCode, code)               │ ->  code
└─────────────────────────────────┘
```

For example:

```
if(true, 5)  -> 5
if(false, 5) -> null
if(null, 5)  -> null
```

The result of the `if` function has type `json`.

#### **initcap** function

The `initcap` function returns its argument with the first letter of each word in uppercase, all other letters in lowercase. For example:
```
┌─────────────────────────────────┐
│ initcap(firstName ' ' lastName) │ ->  full_name
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
initcap('emily johnson')  -> 'Emily Johnson'
initcap('john o\'connor') -> 'John O\'Connor'
initcap('NEW YORK')       -> 'New York'
initcap(null)             -> null
```

The argument of the `initcap` function should have type `text`, and the result has type `text`.

#### **json_parse** function

The `json_parse` function parses its argument as JSON and returns the corresponding `json` value. For example:
```
┌──────────────────────────────────────────────┐
│ json_parse('{"city":"Milan","zip":"20100"}') │ ->  address
└──────────────────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
json_parse('"Android"') -> "Android" as JSON
json_parse('true')      -> true      as JSON
json_parse('[1, 2, 3]') -> [1,2,3]   as JSON
json_parse('null')      -> null      as JSON
json_parse(null)        -> null
```

If the input is not valid JSON, `json_parse` will produce an error, causing the entire mapping to fail.

The argument of the `json_parse` function should have type `text` or `json`. If the argument has type `json`, the value must be a JSON string. The result has type `json`.

#### **len** function

The `len` function returns the length of the given argument based on its type.
```
┌─────────────────────────────────┐
│ len(values)                     │ ->  num_values
└─────────────────────────────────┘
```

The length is determined as follows:

* `text`: Returns the length in characters.
* `array`: Returns the number of elements.
* `map`: Returns the number of key-value pairs.
* `object`: Returns the number of present properties.
* `boolean`, `int`, `uint`, `float`, `decimal`, `datetime`, `date`, `time`, `year`, `uuid`, `inet`: Returns the length of its string representation.

For `json` values:
 
* JSON string: Returns the length in characters.
* JSON object: Returns the number of key-value pairs.
* JSON array: Returns the number of elements.
* JSON boolean: Returns the length of its string representation.
* JSON number: Returns the length of its string representation.
* JSON null: Returns `0`.

If the argument is `null`, the function also returns `0`. For example:
```
len(null) -> 0
len(true) -> 4
len(false) -> 5
len('Seattle') -> 7
len('東京') -> 2
len(array(1.5, 3.9, 0.15)) -> 3
len(520945) -> 6
len(-722) -> 4
len(88.0300) -> 5
```

The result of the `len` function has type `int(32)`.

#### **lower** function

The `lower` function returns its argument with all letters in lower case. For example:
```
┌─────────────────────────────────┐
│ lower(address.city)             │ ->  city
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
lower("aBc") -> "abc"
lower(null)  -> null
```

The argument of the `lower` function should have type `text`, and the result has type `text`.

#### **ltrim** function

The `ltrim` function returns its argument with all leading Unicode whitespace removed. For example:
```
┌─────────────────────────────────┐
│ ltrim(email)                    │ ->  email
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
ltrim("  info@example.com") -> "info@example.com"
ltrim("\t   hello world\n") -> "hello world\n"
ltrim(null)  -> null
```

Both the argument and the result of the  `trim` function have type `text`.

#### **ne** function

The `ne` function takes two values and returns `true` if they are not equal; otherwise, it returns `false`. For example:
```
┌─────────────────────────────────┐
│ ne(status, "inactive")          │ ->  is_active
└─────────────────────────────────┘
```

If an argument is `null`, the function returns `null`. For example:
```
ne(5, 5)      -> false
ne('a', 'b')  -> true
ne('a', 5)    -> true
ne('a', null) -> null
```

The result of the `eq` function has type `boolean`.

#### **not** function

The `not` function returns `false` if its argument is `true`, and `true` if its argument is `false`. For example:
```
┌─────────────────────────────────┐
│ not(traits.active)              │ ->  disactive
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
not(true)  -> false
not(false) -> true
not(null)  -> null
```

The argument of the `not` function should have type `boolean`, and the result has type `boolean`.

#### **or** function

The `or` function returns `true` if at least one of its arguments is `true`; otherwise, it returns `false`. For example:
```
┌─────────────────────────────────┐
| or(hasDriverLicense, age18)     │ ->  eligibility
└─────────────────────────────────┘
```

It returns `null`, if an argument is `null` and there are no `true` arguments. For example:
```
or(true, true)   -> true
or(true, false)  -> true
or(false, false) -> false
or(null, true)   -> true
or(null, false)  -> null
```

The arguments of the `or` function should have type `boolean`, and the result has type `boolean`.

#### **rtrim** function

The `rtrim` function returns its argument with all trailing Unicode whitespace removed. For example:
```
┌─────────────────────────────────┐
│ rtrim(email)                    │ ->  email
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
rtrim("info@example.com\n") -> "info@example.com"
rtrim("\t   hello world\n") -> "\t   hello world"
rtrim(null)  -> null
```

Both the argument and the result of the `rtrim` function have type `text`.

#### **substring** function

The `substring` function extracts a portion of a string based on a specified starting position and length. The indices are 1-based, meaning the first character of the string has an index of 1.

```
┌─────────────────────────────────┐
│ substring(traits.phone, 6, 12)  │ ->  phone
└─────────────────────────────────┘
```

The syntax is `substring(s, start, length)`, where:

- `s` is the input string from which you want to extract a substring.
- `start` is the position in the string where extraction begins, with the first character at position 1.
- `length` is the number of characters to extract from the starting position. If omitted, the function will extract the substring from `start` to the end of the string.

For example:
```
substring('Alice Johnson', 0, 5)  -> 'Alice'
substring('Alice Johnson', 1, 5)  -> 'Alice'
substring('Alice Johnson', 7)     -> 'Johnson'
substring('Alice Johnson', 7, 6)  -> 'Johnson'
substring('Alice Johnson', 7, 20) -> 'Johnson'
substring('Alice Johnson', 20, 5) -> '' 
substring(null, 1, 5)             -> null
```

Note:

- If any argument is `null`, the function returns `null`.
- If the starting position `start` exceeds the length of the string `s`, the result is an empty string.
- The length `length` can be zero but cannot be negative.
- When `length` is greater than the number of characters remaining from `start` to the end of the string, the function returns the substring from `start` to the end of the string.

The `start` and `length` arguments should be of type `integer`, and the `s` argument and the result are of type `text`.

#### **trim** function

The `trim` function returns its argument with all leading and trailing Unicode whitespace removed. For example:
```
┌─────────────────────────────────┐
│ trim(email)                     │ ->  email
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
trim("  info@example.com ") -> "info@example.com"
trim("\t   hello world\n") -> "hello world"
trim(null)  -> null
```

Both the argument and the result of the `trim` function have type `text`.

#### **upper** function

The `upper` function returns its argument with all letters in upper case. For example:
```
┌─────────────────────────────────┐
│ upper(address.countryCode)      │ ->  country
└─────────────────────────────────┘
```

If its argument is `null`, the function returns `null`. For example:
```
upper("usa") -> "USA"
upper(null)  -> null
```

The argument of the `upper` function should have type `text`, and the result has type `text`.

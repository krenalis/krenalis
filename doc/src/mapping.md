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

### Sub-properties, Map Keys, and JSON Object Keys

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

the `first_name` output property will be missing from the mapping result if it cannot be `null`. As a special case, if the output property is of type `JSON` and cannot be `null`, it will be present with the JSON null value.

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
and(true, true)  -> true
and(true, false) -> false
and(true, null)  -> null
and(false, null) -> false
```

The arguments of the `and` function should have type `Boolean`, and the result has type `Boolean`.

#### **array** function

The `array` function returns an array with the passed arguments as elements. For example:
```
┌─────────────────────────────────┐
│ array(email, company.email)     │ ->  emails
└─────────────────────────────────┘
```

The result of the `array` function has type `Array(JSON)`.

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

The result of the `coalesce` function has type `JSON`.

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

The result of the `eq` function has type `Boolean`.

#### **if** function

The `if` function evaluates the first boolean argument. If it is `true`, the function returns the second argument. If the first argument is `false` or `null`, the function returns the third argument. For example, if `hasCode` is `true`, the following code evaluates to the value of `code`; otherwise, it evaluates to an empty `Text` value.
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

The result of the `if` function has type `JSON`.

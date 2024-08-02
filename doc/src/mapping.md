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

### Sub-properties, map keys and JSON Object keys


In expressions, you can reference sub-properties, map keys, and JSON object keys using either a dot or square brackets:
```
┌─────────────────────────────────┐
│ address.city                    │ ->  city
└─────────────────────────────────┘
┌─────────────────────────────────┐
│ properties["birth day"]         │ ->  birth_day
└─────────────────────────────────┘
```

### JSON Handling

When working with JSON objects, if a key does not exist, it evaluates to `null`. For example, consider the `traits` property with the following JSON value:

```json
{
    "address": {
        "city": "Milan"
    },
    "phone": "+39 02 12345678"
}
```

Here, `traits.address` evaluates to the JSON value `{"city": "Milan"}`, but `traits.name` evaluates to `null` because the key `"name"` does not exist in `traits`.

Accessing a non-object JSON value as if it were an object results in an error, causing the entire mapping to fail. For instance, the following mapping returns an error because `traits.phone` is not a JSON object:

```
┌─────────────────────────────────┐
│ traits.phone.mobile             │ ->  phoneNumber
└─────────────────────────────────┘
```

To avoid this error, you can add a `?` after the key, which prevents the error if the accessed JSON value is not an object. However, note that the output property `phoneNumber` will not have a value:

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

The arguments for `and` should be of boolean (`Boolean` type), and the returned value is also a boolean (`Boolean` type).

#### **array** function

The `array` function returns an array with the passed arguments as elements.  For example:
```
┌─────────────────────────────────┐
│ array(email, company.email)     │ ->  emails
└─────────────────────────────────┘
```

The `array` function returns a value in the form of a JSON array (`Array(JSON)` type).

#### **coalesce** function

The `coalesce` function returns the first non-null argument, or `null` if all arguments are `null`.  For example:

```
┌───────────────────────────────────────────┐
│ coalesce(shippingAddress, billingAddress) │ ->  ship_address
└───────────────────────────────────────────┘
```

The `coalesce` function returns a value in the form of a JSON value (`JSON` type).

#### **eq** function

The `eq` function takes two values and returns `true` if they are equal; otherwise, it returns `false`. For example:
```
┌─────────────────────────────────┐
│ eq(level, "VIP")                │ ->  vip_customer
└─────────────────────────────────┘
```

The `eq` function returns a boolean value (`Boolean` type).


#### **if** function

The `if` function evaluates the first boolean argument. If it is `true`, it returns the second argument; otherwise, it returns the third argument. For example, if `hasCode` is `true`, the following evaluates to the value of `code`; otherwise, it evaluates to `null`
```
┌─────────────────────────────────┐
│ if(hasCode, code, null)         │ ->  code
└─────────────────────────────────┘
```

The second or third argument is returned in the form of a JSON value (`JSON` type).

The `if` function can also be called with only two arguments. In this case, if the first argument is `false`, the output property will not have a value, as if it had not been mapped.

In the following example, the property will not have a value if `hasCode` is `false`
```
┌─────────────────────────────────┐
│ if(hasCode, code)               │ ->  code
└─────────────────────────────────┘
```


The second argument, if the first argument is `true`, is returned in the form of a JSON value (`JSON` type).

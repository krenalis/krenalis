{% extends "/layouts/doc.html" %}
{% macro Title string %}JavaScript{% end %}
{% Article %}

# JavaScript

This is how a JavaScript transform function looks like:

```javascript
const transform = (user) => {
    return {}
}
```

So, for example, it could be written like this:

```javascript
const transform = (user) => {
    return {
        email: user.email,
        first_name: user.first_name,
    }
}
```

## Types

The table below outlines the various Meergo types and their corresponding representations in the JavaScript code for the transformation.

| Meergo&nbsp;Type | JavaScript&nbsp;Type | Example                                  |
|------------------|----------------------|------------------------------------------|
| `boolean`        | `Boolean`            | `true`                                   |
| `int(n)` `n≤32`  | `Number`             | `-2586`                                  |
| `int(64)`        | `BigInt`             | `72750672843726543n`                     |
| `uint(n)` `n≤32` | `Number`             | `4063`                                   |
| `uint(64)`       | `BigInt`             | `160386761048264895n`                    |
| `float(n)`       | `Number`             | `37.81`                                  |
| `decimal(p,s)`   | `BigInt`,`String`    | `5930174.18n`                            |
| `datetime`       | `Date`,`String`      | `new Date("2024-01-15T08:51:49.822")`    |
| `date`           | `Date`               | `new Date("2012-05-18")`                 |
| `time`           | `Date`               | `new Date("0001-01-01T08:51:49.822")`    |
| `year`           | `Number`             | `2024`                                   |
| `uuid`           | `String`             | `'f956622d-c421-4eca-8d20-efef87f9749c'` |
| `json`           | `String`             | `'{"score":10}'`                         |
| `inet`           | `String`             | `'172.16.254.1'`                         |
| `text`           | `String`             | `'123 Main Street'`                      |
| `array`          | `Array`              | `[472,182,604]`                          |
| `object`         | `Object`             | `{fistName:'Emily',lastName:'Johnson'}`  |
| `map`            | `Object`             | `{'a':8073,'c':206}`                     |

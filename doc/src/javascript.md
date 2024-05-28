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

The table below outlines the various Chichi types and their corresponding representations in the JavaScript code for the transformation.

| Chichi&nbsp;Type | JavaScript&nbsp;Type | Example                                  |
|------------------|----------------------|------------------------------------------|
| `Boolean`        | `Boolean`            | `true`                                   |
| `Int(n)` `n≤32`  | `Number`             | `-2586`                                  |
| `Int(64)`        | `BigInt`             | `72750672843726543n`                     |
| `Uint(n)` `n≤32` | `Number`             | `4063`                                   |
| `Uint(64)`       | `BigInt`             | `160386761048264895n`                    |
| `Float(n)`       | `Number`             | `37.81`                                  |
| `Decimal(p,s)`   | `BigInt`,`String`    | `5930174.18n`                            |
| `DateTime`       | `Date`,`String`      | `new Date("2024-01-15T08:51:49.822")`    |
| `Date`           | `Date`               | `new Date("2012-05-18")`                 |
| `Time`           | `Date`               | `new Date("0001-01-01T08:51:49.822")`    |
| `Year`           | `Number`             | `2024`                                   |
| `UUID`           | `String`             | `'f956622d-c421-4eca-8d20-efef87f9749c'` |
| `JSON`           | `String`             | `'{"score":10}'`                         |
| `Inet`           | `String`             | `'172.16.254.1'`                         |
| `Text`           | `String`             | `'123 Main Street'`                      |
| `Array`          | `Array`              | `[472,182,604]`                          |
| `Object`         | `Object`             | `{fistName:'Emily',lastName:'Johnson'}`  |
| `Map`            | `Object`             | `{'a':8073,'c':206}`                     |

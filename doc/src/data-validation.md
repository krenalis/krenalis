# Data Validation

Data validation in Meergo occurs in various scenarios, ensuring accuracy and consistency throughout the data lifecycle. Whether collecting user data, transforming it, storing it in a data warehouse, or sending it to a destination location, Meergo validates the data against the relevant schema to maintain data integrity and quality.

Meergo validates data in the following scenarios:

| Data Validation Scenarios                                                                                                   |
|-----------------------------------------------------------------------------------------------------------------------------|
| **Collecting data from a source location**, validation occurs against the connection's source schema.                       |
| **Receiving events from a source location**, validation occurs against the schema of the events.                            |
| **Before data transformation**, validation checks are performed against the transformation's input schema.                  |
| **After data transformation**, the transformed data undergoes validation against the transformation's output schema.        |
| **During storage in the data warehouse**, data is validated against the destination table's schema.                         |
| **Sending data and events to a destination location**, validation is conducted against the connection's destination schema. |

## Data Types

The table below provides the types that can appears in a schema:

| Type           | Description                                                               |
|----------------|---------------------------------------------------------------------------|
| `Boolean`      | A boolean.                                                                |
| `Int(n)`       | A signed integer with `n` bytes. `n` can be 8, 16, 24, 32, or 64. [^1]    |
| `Uint(n)`      | An unsigned integer with `n` bytes. `n` can be 8, 16, 24, 32, or 64. [^1] |
| `Float(n)`     | A floating point number with `n` bytes. `n` can be 32, or 64. [^1] [^2]   |
| `Decimal(p,s)` | A decimal number with precision `p` and scale `s`. [^1] [^3]              |
| `DateTime`     | A date and time with the year in range [1, 9999]. [^4]                    |
| `Date`         | A date with the year in range [1, 9999].                                  |
| `Time`         | A time in the day. [^4]                                                   |
| `Year`         | A year in range [1, 9999].                                                |
| `UUID`         | A UUID.                                                                   |
| `JSON`         | A JSON data.                                                              |
| `Inet`         | An IP4 or IP6 address.                                                    |
| `Text`         | An UTF-8 encoded text. [^5]                                               |
| `Array(T)`     | An array with elements with type `T`. [^6] [^7]                           |
| `Object`       | An object with specified properties.                                      |
| `Map(T)`       | A map with keys of type `Text` and values of type `T`.                    |

[^1]: `Int(n)`, `Uint(n)`, `Float(n)`, and `Decimal(p,s)` can be limited in the range of the allowed values.

[^2]: `Float(n)` can be limited to finite values, excluding `NaN` and `±Infinity` from the allowed values.

[^3]: `Decimal(p,s)` has precision `p` in range [1, 76], scale `s` in range [0, 37], and `s` is less or equal to `p`.   

[^4]: `DateTime` and `Time` have nanosecond precision and no time zone.

[^5]: `Text` can be restricted by a list of allowed values, a regular expression, or maximum lengths in bytes and characters.

[^6]: An `Array` can have a minimum and maximum limit on the number of elements.

[^7]: An `Array` can be constrained to have unique values for its elements, except for arrays of `JSON`, `Array`, `Map`, and `Object`.

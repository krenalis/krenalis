{% extends "/layouts/doc.html" %}
{% macro Title string %}Data Validation{% end %}
{% Article %}

# Data validation

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

## Data types

The table below provides the types that can appears in a schema:

| Type           | Description                                                               |
|----------------|---------------------------------------------------------------------------|
| `text`         | An UTF-8 encoded text. [^1]                                               |
| `boolean`      | A boolean.                                                                |
| `int(n)`       | A signed integer with `n` bytes. `n` can be 8, 16, 24, 32, or 64. [^2]    |
| `uint(n)`      | An unsigned integer with `n` bytes. `n` can be 8, 16, 24, 32, or 64. [^3] |
| `float(n)`     | A floating point number with `n` bytes. `n` can be 32, or 64. [^1] [^3]   |
| `decimal(p,s)` | A decimal number with precision `p` and scale `s`. [^2] [^4]              |
| `datetime`     | A date and time with the year in range [1, 9999]. [^5]                    |
| `date`         | A date with the year in range [1, 9999].                                  |
| `time`         | A time in the day. [^5]                                                   |
| `year`         | A year in range [1, 9999].                                                |
| `uuid`         | A UUID.                                                                   |
| `json`         | A JSON data.                                                              |
| `inet`         | An IP4 or IP6 address.                                                    |
| `array(T)`     | An array with elements with type `T`. [^6] [^7]                           |
| `object`       | An object with specified properties.                                      |
| `map(T)`       | A map with keys of type `text` and values of type `T`.                    |

[^1]: `text` can be restricted by a list of allowed values, a regular expression, or maximum lengths in bytes and characters.

[^2]: `int(n)`, `uint(n)`, `float(n)`, and `decimal(p,s)` can be limited in the range of the allowed values.

[^3]: `float(n)` can be limited to finite values, excluding `NaN` and `±Infinity` from the allowed values.

[^4]: `decimal(p,s)` has precision `p` in range [1, 76], scale `s` in range [0, 37], and `s` is less or equal to `p`.

[^5]: `datetime` and `time` have nanosecond precision and no time zone.

[^6]: An `array` can have a minimum and maximum limit on the number of elements.

[^7]: An `array` can be constrained to have unique values for its elements, except for arrays of `json`, `array`, `map`, and `object`.

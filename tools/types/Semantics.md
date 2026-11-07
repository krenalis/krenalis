# Semantics

| Semantics     | Type restrictions                                                                     |
|---------------|---------------------------------------------------------------------------------------|
| `country`     | `string` without `maxBytes`, `maxLength`, `pattern` or `values`; `format` is required |
| `duration`    | `int`, `decimal` or `float` with `real`; `unit` is required                           |
| `email`       | `string`                                                                              |
| `measurement` | `int`, `decimal` or `float` with `real`; `unit` is required                           |
| `money`       | `decimal`; `currency` is optional                                                     |
| `percentage`  | `decimal`                                                                             |
| `phone`       | `string` without `maxBytes`, `maxLength`, `pattern` or `values`                       |
| `url`         | `string`                                                                              |
| `year`        | signed or unsigned `int`; values are restricted to `[1,9999]`                         |

For `year`, bounds implied by the semantics and bit size are omitted from
JSON. Explicit bounds must be within `[1,9999]` and fit the integer type.

Profile schema properties with `year` semantics must use signed `int(16)`
with the full `[1,9999]` range. This also applies to array and map elements.

Profile properties with `year` semantics cannot be used as identifiers.

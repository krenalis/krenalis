# Semantics

| Semantic      | Type restrictions                                                                     |
|---------------|---------------------------------------------------------------------------------------|
| `country`     | `string` without `maxBytes`, `maxLength`, `pattern` or `values`; `format` is required |
| `duration`    | `int`, `decimal` or `float` with `real`; `unit` is required                           |
| `email`       | `string`                                                                              |
| `measurement` | `int`, `decimal` or `float` with `real`; `unit` is required                           |
| `money`       | `decimal`; `currency` is optional                                                     |
| `percentage`  | `decimal`                                                                             |
| `phone`       | `string` without `maxBytes`, `maxLength`, `pattern` or `values`                       |
| `url`         | `string`                                                                              |

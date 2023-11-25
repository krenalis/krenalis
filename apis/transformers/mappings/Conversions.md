

| from/to  | Boolean | Int  | Uint | Float | Decimal | DateTime | Date | Time | Year | UUID | JSON | Inet | Text | Array(T) | Object | Map(T) |
|----------|---------|------|------|-------|---------|----------|------|------|------|------|------|------|------|----------|--------|--------|
| Boolean  | ✓       | [^1] | [^1] |       |         |          |      |      |      |      | ✓    |      | ✓    |          |        |        |
| Int      | [^1]    | ✓    | ✓    | ✓     | ✓       |          |      |      | ✓    |      | ✓    |      | ✓    |          |        |        |
| Uint     | [^1]    | ✓    | ✓    | ✓     | ✓       |          |      |      | ✓    |      | ✓    |      | ✓    |          |        |        |
| Float    |         | ✓    | ✓    | ✓     | ✓       |          |      |      |      |      | ✓    |      | ✓    |          |        |        |
| Decimal  |         | ✓    | ✓    | ✓     | ✓       |          |      |      |      |      | ✓    |      | ✓    |          |        |        |
| DateTime |         |      |      |       |         | ✓        | ✓    | ✓    |      |      | ✓    |      | ✓    |          |        |        |
| Date     |         |      |      |       |         | ✓        | ✓    |      |      |      | ✓    |      | ✓    |          |        |        |
| Time     |         |      |      |       |         |          |      | ✓    |      |      | ✓    |      | ✓    |          |        |        |
| Year     |         | ✓    | ✓    |       |         |          |      |      | ✓    |      | ✓    |      | ✓    |          |        |        |
| UUID     |         |      |      |       |         |          |      |      |      | ✓    | ✓    |      | ✓    |          |        |        |
| JSON     | ✓       | ✓    | ✓    | ✓     | ✓       | ✓        | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓        | ✓      | ✓      |
| Inet     |         |      |      |       |         |          |      |      |      |      | ✓    | ✓    | ✓    |          |        |        |
| Text     | ✓       | ✓    | ✓    | ✓     | ✓       | ✓        | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    |          |        |        |
| Array(T) |         |      |      |       |         |          |      |      |      |      | ✓    |      |      | ✓        |        |        |
| Object   |         |      |      |       |         |          |      |      |      |      | ✓    |      |      |          | ✓      |        |
| Map(T)   |         |      |      |       |         |          |      |      |      |      | ✓    |      |      |          |        | ✓      |

[^1]: Only for Int(8) and Uint(8).

Note: keep this table in sync with the matrix within the package `mappings`,
which holds information about valid conversions.

# From nil

If the destination property is nullable, `nil` is converted to `nil`; otherwise:

* if the destination type is **Text**, `nil` is converted to an empty string
* if the destination type is **JSON**, `nil` is converted to **JSON** `null`
* in all other cases, it cannot be converted.

## To nil

A value `v` is converted to `nil` if the destination property is nullable and one of the following conditions is true:

* `v` is `nil`
* `v` is **JSON** `null`, and the destination type is not **JSON**
* `v` is an empty **Text**, and the destination type is not **Text**
* `v` is an empty **Text**, and the destination type is **Text** with enums
* `v` is an empty **Text**, and the destination type is **Text** with a regular expression, and `v` does not match


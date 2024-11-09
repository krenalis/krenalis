

| from/to  | Boolean | Int  | Uint | Float | Decimal | DateTime | Date | Time | Year | UUID | JSON | Inet | Text | Array(T) | Object | Map(T) |
|----------|---------|------|------|-------|---------|----------|------|------|------|------|------|------|------|----------|--------|--------|
| Boolean  | ✓       | [^1] | [^1] |       |         |          |      |      |      |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Int      | [^1]    | ✓    | ✓    | ✓     | ✓       |          |      |      | ✓    |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Uint     | [^1]    | ✓    | ✓    | ✓     | ✓       |          |      |      | ✓    |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Float    |         | ✓    | ✓    | ✓     | ✓       |          |      |      |      |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Decimal  |         | ✓    | ✓    | ✓     | ✓       |          |      |      |      |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| DateTime |         |      |      |       |         | ✓        | ✓    | ✓    |      |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Date     |         |      |      |       |         | ✓        | ✓    |      |      |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Time     |         |      |      |       |         |          |      | ✓    |      |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| Year     |         | ✓    | ✓    |       |         |          |      |      | ✓    |      | ✓    |      | ✓    | ✓ [^2]   |        |        |
| UUID     |         |      |      |       |         |          |      |      |      | ✓    | ✓    |      | ✓    | ✓ [^2]   |        |        |
| JSON     | ✓       | ✓    | ✓    | ✓     | ✓       | ✓        | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓        | ✓      | ✓      |
| Inet     |         |      |      |       |         |          |      |      |      |      | ✓    | ✓    | ✓    | ✓ [^2]   |        |        |
| Text     | ✓       | ✓    | ✓    | ✓     | ✓       | ✓        | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓    | ✓ [^2]   |        |        |
| Array(T) |         |      |      |       |         |          |      |      |      |      | ✓    |      |      | ✓        |        |        |
| Object   |         |      |      |       |         |          |      |      |      |      | ✓    |      |      |          | ✓      | ✓      |
| Map(T)   |         |      |      |       |         |          |      |      |      |      | ✓    |      |      |          | ✓      | ✓      |

[^1]: Only for Int(8) and Uint(8).
[^2]: Only if it can be converted to the type of the array item. 

Note: keep this table in sync with the `convertMatrix` global variable in this package,
which holds information about valid conversions.

## Handling of `nil`

### From `nil`

If the destination type is **JSON**, `nil` is converted to **JSON** `null`; otherwise, if the destination property is nullable, `nil` is converted to `nil`; otherwise, it is converted to `void`.

### To `nil`

A value `v` is converted to `nil` if the destination property is nullable and one of the following conditions is true:

* `v` is `nil`
* `v` is **JSON** `null`, and the destination type is not **JSON**
* `v` is an empty **Text**, and the destination type is not **Text**
* `v` is an empty **Text**, and the destination type is **Text** with enums
* `v` is an empty **Text**, and the destination type is **Text** with a regular expression, and `v` does not match


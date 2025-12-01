

| from/to  | string | boolean | int  | float | decimal | datetime | date | time | year | uuid | json | ip | array(T) | object | map(T) |
|----------|--------|---------|------|-------|---------|----------|------|------|------|------|------|----|----------|--------|--------|
| string   | ✓      | ✓       | ✓    | ✓     | ✓       | ✓        | ✓    | ✓    | ✓    | ✓    | ✓    | ✓  |          |        |        |
| boolean  | ✓      | ✓       | [^1] |       |         |          |      |      |      |      | ✓    |    |          |        |        |
| int      | ✓      | [^1]    | ✓    | ✓     | ✓       |          |      |      | ✓    |      | ✓    |    |          |        |        |
| float    | ✓      |         | ✓    | ✓     | ✓       |          |      |      |      |      | ✓    |    |          |        |        |
| decimal  | ✓      |         | ✓    | ✓     | ✓       |          |      |      |      |      | ✓    |    |          |        |        |
| datetime | ✓      |         |      |       |         | ✓        | ✓    | ✓    |      |      | ✓    |    |          |        |        |
| date     | ✓      |         |      |       |         | ✓        | ✓    |      |      |      | ✓    |    |          |        |        |
| time     | ✓      |         |      |       |         |          |      | ✓    |      |      | ✓    |    |          |        |        |
| year     | ✓      |         | ✓    |       |         |          |      |      | ✓    |      | ✓    |    |          |        |        |
| uuid     | ✓      |         |      |       |         |          |      |      |      | ✓    | ✓    |    |          |        |        |
| json     | ✓      | ✓       | ✓    | ✓     | ✓       | ✓        | ✓    | ✓    | ✓    | ✓    | ✓    | ✓  | ✓        | ✓      | ✓      |
| ip       | ✓      |         |      |       |         |          |      |      |      |      | ✓    | ✓  |          |        |        |
| array(T) |        |         |      |       |         |          |      |      |      |      | ✓    |    | ✓        |        |        |
| object   |        |         |      |       |         |          |      |      |      |      | ✓    |    |          | ✓      | ✓      |
| map(T)   |        |         |      |       |         |          |      |      |      |      | ✓    |    |          | ✓      | ✓      |

[^1]: Only for `int(8)`.

Note: keep this table in sync with the `convertMatrix` global variable in this package,
which holds information about valid conversions.

## Handling of `nil`

### From `nil`

If the destination type is **json**, `nil` is converted to **json** `null`; otherwise, if the destination property is nullable, `nil` is converted to `nil`; otherwise, it is converted to `void`.

### To `nil`

A value `v` is converted to `nil` if the destination property is nullable and one of the following conditions is true:

* `v` is `nil`
* `v` is **json** `null`, and the destination type is not **json**
* `v` is an empty **string**, and `v` is not constant, and the destination type is neither **string** nor **json** 
* `v` is an empty **string**, and the destination type is **string** with enums
* `v` is an empty **string**, and the destination type is **string** with a regular expression, and `v` does not match

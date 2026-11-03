

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

If the destination is nullable, `nil` remains `nil`. If it is non-nullable and has type **json**, `nil` becomes **json** `null`. For other non-nullable destinations, a directly mapped property is omitted, but `nil` in an object property, array element, or map value causes an error.

**json** `null` is preserved for **json** destinations. Array elements and map values are not nullable.

### To `nil`

A value `v` is converted to `nil` if the destination property is nullable and one of the following conditions is true:

* `v` is `nil`
* `v` is **json** `null`, and the destination type is not **json**
* `v` is an empty **string**, and `v` is not constant, and the destination type is neither **string** nor **json** 
* `v` is an empty **string**, and the destination type is **string** with enums
* `v` is an empty **string**, and the destination type is **string** with a regular expression, and `v` does not match

## Required properties

For `Create` and `Update`, required properties are checked in the complete output, including in objects within arrays and maps. An object's children are checked only when the object is present and non-null.

The output matching property, when specified, is supplied separately by the exporter and is excluded from these checks. For `Create`, its missing ancestors are checked as empty objects because inserting the matching value will create them. The same applies to `Update` only when the matching property is `UpdateRequired`. Other required children of those ancestors must still be present. These checks do not add the matching property or its ancestors to the mapping output.

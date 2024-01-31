# Allowed Types For Identifiers

Here is a list of allowed types for the [identifiers](workspace-identity-resolution.md#identifiers) used in the [Workspace Identity Resolution](workspace-identity-resolution.md). 

> See the issue [#321](https://github.com/open2b/chichi/issues/321)


| Type           | Allowed                                                                |
|----------------|------------------------------------------------------------------------|
| `Boolean`      | No                                                                     |
| `Int(n)`       | **Yes**                                                                |
| `UInt(n)`      | **Yes**                                                                |
| `Float`        | No                                                                     |
| `Float(n)`     | No                                                                     |
| `Decimal(p,s)` | **Yes**, but only if scale `s` is 0                                    |
| `DateTime`     | No                                                                     |
| `Date`         | No                                                                     |
| `Time`         | No                                                                     |
| `Year`         | No                                                                     |
| `UUID`         | **Yes**                                                                |
| `JSON`         | No                                                                     |
| `Inet`         | **Yes**                                                                |
| `Text`         | **Yes**                                                                |
| `Array(T)`     | **Yes**, but only if the item type `T` is allowed and it's not `Array` |
| `Object`       | No                                                                     |
| `Map(T)`       | No                                                                     |

# Business ID

The Business ID helps you visually recognize the identities imported from the connections.

It does not impact in any way the behavior of the software or the [Identity Resolution](./identity-resolution.md); it is purely a visual piece of information used to identify more clearly the identities associated with a user.

For source connections, a **Business ID name** can be specified.

## Business ID name

The **Business ID name** is a property name or a column name from which the value of the Business ID is read when importing.

* For **app connections**, it must refer to a property of the app with a supported type.
* For **database connections**, it must be a column returned by the database query with a supported type.
* For **events connections** (mobile, website and server), it must be the name of a field of the event's `traits` with a supported type.
* For **file storage connections**, it must be a column of the file with a supported type.

If such property / column does not exist on the source connection, or if the type is not supported, the Business ID is not imported.

### Supported types

The Business ID can be imported only from properties (or column) with one of those [types](data-validation.html#data-types):

* `Int(n)`
* `UInt(n)`
* `Float(n)`
* `Decimal(p, s)`, but only if scale `s` is 0
* `JSON`, but only if the JSON value is Number or String
* `Text`

## Changing the Business ID

Any changes to the Business ID name **becomes effective** on to the identities of a connection **when the corresponding import action is executed**.

Until that moment, the identities will continue to show the settings and the value of the Business ID present at the time when the import of such identities occurred.
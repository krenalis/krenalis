# Importing

## User identifier

Actions on **file storage** and **database** source connections **must** indicate an user identifier which identifies an user. It must be a column with one of the following [types](data-validation.html#data-types):

* `Int(n)`
* `Uint(n)`
* `UUID`
* `JSON`
* `Text`

> It is advisable that the user identifier be unique for each user found on the source connection (be it a file or a set of values returned by a database query), as this value is then used for the [Connection Identity Resolution](./identity-resolution/connection-identity-resolution.md).

## Last change time column

Actions on **file storage** and **database** source connections **may** indicate a column containing a timestamp indicating the last change time of the user. It must be a column with one of the following [types](data-validation.html#data-types):

* `DateTime`
* `Date`
* `JSON`
* `Text`

If a last change time column is provided and its type is JSON or Text, a timestamp format must be provided for parsing its value.

## Importing anonymous user identities from events

In order for a user identity to be imported from an anonymous event, it is necessary that the mapping applied to the event results in at least one property.

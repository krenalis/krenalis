# Connection Identity Resolution

The Connection Identity Resolution determines if an user, imported from a connection, was previously imported by that connection, and eventually merges them.

## Behaviour

When a user incomes from a connection, it is appended to the `users_identities` table of the data warehouse. Then:

for users **imported from apps**, the user is merged with other users within the same connection who share the same app identifier provided by the app (thus this does not require any manual configuration by the user of Chichi);

for users **imported from files** and **databases**, the user is merged with other users within the same connection who share the same value for [user identifier](../importing.md#user-identifier) specified in the action's editing page;

for users **imported from events**, if the user…

* … **is anonymous** (i.e. does not have a value for `userId`), it is merged with **other anonymous users** imported from the same connection who have an Anonymous ID in common.
* … **is not anonymous**, it is merged with other users within the same connection that (1) share the same `userId` or (2) do not have an `userId` and have an Anonymous ID in common.

The behavior for the users imported from events allows the implementation of [strategies](anonymous-users-strategies.md) by controlling how `userId` and `anonymousId` are sent by the client (eg. the [JavaScript SDK](../javascript-sdk.md) in the browser).

## Merging of users

When merging two or more users during the Connection Identity Resolution into a single user:

* the Anonymous IDs are taken from all these users, without duplicated values
* for any other property, the value of the resulting user for that property is taken from the most recently updated user who *has a value* for that property

> NOTE: the meaning of *has a value* is unclear, so the content of this section about which values are merged may be wrong. This must be reviewed. See the issue [#657](https://github.com/open2b/chichi/issues/657).
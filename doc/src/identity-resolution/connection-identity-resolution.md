# Connection Identity Resolution

The Connection Identity Resolution determines if an user, imported from a connection, was previously imported by that connection, and eventually merges them.

## Behaviour

When a user incomes from a connection, it is appended to the `users_identities` table of the data warehouse. Then:

for users **imported from apps**, the user is merged with other users within the same connection who share the same app identifier provided by the app (thus this does not require any manual configuration by the user of Chichi);

for users **imported from databases**, the user is merged with other users within the same connection who share the same value for the `id` column returned by the query;

for users **imported from files**, the user is merged with other users within the same connection who share the same value for the identity column specified in the action's editing page;

for users **imported from events traits**, if the user…

* … **is anonymous** (i.e. does not have a value for `userId`), it is merged with other users imported from the same connection who have an anonymous ID in common.
* … **is not anonymous**, it is merged with other users within the same connection that share the same `userId` or do not have an `userId` and have an anonymous ID in common.

The behavior for the users imported from events traits allows the implementation of [strategies](anonymous-users-strategies.md) by controlling how `userId` and `anonymousId` are sent by the client (eg. the JavaScript SDK in the browser).

## Merging of users

When merging two or more users during the Connection Identity Resolution into a single user:

* the anonymous IDs are taken from all these users, without duplicated values
* for any other property, the value of the resulting user for that property is taken from the most recently updated user who has a value for that property
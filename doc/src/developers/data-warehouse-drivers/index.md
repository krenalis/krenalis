# Data Warehouse Drivers

## Meta properties

### The concept of "meta property"

A meta property is a property that provides additional information about a record. There are meta properties whose presence is necessary for the functioning of Meergo; the list is indicated in the following sections.

### Meta property syntax

Meta properties are, syntactically, properties whose names start and end with `__`, and which are at least 5 characters long. For example, these property names are considered meta properties:

```
__id__
__anonymous_ids__
__GID__
__x__
```

## Driver requirements

> Note: this section needs to be expanded and therefore may be incomplete.

A driver must ensure that the data warehouse contains at least these tables with their respective columns:

| Table Name         | Column Name                             | Type          | Additional Requirements        | Description                                                                                                                              |
|--------------------|-----------------------------------------|---------------|--------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| `users`            | `__id__`                                | `UUID`        | -                              | GID of the user                                                                                                                          |
| `users`            | `__last_change_time__`                  | `DateTime`    | not null                       | Datetime of last modification of the user                                                                                                |
| `_user_identities` | `__pk__`                                | `Int(32)`     | auto-incrementing, primary key | It is used to refer a specific identity. It can probably then be removed, see issue [#687](https://github.com/meergo/meergo/issues/687). |
| `_user_identities` | `__action__`                            | `Int(32)`     | not null                       | Action from which the identity has been imported                                                                                         |
| `_user_identities` | `__is_anonymous__`                      | `Boolean`     | not null, default false        | True for anonymous identities, false in any other case.                                                                                  |
| `_user_identities` | `__identity_id__`                       | `Text`        | not null                       | Identifier for the identity. For anonymous identities, this is the anonymous ID.                                                         |
| `_user_identities` | `__connection__`                        | `Int(32)`     | not null                       | Connection from which the identity has been imported                                                                                     |
| `_user_identities` | `__anonymous_ids__`                     | `Array(Text)` | -                              | List of anonymous IDs for logged in users. Empty for anonymous users, for whom the anonymous ID is inside `__identity_id__`              |
| `_user_identities` | `__last_change_time__`                  | `DateTime`    | not null                       | Datetime of last modification of the identity                                                                                            |
| `_user_identities` | `__execution__`                         | `Int(32)`     | -                              | Identifier of the last execution that updated the identity                                                                               |
| `_user_identities` | `__gid__`                               | `UUID`        | -                              | GID of the user associated with the identity                                                                                             |
| `events`           | *every column in the event schema* [^1] |               | -                              | -                                                                                                                                        |

Note that other tables, for example those to store destination users, are at the discretion of the driver, which must only expose the methods to implement the interface in Go.

[^1]: currently, the event schema is omitted here for brevity.
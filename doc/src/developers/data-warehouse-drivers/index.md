# Data Warehouse Drivers

## Meta properties

### The concept of "meta property"

A meta property is a property that provides additional information about a record. There are meta properties whose presence is necessary for the functioning of Chichi; the list is indicated in the following sections.

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

| Table Name         | Column Name                             | Type          | Additional Requirements                                       |
|--------------------|-----------------------------------------|---------------|---------------------------------------------------------------|
| `users`            | `__id__`                                | uuid          | -                                                             |
| `users_identities` | `__pk__`                                | integer       | auto-incrementing, primary key                                |
| `users_identities` | `__connection__`                        | integer       | not null, default to 0                                        |
| `users_identities` | `__identity_id__`                       | text          | not null, default to empty string                             |
| `users_identities` | `__displayed_property__`                | text          | 40 characters long or more, not null, default to empty string |
| `users_identities` | `__anonymous_ids__`                     | array of text | -                                                             |
| `users_identities` | `__last_change_time__`                  | timestamp     | not null                                                      |
| `users_identities` | `__gid__`                               | uuid          | -                                                             |
| `events`           | *every column in the event schema* [^1] |               |                                                               |

Note that other tables, for example those to store destination users, are at the discretion of the driver, which must only expose the methods to implement the interface in Go.

[^1]: currently, the event schema is omitted here for brevity.
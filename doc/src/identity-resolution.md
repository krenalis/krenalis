{% extends "/layouts/doc.html" %}
{% macro Title string %}The Identity Resolution{% end %}
{% Article %}

# The Identity Resolution

The **Identity Resolution** determines if more user identities, belonging to **any connection** of the workspace (even the same connection), are a single user of the workspace, and eventually merges them. It also associates the workspace users to the events stored in the data warehouse.

In particular, it performs these operations (not necessarily in this order):

* recreates the contents of the `users` table starting from the identities within `_user_identities`
* updates the association between the events within the `events` table and the users within `users`

## When it is executed

The Identity Resolution can be executed manually by the user through a button in the admin.

It is also possible to configure Meergo to automatically perform Identity Resolution at the end of every user import from apps, files, or databases. This can be configured through the Identity Resolution settings in the admin.

## Same user criterion

Given two user identities, they correspond to the *same user* **if at least one** of those cases applies:

* they have been imported by the same connection, while they have the same identity ID and are both anonymous or both non-anonymous;[^differentactions]
* they have at least one [identifier](#identifiers) whose value matches (that is, has the same value[^samevalue]), and one or both identities have a `NULL` value for each identifier with higher priority than the first identifier that has matched.

[^differentactions]: this handles the case when a user identity have been imported by two different actions of the same connection.
[^samevalue]: same value means, currently (in the PostgreSQL driver): cast to `text` and then compare with `=`.

### An example

In this example, with two user identities (A and B) and three identifiers (where #1 has the higher priority):

|            | Identifier #1 | Identifier #2 | Identifier #3 |
|------------|---------------|---------------|---------------|
| Identity A | `NULL`        | 10            | 45            |
| Identity B | `NULL`        | `NULL`        | 45            |

the two identities A and B match because they have the same value for the identifier #3, and one or both have a `NULL` value with identifiers with higher priority (#1 and #2). 

As a corollary of the previous definition, **if there are no identifiers defined in the workspace**, the Identity Resolution considers **every user identity imported** from a connection **always different from any other identity**.

## Identifiers

An identifier consists in **a property path** which refers to a property of the user schema which have [an allowed type](./identity-resolution/allowed-types-for-identifiers).

It is possible to define zero, one, or more identifiers for the identity resolution. 

In case more than one identifier is defined, it is necessary to choose a **priority order**, which will be taken into account by the identity resolution procedure.

So, for example:

```
[ 1 ]  customerId
[ 2 ]  taxCode
[ 3 ]  email
[ 4 ]  address.street1
```

Here, `customerId` is the identifier with the higher priority while `address.street1` has the lower priority.

## Merging of users

In Identity Resolution, **two or more user identities** are merged into a single user by merging, one by one, all their properties, including identifiers.

The values of a property P are merged in this way:

* **if P is an array**: the values of the arrays of the identities are concatenated into a single array, without duplicated values. The ordering of the values in the array is not established a priori and is left to the driver.
* **if P is not an array**: if there is a source connection S that is **set as primary** for P and at least one identity imported from S has a non-null value for P, then the non-null value from the most recently updated identity imported from S is taken. Otherwise, the value is taken from the most recently updated identity which has a non-null value for P. If there is none, then null is assigned to P.

For example, consider these three user identities, with the properties `email`, `name` and `total_orders`, which are considered *the same user* by the Identity Resolution and thus must be merged:

| Connection | email | name   | phone_numbers      | total_orders | Last change time         |
|------------|-------|--------|--------------------|--------------|--------------------------|
| A          | a@b   | John   | {+11 111}          | 10           | Jan 1, 2000, 12:00:00 PM |
| B          | a@b   | `NULL` | {+22 222, +33 333} | 20           | Jan 2, 2000, 12:00:00 PM |
| C          | a@b   | `NULL` | `NULL`             | 21           | Jan 3, 2000, 12:00:00 PM |

The resulting user will be then:

| email | name | phone_numbers               | total_orders | Last change time         |
|-------|------|-----------------------------|--------------|--------------------------|
| a@b   | John | {+11 111, +22 222, +33 333} | 21           | Jan 3, 2000, 12:00:00 PM |

Now let's suppose that the connection B is set as primary source for the property `total_orders`.

The resulting user will be then:

| email | name | phone_numbers               | total_orders           | Last change time         |
|-------|------|-----------------------------|------------------------|--------------------------|
| a@b   | John | {+11 111, +22 222, +33 333} | **20** (instead of 21) | Jan 3, 2000, 12:00:00 PM |

where the value "20" for `total_orders` comes from B, which is primary for this property, instead of coming from the identity from C (even if this last one was updated more recently).

## User GIDs

A GID is a UUID that uniquely identifies a user at a certain point in time.

During the **Identity Resolution**, a user's GID **is retained unless**:

- the users **has been merged** with other users
- the users **has been split** into two or more users

In these cases, the GID of the original user is deleted, and one or more new GIDs are created in its place.

## Association between events and users

From the same connection which receives events, can both be imported users and events, using different actions. The Identity Resolution, as mentioned before, also associated events to the users of the workspace.

Every event is associated to the **users incoming from the same connection** which:

* have the same `userId` (in case of non-anonymous events)
* or have an `anonymousId` in common (in case of anonymous events).

At that point, since users from multiple connections have been merged together through the Identity Resolution, **events from different connections can be associated to the same workspace user**.

<img src="../images/events-users.png" width="70%" alt="event-users">
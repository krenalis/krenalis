
# The Identity Resolution

The **Identity Resolution** determines if more user identities, belonging to **any connection** of the workspace (even the same connection), are a single user of the workspace, and eventually merges them. It also associates the workspace users to the events stored in the data warehouse.

In particular, it performs these operations (not necessarily in this order):

* recreates the contents of the `users` table starting from the identities within `user_identities`
* updates the association between the events within the `events` table and the users within `users`
* deletes the users within `user_identities` and `users` which no longer belong to any connection (i.e. connections that have been deleted).

## When It Is Executed

The Identity Resolution is executed:

* arbitrarily by Chichi, for example when importing from a connection
* explicitly by the user

>  We have a discussion on that, [#354](https://github.com/open2b/chichi/issues/354).

## Same User Criterion

This is the definition: **given two user identities, they correspond to the *same user* if (1) they have at least one *identifier* whose value matches (that is, has the same value[^samevalue]), and (2) one or both identities have `NULL` value for each identifier with higher priority than the first identifier that has matched.**

[^samevalue]: same value means, currently (in the PostgreSQL driver): cast to `text` and then compare with `=`.

In this example, with two user identities (A and B) and three identifiers (where #1 has the higher priority):

|            | Identifier #1 | Identifier #2 | Identifier #3 |
|------------|---------------|---------------|---------------|
| Identity A | `NULL`        | 10            | 45            |
| Identity B | `NULL`        | `NULL`        | 45            |

the two identities A and B match because they have the same value for the identifier #3, and one or both have a `NULL` value with identifiers with higher priority (#1 and #2). 

As a corollary of the previous definition, **if there are no identifiers defined in the workspace**, the Identity Resolution considers **every user identity imported** from a connection **always different from any other identity**.

## Identifiers

An identifier consists in **a property path** which refers to a property of the `user_identities` schema which have [an allowed type](./identity-resolution/allowed-types-for-identifiers.md).

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

## Merging of Users

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

| email | name | phone_numbers               | total_orders |
|-------|------|-----------------------------|--------------|
| a@b   | John | {+11 111, +22 222, +33 333} | 21           |

Now let's suppose that the connection C is set as primary source for the property `total_orders`.

The resulting user will be then:

| email | name | phone_numbers               | total_orders           |
|-------|------|-----------------------------|------------------------|
| a@b   | John | {+11 111, +22 222, +33 333} | **20** (instead of 21) |

## User GIDs

A GID is a UUID that uniquely identifies a user at a certain point in time.

During the **Identity Resolution**, a user's GID **is retained unless**:

- the users **has been merged** with other users
- the users **has been split** into two or more users

In these cases, the GID of the original user is deleted, and one or more new GIDs are created in its place.

## Association Between Events and Users

From the same connection which receives events, can both be imported users and events, using different actions. The Identity Resolution, as mentioned before, also associated events to the users of the workspace.

Every event is associated to the **users incoming from the same connection** which:

* have the same `userId` (in case of non-anonymous events)
* or have an `anonymousId` in common (in case of anonymous events).

At that point, since users from multiple connections have been merged together through the Identity Resolution, **events from different connections can be associated to the same workspace user**.

<img src="../images/events-users.png" width="70%" alt="event-users">
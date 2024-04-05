
# Workspace Identity Resolution

The Workspace Identity Resolution determines if more users, belonging to **any connection** of the workspace (even the same connection), are a single user of the workspace, and eventually merges them. It also associates the workspace users to the events stored in the data warehouse.

## When is executed

The Workspace Identity Resolution is executed:

* arbitrarily by Chichi, for example when importing from a connection
* explicitly by the user

>  We have a discussion on that, [#354](https://github.com/open2b/chichi/issues/354).

## Same user criterion

> NOTE: this paragraph should be reviewed and eventually clarified. See the issue [#657](https://github.com/open2b/chichi/issues/657).

Given two users, they are the *same user* if they have at least one equal value for an **identifier**, and if at least one of the users have no value for identifiers with higher priority.

Hence, it follows that if there are no identifiers defined in the workspace, the Workspace Identity Resolution considers every user imported from a connection always different from any other user.

## Identifiers

An identifier consists in **a property path** which refers to a property of the `users_identities` schema which have [an allowed type](./allowed-types-for-identifiers.md).

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

In the Workspace Identity Resolution, users are merged by taking the `max` value between the values of the properties for the users.

> `max` refers to the `max` function in PostgreSQL, which [is documented here](https://www.postgresql.org/docs/current/tutorial-agg.html).

For example, consider two users with the properties `email`, `name` and `totalOrders`, which are considered *the same user* by the Workspace Identity Resolution and thus must be merged:

| email | name     | totalOrders |
|-------|----------|-------------|
| a@b   | John     | 10          |
| a@b   | *(null)* | 20          |

The resulting user of the workspace will be:

| email | name | totalOrders |
|-------|------|-------------|
| a@b   | John | 20          |

## Association between events and users

From the same connection which receives events, can both be imported users and events, using different actions. The Workspace Identity Resolution, as mentioned before, also associated events to the users of the workspace.

Every event is associated to the **users incoming from the same connection** which:

* have the same `userId` (in case of non-anonymous events)
* or have an `anonymousId` in common (in case of anonymous events).

At that point, since users from multiple connections have been merged together through the Workspace Identity Resolution, **events from different connections can be associated to the same workspace user**.

<img src="../images/events-users.png" width="70%" alt="event-users">

## Deletion of orphan users

When executing the Workspace Identity Resolution, the users within `users_identities` and `users` which no longer belong to any connection (i.e. connections that have been deleted) are deleted.
# Identity Resolution

The Identity Resolution performs two different tasks:

1. Determines if an user, imported from a connection, was already imported **by that connection**, and eventually merges them.
2. Determine if two users, belonging to **two different connections**, should be merged into a single user.

Note that while **the first task overwrites information in Chichi**, updating users imported from a connection with the updated data, **the second task is reversible and can be re-executed from scratch at any time**.

> The second task of the identity resolution procedure is executed at various moments, but it is not clear which ones. Refer to the issue [#354](https://github.com/open2b/chichi/issues/354).

## Identifiers

An identifier is a property from the properties of the the `users_identities` schema which have a *compatible type* (see the issue [#321](https://github.com/open2b/chichi/issues/321)).

It is possible to define zero, one, or more identifiers for the identity resolution. 

There are two types of identifiers: **non-anonymous** and **anonymous**. The difference is not related to the identity resolution procedure – which treats both in the same way – but to how corresponding values for the users are imported.

> The properties shown in the UI are currently wrong. See the issue [#320](https://github.com/open2b/chichi/issues/320). 

> Should be property names or property paths? See the issue [#514](https://github.com/open2b/chichi/issues/514).

### Non-anonymous Identifiers

Non-anonymous identifiers are chosen from the properties of the `users_identities` schema.

It is necessary to choose a **priority order**, which will be taken into account by the identity resolution procedure.

```
[ 1 ]  customerId
[ 2 ]  taxCode
[ 3 ]  email
```

### Anonymous Identifiers

The anonymous identifiers, like non-anonymous ones, are chosen from `users_identities` schema properties and **have lower priority** than the former.

Here, in addition to choosing these identifiers, it is necessary to **specify a mapping between incoming event properties and the chosen anonymous identifiers**. This mapping will be executed when importing traits of an incoming event. 

```
┌────────────────────────┐
│ context.device.id      │ ->  [ 1 ]  ios.id
└────────────────────────┘
┌────────────────────────┐
│ context.ip             │ ->  [ 2 ]  ip
└────────────────────────┘
```

So, why using anonymous identifiers? They are useful to avoid repeating the same transformation on anonymous properties (device IDs, etc...) in every action that import user traits from events, so their behavior can be replaced with non-anonymous identifiers and mappings within actions.


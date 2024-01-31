# Identity Resolution

The Identity Resolution consists in two different procedures:

1. The [Connection Identity Resolution](identity-resolution/connection-identity-resolution.md), which determines if an user, imported from a [connection](./connections.md), was previously imported **by that connection**, and eventually merges them.
2. The [Workspace Identity Resolution](identity-resolution/workspace-identity-resolution.md), which determines if two users, belonging to **any connection** of the workspace (even the same connection), should be merged into a single user of the workspace. It also associates the workspace users to the events stored in the data warehouse.

The former is executed when a user is imported by a connection, while the latter is executed arbitrarily by Chichi when necessary.

> We have a discussion on that, [#354](https://github.com/open2b/chichi/issues/354).

**A note about data overwriting**. While the Connection Identity Resolution overwrites information in Chichi, by updating users imported from a connection with the updated data, the Workspace Identity Resolution is reversible and can be re-executed from scratch at any time.

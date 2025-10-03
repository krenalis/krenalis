{% extends "/layouts/doc.html" %}
{% macro Title string %}Importing{% end %}
{% Article %}

# Importing

> ⚠️⚠️⚠️ NOTE.
> This part of Meergo is under active development, so the content of this page may be outdated in any moment.

When importing users from the action of a connection, be it an app, file, database or a connection that receives events, these users are imported into Meergo in the form of "user identities" and are associated with the importing action.

The **Identity Resolution** procedure will then evaluate whether or not these user identities correspond to the same user, thus establishing and updating the actual users within Meergo.

## Behavior

When a user identity is imported from a connection's action, the identities are updated like this:

* If it is **imported for the first time**, a new identity is created
* If it **has already been imported** previously, the properties of the already imported identity are overwritten with those of the new one (including overwriting values which are null)

## How identities are identified

Identifying a user identity and understanding how this identification occurs is essential to ensure that the import occurs correctly.

* when importing from **app**, the identifier provided by the app is used (thus this does not require any manual configuration by the user of Meergo);
* when importing from a **file** or **database**, the [user identifier](#user-identifier) specified in the action's editing page is compared;
* when importing from an **event**, the `userId` (for non-anonymous user identities) or the `anonymousId` (for anonymous user identities) is compared

> The behavior for the users imported from events allows the implementation of [strategies](identity-resolution/anonymous-users-strategies) by controlling how `userId` and `anonymousId` are sent by the client (eg. the [JavaScript SDK](connectors/sources/javascript-sdk) in the browser).

### User identifier

Actions on **file storage** and **database** source connections **must** indicate an user identifier which identifies an user. It must be a column with one of the following [types](data-validation#data-types): `text`, `int(n)`, `uint(n)`, `uuid`, or `json`.

### Choosing the user identifier

It is advisable that **the user identifier be unique** for each user found on the source connection (be it a file or a set of values returned by a database query), as this value is used to identify the identities. If you use an identifier with non-unique values, this can lead to an unexpected overwriting of connection's identities.

For example, if a file has these properties:

| Email | SSN | First Name |
|-------|-----|------------|
| a@b ⚠️ | 123 | John       |
| c@d   | 456 | Paul       |
| a@b ⚠️ | 789 | Jake       |

it is recommended to use the "SSN" property as User Identifier, since it uniquely identifies a user, while the "Email" property is used by multiple users and would cause John's data to be overwritten with Jake's.

### Choosing the user identifier - Multiple actions per connection

It is important to note that **identities with the same User Identifier value, if they come from distinct actions from the same connection, are put together during the [Identity Resolution](./identity-resolution#same-user-criterion) process**.

This means that, if you have **multiple actions for the same connection**, there may be two distinct scenarios:

* if the actions all read **a property that has the same meaning and the same data set on all the actions**, then it is desirable to **choose that property as User Identifier**, so that the Identity Resolution puts the corresponding identities together

* if the action reads **properties with different meanings**, then it is important that the set of values for each property is different from the set of values of the others (for example, one is a tax code, the other is an email), to avoid unexpected collisions and consequent incorrect merges between multiple identities. If this condition cannot be ensured, then **it is recommended to create separate connections to import users**, so that the merge is then handled exclusively by the identifiers of the Identity Resolution.

## Last change time column

Actions on **file storage** and **database** source connections **may** indicate a column containing a timestamp indicating the last change time of the user. It must be a column with one of the following [types](data-validation#data-types): `text`, `datetime`, `date`, or `json`.

If a last change time column is provided and its type is `text` or `json`, a timestamp format must be provided for parsing its value.

## Anonymous user identities

It is possibile to import anonymous identities through actions that listen to events.

### Deletion of anonymous identities

When a non-anonymous identity is imported from a connection, all anonymous identities with the same Anonymous ID which have been previously imported from the same connection are deleted, and only the non-anonymous identity is retained.

### How anonymous identities are stored in the warehouse

In the data warehouse, anonymous identities are stored in the `_user_identities` table, just like non-anonymous ones.

Note that the anonymous ID of an anonymous identity is stored in the `__identity_id__` column, while the `__anonymous_ids__` field remains empty. This is an implementation choice.

Once an anonymous identity is resolved and its identity ID is determined (for example, if an *identify* event arrives), the anonymous ID is moved to `__anonymous_ids__`, and the `__identity_id__` column will then contain the actual identity ID.

## Empty objects when importing identities

If a transformation returns an empty Object as the value for a property of an identity, this is equivalent to not returning any value for that property.

Therefore, if a transformation returns something like:

```json
{
    "a": 10,
    "b": { },
    "c": "Hello!"
}
```

When the user is read, it will be returned as:

```json
{
    "a": 10,
    "c": "Hello!"
}
```

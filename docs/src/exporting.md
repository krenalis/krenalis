{% extends "/layouts/doc.html" %}
{% macro Title string %}Export{% end %}
{% Article %}

# Export

## Exporting users to apps

The export of Meergo users to an app is performed through a user export action on an app connection.

When executed, this action determines the matches between Meergo users and app users, and it then updates the app users' properties specified in the action (or creates a new user if one did not already exist in the app) with the values returned by the action's transformation.

In this scenario, therefore, **Meergo represents the authoritative source of data on the app**, **limited to the exported users** (determined by the filter, the matching and the export mode) and **limited to the properties exported to the app** (determined by the action transformation and the app connector).

> Note that it is possible to specify for an export action whether to update only users already existing on the app, create only new ones, or perform both operations,s depending on the match result.

### Users matching

Exporting users to an app occurs by determining, through the matching of values of **a property of Meergo users** and **a property of users in the app**, which Meergo users correspond to the users of the app.

```
   Property of Meergo's users            Property of app's users
 ┌────────────────────────────┐         ┌─────────────────────────────┐
 │                            │    =    │                             │
 └────────────────────────────┘         └─────────────────────────────┘
```

The property of app's users cannot be transformed; it is Meergo, in case of creation, to export a value for that property based on the property of Meergo's users.

### Users conflicts

When exporting to an app, **two different types of conflicts** can occur, which are handled differently.

| Case                                                                             | Consequences                                                                                                 |
|----------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------|
| **Multiple users within Meergo** have the same value for the specified property. | **These users are not exported** and an error is shown.                                                      |
| **Multiple users on the app** have the same value for the specified property.    | Whether to proceed with the export of these users or not **depends on the configuration** set in the action. |

### How matching occurs

The comparison is done by comparing the string representations of the user property values in Meergo with the string representations of the user property values in the app.

For this reason, it is also possible to compare properties with different types (e.g. `text` and `uuid`, `int` and `uint`, etc…), as values with different types could still have the same string representation and match.

**For example**

* a Meergo user has a property `my_app_id` with type `uuid` and value `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`
* an user on the app has a property `custom_id` with type `text` and value `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`

Even if the types of the two properties are different, the two values represented as strings are:

* `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`
* `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`

And this determines the fact that, in the example, the Meergo user matches with the user on the app.

## Exporting users to databases

It is possible to export users from Meergo to a database table, creating or updating them.

> Note that exporting to the database **does not handle the deletion of users** who are no longer present in Meergo. Such users will remain in the table to which they have been exported, and it is the user's responsibility to remove them if necessary.

When exporting users to a database, both a **table name** and a **table key property** must be specified.

The table key property must be a property of the table, and it will be used as key for the insert / update queries on the database. It must have one of the following types:

* `text`
* `int(n)`
* `uint(n)`
* `uuid`

A value for the table key property must be returned by the transformation, as, otherwise, would be impossible to match the user with the users on the database's table.

> **Table keys and primary keys**. For some database connectors, for example, MySQL, it is the user's responsibility to choose the primary key of the table as the table key property, otherwise the export won't behave consistently. In this regard, refer to the specific documentation for each database.

## Exporting users to files

### Ordering

When exporting users to a file, the order of the users is indicated through an action setting, where you can choose a property path to sort users by.

This property must have one of these types:

* `text`
* `int(n)`
* `uint(n)`
* `decimal(p,s)`, but only if scale `s` is 0
* `uuid`
* `inet`

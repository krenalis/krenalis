# Export

## Exporting Users to Apps

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

### How Matching Occurs

The comparison is done by comparing the string representations of the user property values in Meergo with the string representations of the user property values in the app.

For this reason, it is also possible to compare properties with different types (e.g. `Text` and `UUID`, `Int` and `UInt`, etc…), as values with different types could still have the same string representation and match.

**For example**

* a Meergo user has a property `my_app_id` with type `UUID` and value `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`
* an user on the app has a property `custom_id` with type `Text` and value `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`

Even if the types of the two properties are different, the two values represented as strings are:

* `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`
* `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`

And this determines the fact that, in the example, the Meergo user matches with the user on the app.

## Exporting Users To Databases

When exporting users to a database, both a **table name** and a **table key property** must be specified.

The table key property must be a property of the table, and it will be used as key for the insert / update queries on the database. It must have one of the following types:

* `Int(n)`
* `UInt(n)`
* `UUID`
* `Text`

A value for the table key property must be returned by the transformation, as, otherwise, would be impossible to match the user with the users on the database's table.

> Note that for some database connectors, for example, MySQL, it is the user's responsibility to choose the primary key of the table as the table key property, otherwise the export won't behave consistently. In this regard, refer to the specific documentation for each database.

## Exporting Users To Files

### Ordering

When exporting users to a file, the order of the users is indicated through an action setting, where you can choose a property path to sort users by.

This property must have one of these types:

* `Int(n)`
* `UInt(n)`
* `Decimal(p,s)`, but only if scale `s` is 0
* `UUID`
* `Inet`
* `Text`

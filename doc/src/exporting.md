# Export

## Exporting Users to Apps

Exporting users to an app occurs by determining, through the matching of values of **a property of Chichi users** and **a property of users in the app**, which Chichi users correspond to the users of the app.

```
   Property of Chichi's users            Property of app's users
 ┌────────────────────────────┐         ┌─────────────────────────────┐
 │                            │    =    │                             │
 └────────────────────────────┘         └─────────────────────────────┘
```

### Users conflicts

When exporting to an app, **two different types of conflicts** can occur, which are handled differently.

| Case                                                                             | Consequences                                                                                  |
|----------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------|
| **Multiple users within Chichi** have the same value for the specified property. | The **export is not initiated** and an error is shown.                                        |
| **Multiple users on the app** have the same value for the specified property.    | Whether to proceed with the export or not **depends on the configuration** set in the action. |

### Allowed Types For Properties

Both matching properties must have one of the following types (but not necessarily the same one):

* `Int(n)`
* `UInt(n)`
* `UUID`
* `Text`

### How Matching Occurs

The comparison is done by comparing the JSON representations of the user property values in Chichi with the JSON representations of the user property values in the app.

For this reason, it is also possible to compare properties with different types (e.g. `Text` and `UUID`, `Int` and `UInt`, etc…), as values with different types could still have the same JSON representation and match.

**For example**

* a Chichi user has a property `my_app_id` with type `UUID` and value `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`
* an user on the app has a property `custom_id` with type `Text` and value `7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd`

Even if the types of the two properties are different, the two values represented in JSON are:

* `"7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd"`
* `"7315ad40-fbe9-4ae3-80eb-6fd06f22f1fd"`

And this determines the fact that, in the example, the Chichi user matches with the user on the app.

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

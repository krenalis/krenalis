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

## Exporting Users To Files

### Ordering

When exporting users to a file, the ordering of users is according to *the first property mapped and/or transformed*.

> NOTE: this behavior is only a temporary workaround and will be changed. See the [issue 757](https://github.com/open2b/chichi/issues/757).
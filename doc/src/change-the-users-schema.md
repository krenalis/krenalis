# Change the "users" schema

The "users" schema can be modified both: 

* through the UI
* by using the "chichi-cli" tool from the command line.

When changing the `users` schema, these operations are supported:

* add a new property to the schema, necessarily at top level (so already existent Object properties cannot be extended with new properties) and at the end of the already present properties.
* dropping a property
* renaming a property, that is changing a property name without altering its position in the schema
* changing the label or the description of a property

Any other operation (as changes in the order of the properties, or the change of a type or nullability) is not supported.

Further limits may be introduced by data warehouses. See [Data Warehouse](./data-warehouse.md) and its subsections.

## Object properties and nullability

Properties with type Object cannot be "nullable", as this would lead to confusion and representation issues regarding type and values in various data warehouses.
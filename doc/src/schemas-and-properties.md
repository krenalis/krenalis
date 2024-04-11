# Schemas and properties

## Changing the `users` schemas

The `users` schema can be changed through the `chichi-cli` tool.

When changing the `users` schema, these operations are supported:

* add a new property to the schema, necessarily at top level (so already existent Object properties cannot be extended with new properties) and at the end of the already present properties. In case of Object properties, these cannot be nullable.
* dropping a property
* renaming a property, that is changing a property name without altering its position in the schema
* changing the label or the description of a property

Any other operation (as changes in the order of the properties, or the change of a type or nullability) is not supported.

Further limits may be introduced by data warehouses. See [Data Warehouse](./data-warehouse.md) and its subsections.
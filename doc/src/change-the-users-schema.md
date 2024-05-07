# Change the "users" schema

## How to change the users schema

The "users" schema can be modified both: 

* through the UI
* by using the "chichi-cli" tool from the command line.

## Supported operations

When changing the `users` schema, these operations are supported:

* adding a new property to the schema
* dropping a property
* renaming a property
* changing the order of the properties
* changing the label or the description of a property

Any other operation (as changing a property type or nullability) is not supported.

Further limits may be introduced by data warehouses. See [Data Warehouse](./data-warehouse.md) and its subsections.

> NOTE: altering of Object properties is not fully implemented. See the issue [#722](https://github.com/open2b/chichi/issues/722).

> NOTE: renaming of Object properties is not implemented. See the issue [#691](https://github.com/open2b/chichi/issues/691).

## Object properties and nullability

Properties with type Object cannot be "nullable", as this would lead to confusion and representation issues regarding type and values in various data warehouses.
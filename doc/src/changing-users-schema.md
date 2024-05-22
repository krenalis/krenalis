# Changing Users Schema

The user schema associated with a newly created workspace contains only one property, `email`. This schema should be modified with operations such as adding, removing, and modifying properties to reflect the model that best represents the users you want to manage in Chichi.

## How to change the users schema

The "users" schema can be modified both: 

* through the UI
* by using the "chichi-cli" tool from the command line.

## Supported operations

When changing the `users` schema, these operations are supported:

* **adding** properties **at any level**
* **dropping** properties **at any level**
* **renaming** properties **at any level**
* **reordering** properties **at top level** and **within Object properties**
* **changing labels** and **descriptions** of properties **at any level**

Any other operation (as changing a property type or nullability) is not supported.

## Properties and types limitations

Here are the restrictions on properties and their types imposed directly by Chichi, which apply regardless of the data warehouse used. Each data warehouse may further restrict the supported types (see [Data Warehouse](./data-warehouse.md) and its subsections for more details on this).

These are the limits imposed by Chichi:

* Array types cannot have items of type Array, Object, or Map.
* Map types cannot have values of type Array, Object, or Map.
* Properties with type Object cannot be "nullable", as this would lead to confusion and representation issues regarding type and values in various data warehouses.
* Properties cannot specify a placeholder
* Properties cannot be required
* Properties cannot specify a role

## Conflicting properties

The "users" schema cannot contain conflicting properties, meaning properties whose representations as columns in the data warehouse would have the same column name.

For example, this schema:

```
x {
    a
    b
}
x_a
```

is not valid because it contains two conflicting properties: `x.a` and `x_a`, as both should be represented by a column named `x_a` in the data warehouse, which would be impossible.

For more details on how properties are represented as columns, see [the dedicated section](./data-warehouse.md#properties-to-columns-name).
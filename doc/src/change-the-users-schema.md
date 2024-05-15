# Change the "users" schema

## How to change the users schema

The "users" schema can be modified both: 

* through the UI
* by using the "chichi-cli" tool from the command line.

## Supported operations

When changing the `users` schema, these operations are supported:

* **adding** properties **at any level**
* **dropping** properties **at any level**
* **renaming** properties **at any level**
* **reordering** properties (excluding properties of Objects, see the issue [#739](https://github.com/open2b/chichi/issues/739)).
* **changing labels** and **descriptions** of properties **at any level**

Any other operation (as changing a property type or nullability) is not supported.

> Note that further limits may be introduced by data warehouses. See [Data Warehouse](./data-warehouse.md) and its subsections.

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

## Object properties and nullability

Properties with type Object cannot be "nullable", as this would lead to confusion and representation issues regarding type and values in various data warehouses.
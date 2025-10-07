{% extends "/layouts/doc.html" %}
{% macro Title string %}Data warehouses{% end %}
{% Article %}

# Data warehouses

## Supported types

The types of Meergo supported by a certain data warehouse are those for which it is possible to define a column in the `users` table that ensures values cannot be written (and consequently returned) that do not conform to the Meergo type.

For example, the `text` type with a limit of 50 characters is supported in PostgreSQL because it is possible to define a `varchar(50)` column that prevents the insertion of strings longer than 50 characters, while the `text` type with a limit of 50 bytes is not supported because it is not possible to declare a column with such a constraint.

## Properties to columns

Properties of `object` types are transformed into columns starting with the `object` property name as prefix, followed by `_` (underscore), then followed by the property name.

So, for example, this `object` property:

```plain
address {
    street {
        name
        number
    }
    city
    country
}
```

is represented in the data warehouse as:

```plain
address_street_name
address_street_number
address_city
address_country
```

### How Meergo stores data in the warehouse (schemas, tables, naming conventions, partitioning, indexes).

### The difference between batch vs. real-time ingestion into the warehouse.

### How event and user data are structured once inside the warehouse.

### Best practices for querying the warehouse directly (e.g., SQL examples, joining events with user profiles).

### How Meergo integrates with existing warehouse governance (schemas, access control, compliance).

### Performance considerations (incremental loads, large-scale data).

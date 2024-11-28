{% extends "/layouts/doc.html" %}
{% macro Title string %}Data Warehouse{% end %}
{% Article %}

# Data Warehouse

## Supported types

The types of Meergo supported by a certain data warehouse are those for which it is possible to define a column in the `users` table that ensures values cannot be written (and consequently returned) that do not conform to the Meergo type.

For example, the `Text` type with a limit of 50 characters is supported in PostgreSQL because it is possible to define a `varchar(50)` column that prevents the insertion of strings longer than 50 characters, while the `Text` type with a limit of 50 bytes is not supported because it is not possible to declare a column with such a constraint.

## Properties To Columns

Properties of Object types are transformed into columns starting with the Object property name as prefix, followed by `_` (underscore), then followed by the property name.

So, for example, this Object property:

```
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

```
address_street_name
address_street_number
address_city
address_country
```
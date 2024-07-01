# Data Warehouse

## Default values

The columns created by Chichi always have a default value, that is `NULL` (in case of nullable properties) or the zero of the type otherwise.

## Supported types

The types of Chichi supported by a certain data warehouse are those for which it is possible to define a column in the `users` table that ensures values cannot be written (and consequently returned) that do not conform to the Chichi type.

For example, the `Text` type with a limit of 50 characters is supported in PostgreSQL because it is possible to define a `varchar(50)` column that prevents the insertion of strings longer than 50 characters, while the `Text` type with a limit of 50 bytes is not supported because it is not possible to declare a column with such a constraint.

## Properties to columns name

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

Properties with type Object are transformed into columns starting with the same prefix, for example:

```
address_street_name
address_street_number
address_city
address_country
```

#### Objects with "nullable" properties

Properties of type Object can never be "nullable", but the properties of the Object may be.

In such cases, "nullable" properties are represented in the data warehouse as "nullable" columns.

For example, the `ios` property defined this way:

```
ios {
    id (nullable)
    idfa (nullable)
}
```

is represented as:

```
ios_id (nullable)
ios_idfa (nullable)
```

# Data Warehouse

## Naming Requirements

The column names in a table must adhere to the following criteria:

* only lower case letters (a-z), numbers (0-9), and the underscore (_).
* cannot start with a number.

A column with a name starting and ending with two underscores (__), such as `__cluster__`, is completely hidden, and as such, it is as if it were not present in the table.

## Other requirements

All columns in a table must either be declared with a `NULL` option or have a default value assigned. If a column is declared as `NOT NULL` without a default value, an error occurs when attempting to create a row without a value for that particular column.

Additional requirements are dictated by the specific type of data warehouse.

## Column names to properties

Consecutive columns starting with the same prefix are grouped under a single property. For example, the following columns:

```
address_street_name
address_street_number
address_city
address_country
```

are represented as a single property `address`:

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
When a column, or part of it, is not grouped with other columns, its name is converted from snake_case to camelCase. For example, the columns:

```
first_name
last_name
```

are represented as properties:

```
firstName
lastName
```

#### Nullability of Objects

In the case that a property with type Object has been obtained through the grouping of columns, such property will never be nullable, regardless of the nullability of the individual columns.

For example:

```
ios_id (nullable)
ios_idfa (nullable)
```

is represented as:

```
ios {
    id (nullable)
    idfa (nullable)
}
```

where the `ios` property is non-nullable.

## Meta properties

Properties representing columns with names starting with an underscore are referred to as meta properties and are not writable during transformations. Such properties start with an uppercase letter. For example the column

```
_anonymous_ids
```

is represented by the meta property

```
AnonymousIds
```

## Hidden columns

To hide a column so that it cannot be read or written, append an underscore to its name. For example, the column `middle_name_` is hidden:

```
first_name
middle_name_
last_name
```

As a result, the properties corresponding to the three previous columns becomes:

```
firstName
lastName
```


A hidden column does not break the grouping rule. For example, the following columns:

```
point_x
point_y
```

correspond to a single property:

```
point {
    x
    y
}
```

If you hide one of the properties of `point`, for example, hiding the `point_y` column:

```
point_x
point_y_
```

The `point` property continues to be an object, but with one less property:

```
point {
    x
}
```
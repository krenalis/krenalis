# Data Warehouse

## Default values

The columns created by Chichi always have a default value, that is `NULL` (in case of nullable properties) or the zero of the type otherwise.

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

Properties with type Object are transformed into consecutive columns starting with the same prefix, for example:

```
address_street_name
address_street_number
address_city
address_country
```

Property names are converted from camelCase to snake_case. For example, the property names:

```
firstName
lastName
```

are represented as columns:

```
first_name
last_name
```


#### Nullability of Objects

> This section **may be obsolete**. See the issue [#574](https://github.com/open2b/chichi/issues/574).

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

> This section **may be obsolete**. See the issue [#573](https://github.com/open2b/chichi/issues/573).

Properties representing columns with names starting with an underscore are referred to as meta properties and are not writable during transformations. Such properties start with an uppercase letter. For example the column

```
_anonymous_ids
```

is represented by the meta property

```
AnonymousIds
```
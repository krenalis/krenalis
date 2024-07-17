# PostgreSQL Driver

## Data Types

The table below provides a list of supported property types when using a PostgreSQL data warehouse, along with the corresponding column types that are generated on the data warehouse.

| Property Type       | Column Type                                                                                        |
|---------------------|----------------------------------------------------------------------------------------------------|
| `Bool`              | [`boolean`](https://www.postgresql.org/docs/current/datatype-boolean.html)                         |
| `Int(16)`           | [`smallint`](https://www.postgresql.org/docs/current/datatype-numeric.html)                        |
| `Int(32)`           | [`integer`](https://www.postgresql.org/docs/current/datatype-numeric.html)                         |
| `Int(64)`           | [`bigint`](https://www.postgresql.org/docs/current/datatype-numeric.html)                          |
| `Float(32)`         | [`real`](https://www.postgresql.org/docs/current/datatype-numeric.html#DATATYPE-FLOAT)             |
| `Float(64)`         | [`double precision`](https://www.postgresql.org/docs/current/datatype-numeric.html#DATATYPE-FLOAT) |
| `Decimal(p,s)`      | [`decimal(p,s)`](https://www.postgresql.org/docs/current/datatype-numeric.html)                    |
| `DateTime`          | [`timestamp without time zone`](https://www.postgresql.org/docs/current/datatype-datetime.html)    |
| `Date`              | [`date`](https://www.postgresql.org/docs/current/datatype-datetime.html)                           |
| `Time`              | [`time without time zone`](https://www.postgresql.org/docs/current/datatype-datetime.html)         |
| `UUID`              | [`uuid`](https://www.postgresql.org/docs/current/datatype-uuid.html)                               |
| `JSON`              | [`jsonb`](https://www.postgresql.org/docs/current/datatype-json.html)                              |
| `Text`              | [`varchar`](https://www.postgresql.org/docs/current/datatype-character.html)                       |
| `Array(T)` [^array] | [`T[]`](https://www.postgresql.org/docs/current/arrays.html)                                       |

[^array]: where `T` is not `Array`

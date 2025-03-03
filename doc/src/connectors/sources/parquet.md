{% extends "/layouts/doc.html" %}
{% macro Title string %}Parquet data source{% end %}
{% Article %}

# Parquet data source

The Parquet data source allows you to import user data from Parquet files, which can then be unified as users within Meergo.

> Before adding a Parquet data source, ensure that you have configured a storage data source such as S3, SFTP, or HTTP Files. If you haven’t set up a storage source yet, please do so before proceeding with the Parquet file import.

### On this page

- [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)
- [How Parquet columns are imported](#how-parquet-columns-are-imported)
  - [Physical types](#physical-types)
  - [Logical and Converted types](#logical-and-converted-types)
  - [Column groups](#column-groups)

### Import users into the workspace's data warehouse

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the storage data source from which you want to import the Parquet file.
3. If there are no actions, click  **Add**; otherwise, click  **Add new action**.
4. From the **Format** menu, select **Parquet**.
5. In the **Path** field, enter the path of the Parquet file relative to the storage root path. When you enter the relative path, the absolute path of the file will be displayed, allowing you to verify that the path you have entered is correct.
6. Optional: In the **Compression** field, if the JSON file is compressed, select the compression format; Meergo automatically decompresses the file upon reading.
7. Click **Preview** to view a preview of the file.
8. Click **Confirm** to confirm your selections. You can change them at any time later if needed.

### How Parquet columns are imported

This section summarizes how Parquet column types are imported into Meergo.

#### Physical types

This table describes how Parquet physical types (without any logical type annotations) are imported into Meergo.

| Parquet Type           | Imported in Meergo as |
|------------------------|-----------------------|
| `BOOLEAN`              | `boolean`             |
| `INT32`                | `int(32)`             |
| `INT64`                | `int(64)`             |
| `INT96`                | `datetime` [^int96]   |
| `FLOAT`                | `float(32)`           |
| `DOUBLE`               | `float(64)`           |
| `BYTE_ARRAY`           | `text`                |
| `FIXED_LEN_BYTE_ARRAY` | `text`                |

#### Logical and Converted types

This table describes how Parquet logical and converted types are imported into Meergo.

| Parquet Type       | Imported in Meergo as                  |
|--------------------|----------------------------------------|
| `STRING`           | `text`                                 |
| `ENUM`             | `text`                                 |
| `UUID`             | `uuid`                                 |
| `INT(8, true)`     | `int(8)`                               |
| `INT(16, true)`    | `int(16)`                              |
| `INT(32, true)`    | `int(32)`                              |
| `INT(64, true)`    | `int(64)`                              |
| `INT(8, false)`    | `uint(8)`                              |
| `INT(16, false)`   | `uint(16)`                             |
| `INT(32, false)`   | `uint(32)`                             |
| `INT(64, false)`   | `uint(64)`                             |
| `INT_8`            | `int(8)`                               |
| `INT_16`           | `int(16)`                              |
| `INT_32`           | `int(32)`                              |
| `INT_64`           | `int(64)`                              |
| `UINT_8`           | `uint(8)`                              |
| `UINT_16`          | `uint(16)`                             |
| `UINT_32`          | `uint(32)`                             |
| `UINT_64`          | `uint(64)`                             |
| `DECIMAL`          | Not supported [^decimal_support]       |
| `FLOAT16`          | Not supported                          |
| `DATE`             | `date`                                 |
| `TIME`             | `time`                                 |
| `TIME_MILLIS`      | `time`                                 |
| `TIME_MICROS`      | `time`                                 |
| `TIMESTAMP`        | `datetime`                             |
| `TIMESTAMP_MILLIS` | Not supported [^timestamp_milli_micro] |
| `TIMESTAMP_MICROS` | Not supported [^timestamp_milli_micro] |
| `INTERVAL`         | Not supported                          |
| `JSON`             | `json`                                 |
| `BSON`             | `json`                                 |
| `VARIANT`          | Not supported                          |
| `GEOMETRY`         | Not supported                          |
| `GEOGRAPHY`        | Not supported                          |
| `LIST`             | Not supported [^list_support]          |
| `MAP`              | Not supported [^map_support]           |
| `UNKNOWN`          | Not supported                          |

#### Column groups

Import of columns groups is currently not supported.

[^int96]: `INT96` types are always treated as `datetime` Meergo types, because that is in fact how they are used in the Parquet files. However, please note that this type of representation is deprecated, and is kept in the Parquet connector only for compatibility with older Parquet files.
[^decimal_support]: Support for importing `DECIMAL` columns is discussed here: https://github.com/meergo/meergo/issues/1370
[^list_support]: Support for importing `LIST` columns is discussed here: https://github.com/meergo/meergo/issues/1325
[^map_support]: Support for importing `MAP` columns is discussed here: https://github.com/meergo/meergo/issues/1371
[^timestamp_milli_micro]: Support for importing `TIME_MILLIS` and `TIMESTAMP_MICROS` is discussed here: https://github.com/meergo/meergo/issues/1385
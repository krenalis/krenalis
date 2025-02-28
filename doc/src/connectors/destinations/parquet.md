{% extends "/layouts/doc.html" %}
{% macro Title string %}Parquet data destination{% end %}
{% Article %}

# Parquet data destination

The Parquet data destination allows you to export unified users (i.e., users consolidated through identity resolution) into a Parquet file and save it to a storage location, such as S3 or SFTP.

> Before adding a Parquet data destination, ensure that you have configured a storage data destination such as S3, SFTP, or HTTP Files. If you haven’t set up a storage destination yet, please do so before proceeding with the Parquet file export.

### On this page

- [Add a Parquet data destination](#add-a-parquet-data-destination)
- [How Meergo types are exported to Parquet](#how-meergo-types-are-exported-to-parquet)

### Add a Parquet data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. Click on the storage data destination where you want to export the Parquet file.
3. If there are no actions, click **Add**; otherwise, click **Add new action**.
4. From the **Format** menu, select **Parquet**.
5. In the **Path** field, enter the path of the Parquet file, relative to the storage root path. The absolute path will be displayed so you can verify its accuracy.
6. Optionally, proceed with the other fields:
    * **Compression**: Compression format. Select a format if you want the Parquet file to be compressed.
    * **Order users by**: Sorting of users. Select a property if you want the users to be written in ascending order based on this property.
7. Click **Add** to add the action.

### How Meergo types are exported to Parquet

The following table shows how the user property types in Meergo are mapped to the column types in the exported Parquet file:

| Type of user property in Meergo | Physical Type of exported Parquet column | Logical Type of exported Parquet column       |
|---------------------------------|------------------------------------------|-----------------------------------------------|
| `boolean`                       | `BOOLEAN`                                | *(none)*                                      |
| `int(8)`                        | `INT32`                                  | `INT(8, true)`                                |
| `int(16)`                       | `INT32`                                  | `INT(16, true)`                               |
| `int(24)`                       | `INT32`                                  | *(none)*                                      |
| `int(32)`                       | `INT32`                                  | *(none)*                                      |
| `int(64)`                       | `INT36`                                  | *(none)*                                      |
| `uint(8)`                       | `INT32`                                  | `INT(8, false)`                               |
| `uint(16)`                      | `INT32`                                  | `INT(16, false)`                              |
| `uint(24)`                      | `INT32`                                  | `INT(32, false)`                              |
| `uint(32)`                      | `INT64`                                  | `INT(32, false)`                              |
| `uint(64)`                      | `INT64`                                  | `INT(64, false)`                              |
| `float(32)`                     | `FLOAT`                                  | *(none)*                                      |
| `float(64)`                     | `DOUBLE`                                 | *(none)*                                      |
| `decimal(p, s)`                 | Not supported [^decimal_support]         | -                                             |
| `datetime`                      | `INT64`                                  | `TIMESTAMP(isAdjustedToUTC=true, unit=NANOS)` |
| `date`                          | `INT32`                                  | `DATE`                                        |
| `time`                          | Not supported [^time_date_support]       | -                                             |
| `year`                          | `INT32`                                  | *(none)*                                      |
| `uuid`                          | `FIXED_LEN_BYTE_ARRAY` with length 16    | `UUID`                                        |
| `json`                          | `BYTE_ARRAY`                             | `JSON`                                        |
| `inet`                          | `BYTE_ARRAY`                             | `STRING`                                      |
| `text`                          | `BYTE_ARRAY`                             | `STRING`                                      |
| `array`                         | Not supported [^array_support]           | -                                             |
| `object`                        | *(column groups)*                        | -                                             |
| `map`                           | Not supported [^map_support]             | -                                             |

[^decimal_support]: Support for decimal properties is discussed here: https://github.com/meergo/meergo/issues/1370
[^time_date_support]: Support for time properties is discussed here: https://github.com/meergo/meergo/issues/1376
[^array_support]: Support for array properties is discussed here: https://github.com/meergo/meergo/issues/1325
[^map_support]: Support map properties is discussed here: https://github.com/meergo/meergo/issues/1371
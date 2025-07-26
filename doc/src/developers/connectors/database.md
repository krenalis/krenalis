{% extends "/layouts/doc.html" %}
{% macro Title string %}Database Connectors{% end %}
{% Article %}

# Database connectors

Database connectors allow to connect to databases (DBMS), such as PostgreSQL, MySQL, or Snowflake, to execute queries to import and export data.

Database connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and methods.

## Quick start

In the creation of a new Go module, for your database connector, you can utilize the following template by pasting it into a Go file. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package postgresql implements the PostgreSQL database connector.
package postgresql

import (
    "context"

    "github.com/meergo/meergo"
    "github.com/meergo/meergo/types"
)

func init() {
    meergo.RegisterDatabase(meergo.DatabaseInfo{
        Name:        "PostgreSQL",
        Categories:  meergo.CategoryDatabase,
        SampleQuery: "SELECT *\nFROM users\nWHERE ${last_change_time}\n",
    }, New)
}

type PostgreSQL struct {
    // Your connector fields.
}

// New returns a new PostgreSQL connector instance.
func New(env *meergo.DatabaseEnv) (*PostgreSQL, error) {
    // ...
}

// Close closes the database.
func (ps *PostgreSQL) Close() error {
    // ...
}

// Columns returns the columns of the given table.
func (ps *PostgreSQL) Columns(ctx context.Context, table string) ([]meergo.Column, error) {
    // ...
}

// Merge performs batch insert and update operations on the specified table,
// basing on the table keys.
func (ps *PostgreSQL) Merge(ctx context.Context, table meergo.Table, rows [][]any) error {
    // ...
}

// Query executes the given query and returns the resulting rows and columns.
func (ps *PostgreSQL) Query(ctx context.Context, query string) (meergo.Rows, []meergo.Column, error) {
    // ...
}

// QuoteTime returns a quoted time value for the specified type or "NULL" if the
// value is nil.
func (ps *PostgreSQL) QuoteTime(value any, typ types.Type) string {
    // ...
}
```

## Implementation

Let's explore how to implement a database connector, for example for PostgreSQL.

First create a Go module:

```sh
$ mkdir postgresql
$ cd postgresql
$ go mod init postgresql
```

Then add a Go file to the new directory. For example copy the previous template file.

Later on, you can [build an executable with your connector](../../getting-started#build-with-your-custom-connectors).

### About the connector

The `DatabaseInfo` type describes information about the database connector:

- `Name`: short name, typically the name of the DBMS. For example, "PostgreSQL", "MySQL", "Snowflake", etc.
- `Categories`: the categories that the connector falls into. There must be at least one category.
- `SampleQuery`: sample query displayed in the query editor when creating a new database source action.
- `TimeLayouts`: layouts for the `datetime`, `date`, and `time` values when they are represented as strings. See [Time Layouts](data-values#time-layouts) in [Data Values](data-values) for more details.
- `Icon`: icon in SVG format representing the DBMS. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterDatabase` function that, executed during package initialization, registers the database connector:

```go
func init() {
    meergo.RegisterDatabase(meergo.DatabaseInfo{
        Name:        "PostgreSQL",
        SampleQuery: "SELECT *\nFROM users\nWHERE ${last_change_time}\n",
        Icon:        icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterDatabase` function is the function utilized for creating a connector instance:

```go
func New(env *meergo.DatabaseEnv) (*PostgreSQL, error)
```

This function accepts a database environment and yields a value representing your custom type.

The structure of `DatabaseEnv` is outlined as follows:

```go
// DatabaseEnv is the environment for a database connector.
type DatabaseEnv struct {

    // Settings holds the raw settings data.
    Settings []byte

    // SetSettings is the function used to update the settings.
    SetSettings SetSettingsFunc
}
```

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSetting`: A function that enables the connector to update its settings as necessary.

### Close method

```go
Close() error
```

The `Close` method is invoked by Meergo when no calls to the connector instance's methods are in progress and no more will be made, so the connector can close any connections eventually opened with the database.

### Columns method

```go
Columns(ctx context.Context, table string) ([]meergo.Column, error)
```

Meergo invokes the `Columns` method when creating or updating a database destination action, retrieving the columns of the table to which data should be exported.

The `Columns` method returns the table's columns as a slice of `Column` values, detailing the names and types of each column:

```go
// Column represents a database table column.
type Column struct {
    Name     string     // column name
    Type     types.Type // data type of the column
    Nullable bool       // true if the column can contain NULL values
    Writable bool       // true if the column is writable
    Issue    string     // issue message
}
```

#### Handling column issues

If a column's type is not supported, its name is not a valid property name, or any other issue occurs with the column, leave `Column.Type` unset. Likewise, leave the other fields unset, as they are not relevant in this case, and describe the issue in `Column.Issue`.

Such a column will not appear among the available table columns. However, the issue will be brought to the user's attention without preventing the use of the other columns.

The following are examples of common issue messages used by database connectors:

* _Column "perf" has an unsupported type "INT96"._
* _Column "score:value " does not have a valid property name. Valid names start with a letter or underscore, followed by only letters, numbers, or underscores._
* _Column "amount" has a precision of 100, which exceeds the maximum supported precision of 76._
* _Column "value" has a scale of 50, which exceeds the maximum supported precision of 37._

### Merge method

```go
Merge(ctx context.Context, table meergo.Table, rows [][]any) error
```

The `Merge` method is used by Meergo during data export to a database. It updates existing rows if matching keys are found, or inserts new rows into the specified table.

The `table` parameter provides details about the table to update, including its name, columns, and keys. Defined as:

```go
type Table struct {
    Name    string
    Columns []Column
    Keys    []string
}
```

- `Name`: The name of the table.
- `Columns`: The columns in the table that need to be updated. It may not include all the columns in the table.
- `Keys`: The columns that serve as keys for the table. This typically includes the primary key but does not have to. The columns specified in `table.Keys` are also included in `table.Columns`.

The `rows` parameter contains the rows to be updated or inserted. For each `row`, `row[i]` contains the value for the column `table.Columns[i]`. If a column is nullable, the corresponding value in a raw can be `nil`, representing the `NULL` value in the database.

Meergo ensures that `table.Columns` (and consequently each row in `rows`) contains at least one additional column besides the table key values.

Furthermore, Meergo ensures that during the entire execution of an export to the database, to the `Merge` method are never passed two or more duplicate rows, meaning rows that have the same value for the table keys.

A database connector can require that the columns in `table.Keys` form the primary key and can return an error if they do not.

### Query method

```go
Query(ctx context.Context, query string) (meergo.Rows, []meergo.Column, []string, error)
```

Meergo invokes the `Query` method when previewing the rows returned by a query while creating or updating a database source action, and to get the data during an import. The query is provided after replacing any placeholders like `${limit}`.

The `Query` method runs the query and gives back two things: the rows themselves, which follow the `Rows` interface, and the columns as a slice of `Column` values. Here's what the `Rows` interface look like:

```go
type Rows interface {
    Close() error
    Err() error
    Next() bool
    Scan(dest ...any) error
}
```

The standard Go library's `sql.Rows` type implements this interface. So, the connector can just return a `sql.Rows` value.

Here's what the `Column` type look like:

```go
// Column represents a database table column.
type Column struct {
    Name     string     // column name
    Type     types.Type // data type of the column
    Nullable bool       // true if the column can contain NULL values
    Writable bool       // true if the column is writable (generally false for columns returned by Query).
    Issue    string     // issue message
}
```

#### Handling column issues

If a column's type is not supported, its name is not a valid property name, or any other issue occurs with the column, leave `Column.Type` unset. Likewise, leave the other fields unset, as they are not relevant in this case, and describe the issue in `Column.Issue`.

Such a column will not appear among the available table columns. However, the issue will be brought to the user's attention without preventing the use of the other columns.

The following are examples of common issue messages used by database connectors:

* _Column "perf" has an unsupported type "INT96"._
* _Column "score:value " does not have a valid property name. Valid names start with a letter or underscore, followed by only letters, numbers, or underscores._
* _Column "amount" has a precision of 100, which exceeds the maximum supported precision of 76._
* _Column "value" has a scale of 50, which exceeds the maximum supported precision of 37._

### QuoteTime method

```go
QuoteTime(value any, typ types.Type) string
```

Meergo calls the `QuoteTime` method to construct the value for the `last_change_time` placeholder, used in a query to implement a cursor that returns rows starting from a specified time.

The `last_change_time` placeholder can either be `NULL` or a timestamp representation. The `QuoteTime` method receives the value and its type, which can be one of `datetime`, `date`, `json`, or `text`. For `datetime` and `date` types, the value is a `time.Time` object set to UTC. For `json` and `text`, the value is a string. If the value is `nil`, `QuoteTime` returns `"NULL"` or the appropriate database representation.

#### Examples

Consider the following query:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= ${last_change_time} OR ${last_change_time} IS NULL 
ORDER BY updated_at
```

The call `QuoteTime(time.Date(2025, 01, 30, 16, 12, 25, 837, time.UTC), types.DateTime())` might return `"'2025-01-30 16:12:25.837'"`, resulting in:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= '2025-01-30 16:12:25.837' OR '2025-01-30 16:12:25.837' IS NULL
ORDER BY updated_at
```

The call `QuoteTime("2025-02-13T16:12:25", types.Text())` might return `"'2025-02-13T16:12:25'"`, resulting in:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= '2025-02-13T16:12:25' OR '2025-02-13T16:12:25' IS NULL
ORDER BY updated_at
```

The call `QuoteTime(nil, time.Time{})` might return `"NULL"`, resulting in:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE updated_at >= NULL OR NULL IS NULL
ORDER BY updated_at
```

In this case, `updated_at >= NULL OR NULL IS NULL` evaluates to `TRUE`, returning all rows as expected.

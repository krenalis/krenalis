{% extends "/layouts/doc.html" %}
{% macro Title string %}Database Connectors{% end %}
{% Article %}

<span>Extend Meergo</span>
# Database connectors

Database connectors allow to connect to databases (DBMS), such as PostgreSQL, MySQL, or Snowflake, to execute queries to import and export data.

Database connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and interfaces.

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
		SampleQuery: "SELECT * FROM users LIMIT ${limit}",
	}, New)
}

type PostgreSQL struct {
	// Your connector fields.
}

// New returns a new PostgreSQL connector instance.
func New(conf *meergo.DatabaseConfig) (*PostgreSQL, error) {
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

// LastChangeTimeCondition returns the query condition used for the
// last_change_time placeholder in the form "column >= value" or, if column is
// empty, a true value.
func (ps *PostgreSQL) LastChangeTimeCondition(column string, typ types.Type, value any) string {
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
- `SampleQuery`: sample query displayed in the query editor when creating a new database source action.
- `TimeLayouts`: layouts for the `DateTime`, `Date`, and `Time` values when they are represented as strings. See [Time Layouts](data-values#time-layouts) in [Data Values](data-values) for more details.
- `Icon`: icon in SVG format representing the DBMS. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterDatabase` function that, executed during package initialization, registers the database connector:

```go
func init() {
    meergo.RegisterDatabase(meergo.DatabaseInfo{
        Name:        "PostgreSQL",
        SampleQuery: "SELECT * FROM users LIMIT ${limit}",
        Icon:        icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterDatabase` function is the function utilized for creating a connector instance:

```go
func New(conf *meergo.DatabaseConfig) (*PostgreSQL, error)
```

This function accepts a database configuration and yields a value representing your custom type.

The structure of `DatabaseConfig` is outlined as follows:

```go
type DatabaseConfig struct {
    Settings    []byte
    SetSettings meergo.SetSettingsFunc
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
}
```

If a column has an unsupported type, return an `*UnsupportedColumnTypeError` error. Use the `NewUnsupportedColumnTypeError` function from the `meergo` package to create this error.  

### LastChangeTimeCondition method

```go
LastChangeTimeCondition(column string, typ types.Type, value any) string
```

Meergo calls the `LastChangeTimeCondition` method to construct the value for the `last_change_time` placeholder. The `last_change_time` placeholder is used in a query to implement a cursor, returning only the rows starting from a specified time.

The value of the `last_change_time` placeholder is a condition that can be used in a `WHERE` statement, like `"updated_at" >= '2024-03-16 09:26:33'`. `LastChangeTimeCondition` receives the name of the column, its type, and the value. The type can only be `DateTime`, `Date`, `JSON`, and `Text`. For the `DateTime` and `Date` types, the value is of type `time.Time` set to UTC, while for the `JSON` and `Text` types, the value is of type `string`. `LastChangeTimeCondition` must construct the condition based on these parameters.

As a special case, if `column` is empty, it must return an always true condition, usually returning `"TRUE"`. This occurs when the query should not limit the rows returned.

#### Examples

Let's take the following query as an example:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE ${last_change_time}
LIMIT ${limit}
```

Suppose the `limit` placeholder is 1000.

The call `LastChangeTimeCondition("updated_at", types.DateTime(), time.Date(2024, 6, 18, 16, 12, 25, 837, time.UTC))` might return `"\"updated_at\" >= '2024-06-18 16:12:25.837'"` and the query would become:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE "updated_at" >= '2024-06-18 16:12:25.837'
LIMIT 1000
```

The call `LastChangeTimeCondition("timestamp", types.Text(), "2014-07-18T16:12:25")` might return `"\"timestamp\" >= '2024-06-18T16:12:25'"` and the query would become:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE "timestamp" >= '2024-06-18T16:12:25'
LIMIT 1000
```

Note that if the value is a string, `LastChangeTimeCondition` simply needs to quote the string.

The call `LastChangeTimeCondition("", time.Time{}, nil)` might return `"TRUE"` and the query would become:

```sql
SELECT first_name, last_name, phone_number
FROM customers
WHERE TRUE
LIMIT 1000
```

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

A database connector can require that the columns in `table.Keys` form the primary key and can return an error if they do not.

### Query method

```go
Query(ctx context.Context, query string) (meergo.Rows, []meergo.Column, error)
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

If a column has an unsupported type, return an `*UnsupportedColumnTypeError` error. Use the `NewUnsupportedColumnTypeError` function from the `meergo` package to create this error.

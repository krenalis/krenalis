# Database Connectors

Database connectors allow to connect to databases (DBMS), such as PostgreSQL, MySQL, or Snowflake, to execute queries to import and export data.

Database connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and interfaces.

## Quick Start

In the creation of a new Go module, for your database connector, you can utilize the following template by pasting it into a Go file. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package postgresql implements the PostgreSQL database connector.
package postgresql

import (
	"context"
	_ "embed"

	"github.com/open2b/chichi"
	"github.com/open2b/chichi/types"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	chichi.RegisterDatabase(chichi.DatabaseInfo{
		Name:        "PostgreSQL",
		SampleQuery: "SELECT * FROM users LIMIT ${limit}",
		Icon:        icon,
	}, New)
}

type PostgreSQL struct {
	// Your connector fields.
}

// New returns a new PostgreSQL connector instance.
func New(conf *chichi.FileConfig) (*PostgreSQL, error) {
	// ...
}

// Close closes the database.
func (ps *PostgreSQL) Close() error {
    // ...
}

// Columns returns the columns of the given table.
func (ps *PostgreSQL) Columns(ctx context.Context, table string) ([]types.Property, error) {
	// ...
}

// Query executes the given query and returns the resulting rows and columns.
func (ps *PostgreSQL) Query(ctx context.Context, query string) (Rows, []types.Property, error) {
	// ...
}

// Upsert creates or updates the provided rows in the specified table.
func (ps *PostgreSQL) Upsert(ctx context.Context, table string, rows []map[string]any, columns []types.Property) error {
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

As you can see, the template file embeds a SVG file, this is the icon that represent the connector. Choose an appropriate SVG icon and put it into the module directory.

### About the Connector

The `DatabaseInfo` type describes information about the database connector:

- `Name`: short name, typically the name of the DBMS. For example, "PostgreSQL", "MySQL", "Snowflake", etc.

- `SampleQuery`: sample query displayed in the query editor when creating a new database source action.

- `Icon`: icon in SVG format representing the DBMS. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterDatabase` function that, executed during package initialization, registers the database connector:

```go
func init() {
    chichi.RegisterDatabase(chichi.DatabaseInfo{
        Name:        "PostgreSQL",
        SampleQuery: "SELECT * FROM users LIMIT ${limit}",
        Icon:        icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterDatabase` function is the function utilized for creating a connector instance:

```go
func New(conf *chichi.DatabaseConfig) (*PostgreSQL, error)
```

This function accepts a database configuration and yields a value representing your custom type. A connector can be instantiated either as a source or a destination, but not both simultaneously. Consequently, an instance of a connector will be responsible for either reading or writing to a database, depending on its role.

### Database Configuration

The structure of `DatabaseConfig` is outlined as follows:

```go
type DatabaseConfig struct {
    Role        chichi.Role
    Settings    []byte
    SetSettings chichi.SetSettingsFunc
}
```

- `Role`: Specifies the intended role of the resulting instance, which can be either `Source` or `Destination`.

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.

- `SetSetting`: A function that enables the connector to update its settings as necessary.

### Close method

```go
Close() error
```

The `Close` method is invoked by Chichi when no calls to the connector instance's methods are in progress and no more will be made, so the connector can close any connections eventually opened with the DBMS.

### Columns method

```go
Columns(ctx context.Context, table string) ([]types.Property, error)
```

Chichi invokes the `Columns` method when creating or updating a database destination action, retrieving the columns of the table to which data should be exported.

The `Columns` method returns the table's columns as a slice of `Property` values, detailing the names and types of each column. 

### Query method

```go
Query(ctx context.Context, query string) (Rows, []types.Property, error)
```

Chichi invokes the `Query` method when previewing the rows returned by a query while creating or updating a database source action, and to get the data during an import. The query is provided after replacing any placeholders like `${limit}`.

The `Query` method runs the query and gives back two things: the rows themselves, which follow the `Rows` interface, and the columns as a slice of `Property` values. Here's what the `Rows` interface look like:

```go
type Rows interface {
    Close() error
    Err() error
    Next() bool
    Scan(dest ...any) error
}
```

The standard Go library's `sql.Rows` type implements this interface. So, the connector can just return a `sql.Rows` value.

### Upsert Method

```go
Upsert(ctx context.Context, table string, rows []map[string]any{}, columns []types.Property) error
```

The `Upsert` method is called by Chichi during an export operation. It either creates new rows or updates existing ones in the specified table. The `columns` parameter defines the columns of the rows being processed, including a mandatory "id" column that acts as the table's primary key. If a column's value is absent in a row, the default column value should be applied.

If the specified table or any column does not exist, the method should return an error.

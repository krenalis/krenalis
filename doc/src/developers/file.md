# File Connectors

File connectors allow to read and write specific types of files such as Excel, CSV, or Parquet files.

File connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and interfaces.

## Quick Start

In the creation of a new Go module, for your file connector, you can utilize the following template by pasting it into a Go file. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package csv implements the CSV file connector.
package csv

import (
	"context"
	"io"

	"github.com/open2b/chichi"
)

func init() {
	chichi.RegisterFile(chichi.FileInfo{
		Name:      "CSV",
		Extension: "csv",
	}, New)
}

type CSV struct {
	// Your connector fields.
}

// New returns a new CSV connector instance.
func New(conf *chichi.FileConfig) (*CSV, error) {
	// ...
}

// ContentType returns the content type of the file.
func (csv *CSV) ContentType(ctx context.Context) string {
	return "text/csv"
}

// Read reads the records from r and writes them to records.
func (csv *CSV) Read(ctx context.Context, r io.Reader, sheet string, records chichi.RecordWriter) error {
	// ...
}

// Write writes to w the records read from records.
func (csv *CSV) Write(ctx context.Context, w io.Writer, sheet string, records chichi.RecordReader) error {
	// ...
}
```

## Implementation

Let's explore how to implement a file connector, for example for the CSV file format.

First create a Go module:

```sh
$ mkdir csv
$ cd csv
$ go mod init csv
```

Then add a Go file to the new directory. For example copy the previous template file.

### About the Connector

The `FileInfo` type describes information about the file connector:

- `Name`: short name, typically the name of the file type. For example, "Excel", "CSV", "Parquet", etc.
- `Icon`: icon in SVG format representing the file type. Since it's embedded in HTML pages, it's best to be minimized.
- `Extension`: main extension of the file type that the connector reads and writes. It's used as a placeholder in the input field, where the user indicates the file name to read or write.

This information is passed to the `RegisterFile` function that, executed during package initialization, registers the file connector:

```go
func init() {
    chichi.RegisterFile(chichi.FileInfo{
        Name:      "CSV",
        Icon:      icon,
        Extension: "csv",
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterFile` function is the function utilized for creating a connector instance:

```go
func New(conf *chichi.FileConfig) (*CSV, error)
```

This function accepts a file configuration and yields a value representing your custom type. A connector can be instantiated either as a source or a destination, but not both simultaneously. Consequently, an instance of a connector will be responsible for either reading or writing a file, depending on its role.

### File Configuration

The structure of `FileConfig` is outlined as follows:

```go
type FileConfig struct {
    Role        chichi.Role
    Settings    []byte
    SetSettings chichi.SetSettingsFunc
}
```

- `Role`: Specifies the intended role of the resulting instance, which can be either `Source` or `Destination`.
- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSetting`: A function that enables the connector to update its settings as necessary.

### ContentType method

```go
ContentType(ctx context.Context) string
```

The `ContentType` method is used by Chichi to find out what type of content should be used when saving a file to a storage location. For example, the CSV connector always says it's "text/csv; charset=UTF-8". This method might always give the same answer or change depending on the settings.

### Read method

```go
Read(ctx context.Context, r io.Reader, sheet string, records chichi.RecordWriter) error
```

The `Read` method is called by Chichi to read records from a file. This happens both when previewing the file and when performing an import.

The `Read` method takes an `io.Reader` as an argument from which to read the file's contents, and a `RecordWriter` onto which to write the read records. `RecordWriter` is defined as:

```go
type RecordWriter interface {

	// Columns sets the columns of the records as properties.
	// Columns must be called before Record, RecordMap, and RecordString.
	Columns([]types.Property) error

	// Record writes a record as a slice of any.
	Record([]any) error

	// RecordMap writes a record as a string to any map.
	RecordMap(record map[string]any) error

	// RecordString writes a record as a string slice.
	RecordString([]string) error
}
```

The `Read` method must first call the `Columns` method of `RecordWriter`, passing the columns present in the file as arguments. Then, it calls one of the methods `Record`, `RecordMap`, or `RecordString`, based on which one is most convenient, for each record in the file.

If a call to any of the `RecordWriter` methods returns an error, it must halt and return the received error exactly.

Once it has finished writing all the read records, it simply returns without errors.

The `sheet` parameter is only used if the connector supports multiple sheets, meaning if the files it reads can contain multiple sheets. In this case, Chichi passes the name of the sheet to read as an argument. See the [Sheets method](#sheets-method) for more details.

### Write method

```go
Write(ctx context.Context, w io.Writer, sheet string, records chichi.RecordReader) error
```

The `Write` method is invoked by Chichi to write records to a new file. This occurs during an export process.

The `Write` method takes an `io.Writer` as an argument to write the contents of the entire file and a `RecordReader` from which to read the records to be written. `RecordWriter` is an interface defined as follows:

```go
type RecordReader interface {

	// Ack acknowledges the processing of the record with the given GID.
	Ack(gid int, err error)

	// Columns returns the columns of the records as properties.
	Columns() []types.Property

	// Record returns the next record as a slice of any with its GID.
	Record(ctx context.Context) (gid int, record []any, err error)
}
```

The `Write` method first needs to call the `Columns` method to determine the columns of the records to be written. Then it calls the `Record` method to read each individual record. It can then either immediately write the read record to the `io.Writer` or continue reading to do so later.

The `Ack` method is called to confirm the writing of a record. If a decision is made not to write a record, `Ack` still needs to be called, but this time passing a non-nil error as the second argument.

### Sheets method

If your file format supports multiple sheets, as the Excel file format, to determine which sheets are present in a file, the connector should implement the `Sheets` method:

```go
Sheets(ctx context.Context, r io.Reader) ([]string, error)
```

The `Read` and `Write` methods receive the sheet from which to read or write records as an argument. If the connector implements the `Sheets` method, the argument passed will always be a valid sheet name, otherwise it will always be an empty string.

> A valid sheet name is UTF-8 encoded, has a length in the range [1,31], does not start or end with “ ' ”, and does not contain any of *, /, :, ?,[, \\, and ]. Sheet names are case-insensitive.

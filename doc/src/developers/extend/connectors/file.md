{% extends "/layouts/doc.html" %}
{% macro Title string %}File Connectors{% end %}
{% Article %}

<span>Extend Meergo</span>
# File connectors

File connectors allow to read and write specific types of files such as Excel, CSV, or Parquet files.

File connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and methods.

Note that it is possible to implement a file connector that supports only reading or only writing of records, as it is not necessary that a file connector supports both. It is sufficient to specify the functionalities that the connector implements through the `FileInfo`, described below, then implement the required methods for those functionalities.

## Quick start

In the creation of a new Go module, for your file connector, you can utilize the following template by pasting it into a Go file. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package csv implements the CSV file connector.
package csv

import (
	"context"
	"io"

	"github.com/meergo/meergo"
)

func init() {
	meergo.RegisterFile(meergo.FileInfo{
		Name:      "CSV",
		Extension: "csv",
		AsSource: &meergo.AsSourceFile{
			HasSettings: true,
		},
		AsDestination: &meergo.AsDestinationFile{
			HasSettings: true,
		},
	}, New)
}

type CSV struct {
	// Your connector fields.
}

// New returns a new CSV connector instance.
func New(conf *meergo.FileConfig) (*CSV, error) {
	// ...
}

// ContentType returns the content type of the file.
func (csv *CSV) ContentType(ctx context.Context) string {
	return "text/csv"
}

// Read reads the records from r and writes them to records.
func (csv *CSV) Read(ctx context.Context, r io.Reader, sheet string, records meergo.RecordWriter) error {
	// ...
}

// Write writes to w the records read from records.
func (csv *CSV) Write(ctx context.Context, w io.Writer, sheet string, records meergo.RecordReader) error {
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

Later on, you can [build an executable with your connector](../../getting-started#build-with-your-custom-connectors).

### About the connector

The `FileInfo` type describes information about the file connector:

- `Name`: short name, typically the name of the file type. For example, "Excel", "CSV", "Parquet", etc.
- `AsSource`: information about the file connector when it used as source. This should be set only when the file connector can be used as a source, otherwise should be nil.
  - `HasSettings`: indicates whether the connection has format settings when used as source.
- `AsDestination`: information about the file connector when it used as destination. This should be set only when the file connector can be used as a destination, otherwise should be nil.
  - `HasSettings`: indicates whether the connection has format settings when used as destination.
- `HasSheets`
- `TimeLayouts`: layouts for the `datetime`, `date`, and `time` values when they are represented as strings. See [Time Layouts](data-values#time-layouts) in [Data Values](data-values) for more details.
- `Extension`: main extension of the file type that the connector reads and/or writes. It's used as a placeholder in the input field, where the user indicates the file name to read or write.
- `Icon`: icon in SVG format representing the file type. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterFile` function that, executed during package initialization, registers the file connector:

```go
func init() {
    meergo.RegisterFile(meergo.FileInfo{
		Name:      "CSV",
		Icon:      icon,
		Extension: "csv",
		AsSource: &meergo.AsSourceFile{
			HasSettings: true,
		},
		AsDestination: &meergo.AsDestinationFile{
			HasSettings: true,
		},
	}, New)
}
```

### Constructor

The second argument supplied to the `RegisterFile` function is the function utilized for creating a connector instance:

```go
func New(conf *meergo.FileConfig) (*CSV, error)
```

This function accepts a file configuration and yields a value representing your custom type.

The structure of `FileConfig` is outlined as follows:

```go
type FileConfig struct {
    Settings    []byte
    SetSettings meergo.SetSettingsFunc
}
```

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSetting`: A function that enables the connector to update its settings as necessary.

### ContentType method

```go
ContentType(ctx context.Context) string
```

The `ContentType` method is used by Meergo to find out what type of content should be used when saving a file to a storage location. For example, the CSV connector always says it's "text/csv; charset=UTF-8". This method might always give the same answer or change depending on the settings.

### Read method

```go
Read(ctx context.Context, r io.Reader, sheet string, records meergo.RecordWriter) error
```

The `Read` method is called by Meergo to read records from a file. This happens both when previewing the file and when performing an import.

The `Read` method takes an `io.Reader` as an argument from which to read the file's contents, and a `RecordWriter` onto which to write the read records. `RecordWriter` is defined as:

```go
// A RecordWriter interface is used by file connectors to write read records.
type RecordWriter interface {

	// Columns sets the columns of the records as properties.
	// Columns must be called before Record, RecordSlice, and RecordStrings.
	Columns(columns []types.Property) error

	// Record writes a record represented as a string to any map.
	// The record's length must equal to the number of columns.
	Record(record map[string]any) error

	// RecordSlice writes a record represented as a slice of any.
	// The record's length must equal to the number of columns.
	RecordSlice(record []any) error

	// RecordStrings writes a record represented as a string slice.
	// The record's length must be less than or equal to the number of columns, and
	// record cannot be nil.
	//
	// RecordStrings may modify the elements of the record.
	RecordStrings(record []string) error
}
```

The `Read` method must first call the `Columns` method of `RecordWriter`, passing the columns present in the file as arguments. Then, it calls one of the methods `Record`, `RecordSlice`, or `RecordStrings`, based on which one is most convenient, for each record in the file.

For `Record` and `RecordString`, values must align with the columns' positions returned by `Columns`. Alternatively, using `RecordMap`, if a user may lack a value for a property, ensure the property is marked as `Required: false` in the schema; it may then be omitted from the map passed to `RecordMap`. This behavior is distinct from a property having a value of `nil`.

If a call to any of the `RecordWriter` methods returns an error, it must halt and return the received error exactly.

Once it has finished writing all the read records, it simply returns without errors.

The `sheet` parameter is only used if the connector supports multiple sheets, meaning if the files it reads can contain multiple sheets. In this case, Meergo passes the name of the sheet to read as an argument. See the [Sheets method](#sheets-method) for more details.

> The columns passed to the `Columns` method must have valid property names. To assist with this, you can use the `meergo.SuggestPropertyName` function, which suggests column names that are valid.
> For example, if a column in the file is named `"Prénom"`, calling `SuggestPropertyName("Prénom")` will suggest `"Prenom"`, which is a valid property name that can be used with the `Columns` method.

If a column has an unsupported type, return an `*UnsupportedColumnTypeError` error. Use the `NewUnsupportedColumnTypeError` function from the `meergo` package to create this error.

### Write method

```go
Write(ctx context.Context, w io.Writer, sheet string, records meergo.RecordReader) error
```

The `Write` method is invoked by Meergo to write records to a new file. This occurs during an export process.

The `Write` method takes an `io.Writer` as an argument to write the contents of the entire file and a `RecordReader` from which to read the records to be written. `RecordReader` is an interface defined as follows:

```go
type RecordReader interface {

	// Ack acknowledges the processing of the record with the given GID.
	// err is the error occurred processing the record, if any.
	Ack(gid uuid.UUID, err error)

	// Columns returns the columns of the records as properties.
	Columns() []types.Property

	// Record returns the next record with its ack ID. The keys of record represent
	// column names. A record may be empty or contain only a subset of columns.
	// It returns "", nil, and io.EOF if there are no more records.
	//
	// After a record has been read and processed, the caller should call Ack
	// to acknowledge the processing of the record.
	Record(ctx context.Context) (ackID string, record map[string]any, err error)}
```

The `Write` method first needs to call the `Columns` method to determine the columns of the records to be written. Then it calls the `Record` method to read each individual record. It can then either immediately write the read record to the `io.Writer` or continue reading to do so later.

The `Ack` method is called to confirm the writing of a record. If a decision is made not to write a record, `Ack` still needs to be called, but this time passing a non-nil error as the second argument.

### Sheets method

If your file format supports multiple sheets, as the Excel file format, to determine which sheets are present in a file, the connector should indicate that in the `FileInfo` with `HasSheets: true` and implement the `Sheets` method:

```go
Sheets(ctx context.Context, r io.Reader) ([]string, error)
```

The `Read` and `Write` methods receive the sheet from which to read or write records as an argument. If the connector has sheets, the argument passed will always be a valid sheet name, otherwise it will always be an empty string.

> A valid sheet name is UTF-8 encoded, has a length in the range [1,31], does not start or end with “ ' ”, and does not contain any of *, /, :, ?,[, \\, and ]. Sheet names are case-insensitive.

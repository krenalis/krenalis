# File Storage Connectors

File storage connectors allow to read and write file content on a file storage such as SFTP, HTTP and S3.

File storage connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and interfaces.

## Quick Start

In the creation of a new Go module, for your file storage connector, you can utilize the following template by pasting it into a Go file. Customize the template with your desired package name, type name, and pertinent connector information:


```go
// Package s3 implements the S3 file storage connector.
package s3

import (
	_ "embed"

	"github.com/open2b/chichi"
)

// Connector icon.
var icon = "<svg></svg>"

func init() {
	chichi.RegisterFileStorage(chichi.FileStorageInfo{
		Name: "S3",
		Icon: icon,
	}, New)
}

type S3 struct {
	// Your connector fields.
}

// New returns a new S3 connector instance.
func New(conf *chichi.FileStorageConfig) (*S3, error) {
	// ...
}

// CompletePath returns the complete representation of the given path name.
func (s3 *S3) CompletePath(ctx context.Context, name string) (string, error) {
	// ...
}

// Reader opens the file at the given path name and returns a ReadCloser from
// which to read the file and its last update time.
func (s3 *S3) Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error) {
	// ...
}

// Write writes the data read from r into the file with the given path name.
func (s3 *S3) Write(ctx context.Context, r io.Reader, name, contentType string) error {
	// ...
}
```

## Implementation

Let's explore how to implement a file storage connector, for example for the S3 file format.

First create a Go module:

```sh
$ mkdir s3
$ cd s3
$ go mod init s3
```

Then add a Go file to the new directory. For example copy the previous template file.

As you can see, the template file embeds a SVG file, this is the icon that represent the connector. Choose an appropriate SVG icon and put it into the module directory.

### About the Connector

The `FileInfo` type describes information about the file storage connector:


- `Name`: short name, typically the name of the storage. For example, "S3", "HTTP", "SFTP", etc.

- `SourceDescription`: brief description of the connector when the connector is used as a source. It should complete the sentence "Add an action to ...".

- `DestinationDescription`: brief description of the connector when the connector is used as a destination. It should complete the sentence "Add an action to ...".

- `Icon`: icon in SVG format representing the file type. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterFileStorage` function that, executed during package initialization, registers the file storage connector:

```go
func init() {
    chichi.RegisterFileStorage(chichi.FileStorageInfo{
        Name: "S3",
        Icon: icon,
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterFileStorage` function is the function utilized for creating a connector instance:

```go
func New(conf *chichi.FileStorageConfig) (*S3, error)
```

This function accepts a file storage configuration and yields a value representing your custom type. A connector can be instantiated either as a source or a destination, but not both simultaneously. Consequently, an instance of a connector will be responsible for either reading or writing files, depending on its role.

### File Storage Configuration

The structure of `FileStorageConfig` is outlined as follows:

```go
type FileStorageConfig struct {
    Role        chichi.Role
    Settings    []byte
    SetSettings chichi.SetSettingsFunc
}
```

- `Role`: This field specifies the anticipated role of the resulting instance, which can either be `Source` or `Destination`.

- `Settings`: It contains the instance settings in JSON format. Later, we'll delve into how the connector defines its settings.

- `SetSetting`: Is a function that allows the connector to update its settings as needed.

### CompletePath method

```go
CompletePath(ctx context.Context, name string) (string, error)
```

The `CompletePath` method is invoked by Chichi to present the user with the full path of a file in the given storage, based on the path specified by the user.

The `name` parameter is always a UTF-8 encoded string with a length in runes ranging from 1 to 1024. The `CompletePath` method is responsible for validating the path. If the path is invalid, it should return an `InvalidPathError` error; otherwise, it should return a UTF-8 encoded string representing the complete path. The returned path is intended solely for display to the user.

### Reader method

```go
Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
```

The `Reader` method is used by Chichi to read files, such as during previews or exports.

The `name` parameter represents the file path entered by the user, which has been validated with the `CompletePath` method. It also gives the last time the file was changed, if known; otherwise, it returns zero time.

### Write method

```go
Write(ctx context.Context, r io.Reader, name, contentType string) error
```

The `Write` method is used by Chichi to save a file to the storage when exporting.

`Write` reads the content from the provided `Reader`. The `name` parameter represents the file path specified by the user, which has been validated with the `CompletePath` method. The `contentType` parameter indicates the type of content in the file, obtained from the `ContentType` method of the file connector.

The connector should ensure, as much as possible, that the write operation is both atomic and durable. If `Write` returns an error, no file should be created. Conversely, if it returns a nil error, the file has been successfully written. 

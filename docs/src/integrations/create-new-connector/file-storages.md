{% extends "/layouts/doc.html" %}
{% macro Title string %}File storage Connectors{% end %}
{% Article %}

# File storages

File storage connectors allow to read and write file content on a file storage such as SFTP, HTTP and S3.

File storage connectors, like other types of connectors, are written in Go. A connector is a Go module that implements specific functions and methods.

Note that it is possible to implement a file storage connector that supports only reading or only writing operations, as it is not necessary that a file storage connector supports both. It is sufficient to specify the functionalities that the connector implements through the `FileStorageInfo`, described below, then implement the required methods for those functionalities.

## Quick start

In the creation of a new Go module, for your file storage connector, you can utilize the following template by pasting it into a Go file. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package s3 provides a connector for S3.
package s3

import (
    "context"
    "io"
    "time"

    "github.com/meergo/meergo"
)

func init() {
	meergo.RegisterFileStorage(meergo.FileStorageInfo{
		Code:          "s3",
		Label:         "S3",
		Categories:    meergo.CategoryFileStorage,
		AsSource:      &meergo.AsFileStorageSource{},
		AsDestination: &meergo.AsFileStorageDestination{},
		Icon:          icon,
	}, New)
}

type S3 struct {
    // Your connector fields.
}

// New returns a new connector instance for S3.
func New(env *meergo.FileStorageEnv) (*S3, error) {
    // ...
}

// AbsolutePath returns the absolute representation of the given path name.
func (s3 *S3) AbsolutePath(ctx context.Context, name string) (string, error) {
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

Later on, you can [build an executable with your connector](/installation/from-source#building-using-the-go-tools).

### About the connector

The `FileStorageInfo` type describes information about the file storage connector:

- `Code`: unique identifier in kebab-case (`a-z0-9-`), e.g. "s3", "http", "sftp".
- `Label`: display label in the Admin console, typically the storage's name (e.g. "S3", "HTTP", "SFTP").
- `Categories`: the categories that the connector falls into. There must be at least one category.
- `Icon`: icon in SVG format representing the file storage. Since it's embedded in HTML pages, it's best to be minimized.

This information is passed to the `RegisterFileStorage` function that, executed during package initialization, registers the file storage connector:

```go
func init() {
	meergo.RegisterFileStorage(meergo.FileStorageInfo{
		Code:          "s3",
		Label:         "S3",
		AsSource:      true,
		AsDestination: true,
		Icon:          icon,
	}, New)
}
```

### Constructor

The second argument supplied to the `RegisterFileStorage` function is the function utilized for creating a connector instance:

```go
func New(env *meergo.FileStorageEnv) (*S3, error)
```

This function accepts a file storage environment and yields a value representing your custom type.

The structure of `FileStorageEnv` is outlined as follows:

```go
// FileStorageEnv is the environment for a file storage connector.
type FileStorageEnv struct {

    // Settings is the raw settings data.
    Settings []byte

    // SetSettings is the function used to update the settings.
    SetSettings SetSettingsFunc
}
```

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSetting`: A function that enables the connector to update its settings as necessary.

### AbsolutePath method

```go
AbsolutePath(ctx context.Context, name string) (string, error)
```

The `AbsolutePath` method is invoked by Meergo to present the user with the absolute path of a file in the given storage, based on the path specified by the user.

The `name` parameter is always a UTF-8 encoded string with a length in runes ranging from 1 to 1024. It is the responsibility of the `AbsolutePath` method to validate the path based on its specific rules. If the path is invalid, it should return an `InvalidPathError` error; otherwise, it should return a UTF-8 encoded string representing the absolute path. The returned path is intended solely for display to the user.

If the connector accepts paths with a slash (“/”) separator, the method should accommodate both paths with and without a leading slash.

### Reader method

```go
Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
```

The `Reader` method is used by Meergo to read files, such as during previews or exports.

The `name` parameter represents the file path entered by the user, which has been validated with the `AbsolutePath` method. It also gives the last time the file was changed, if known; otherwise, it returns zero time.

### Write method

```go
Write(ctx context.Context, r io.Reader, name, contentType string) error
```

The `Write` method is used by Meergo to save a file to the storage when exporting.

`Write` reads the content from the provided `Reader`. The `name` parameter represents the file path specified by the user, which has been validated with the `AbsolutePath` method. The `contentType` parameter indicates the type of content in the file, obtained from the `ContentType` method of the file connector.

The connector should ensure, as much as possible, that the write operation is both atomic and durable. If `Write` returns an error, no file should be created. Conversely, if it returns a `nil` error, the file has been successfully written.

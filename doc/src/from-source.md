{% extends "/layouts/doc.html" %}
{% macro Title string %}From source{% end %}
{% Article %}

# From source

There are two ways to install Meergo from the source:

* [Build using Go modules](#build-using-go-modules): Let Go download and compile the main module.

* [Build from repository](#build-from-repository): Clone the repository locally and compile the source.

### Build using Go modules

For this installation method you need to have [Go](https://go.dev/doc/install) 1.23 and [PostgreSQL](https://www.postgresql.org/download/) 13 or higher.

1. **Create a new directory**

   ```sh
   mkdir meergo
   cd meergo 
   ```

2. **Copy the main.go file**

   Obtain the `main.go` file from [Meergo's GitHub repository](https://github.com/meergo/meergo/blob/main/cmd/meergo/main.go) and place it in the directory you just created.

3. **Initialize a Go module**

   ```sh
   go mod init meergo
   go mod tidy
   ```

4. **Generate the admin assets and build the executable**

   Use the following commands to generate the admin assets and to build the executable:

   ```sh
   go generate
   go build
   ```

   > Note: You can provide the `-trimpath` option to the `go build` command to remove absolute paths from any error stack traces in Meergo. This way, if telemetry is enabled, the absolute paths will not be sent.

   Verify that the executable `meergo` (or `meergo.exe` on Windows) has been created in the current directory.

Proceed with the [configuration](#configuration) after completing these steps.

### Build from repository

For this installation method you need to have [Git](https://git-scm.com/downloads), [Go](https://go.dev/doc/install) 1.23, and [PostgreSQL](https://www.postgresql.org/download/) 13 or higher.

1. **Clone the repository and change into the _meergo/cmd/meergo_ directory**

   > Since the repository of Meergo is private, you may need to configure your local `GOPRIVATE` variable in order to test and develop some of the installation methods listed below.
   > It may be enough to add `github.com/meergo/meergo` to the paths listed in the `GOPRIVATE` values (which are separated by a comma `,`).

   ```sh
   git clone https://github.com/meergo/meergo
   cd meergo/cmd/meergo
   ```

2. **Generate the admin assets and build the executable**

   Use the following commands to generate the admin assets and to build the executable:

   ```sh
   go generate
   go build
   ```

   > Note: You can provide the `-trimpath` option to the `go build` command to remove absolute paths from any error stack traces in Meergo. This way, if telemetry is enabled, the absolute paths will not be sent.

   Verify that the executable `meergo` (or `meergo.exe` on Windows) has been created in the current directory.

Then you can proceed with the [configuration](#configuration).

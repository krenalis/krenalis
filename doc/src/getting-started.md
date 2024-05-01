# Getting started

In this section you will find indications and instructions for getting started with the use of Chichi.

## Requirements

First of all, you will need:

- PostgreSQL 15 or higher
- Linux, Unix, OSX or Windows
- A data warehouse (currently only PostgreSQL)

Various installation methods may then require additional requirements depending on the level of customization required, specified in the sections below.

> Since the repository of Chichi is private, you may need to configure your local `GOPRIVATE` variable in order to test and develop some of the installation methods listed below.
> It may be enough to add `github.com/open2b/chichi` to the paths listed in the `GOPRIVATE` values (which are separated by a comma `,`).

## Installation

> 🚧 Some installation methods are currently under development and may have some problems that still needs to be resolved. For now, a working method is [Locally cloning the repository](#locally-cloning-the-repository).

There are several ways you can install Chichi. Choose the method that you deem most suitable for your needs and skills:

- [Download a pre-built release](#download-a-pre-built-release). Recommended for those who want to start using Chichi immediately without needing of compiling or customize the executable.
- [Build from Git source](#build-from-git-source). Recommended for those familiar with Go tools and want some degree of control over their builds.
- [Build with your custom connectors](#build-with-your-custom-connectors). Recommended for those with advanced customization needs who want to configure the available connectors in Chichi.

### Download a pre-built release

> 🚧 Releases are not available yet, so this section is just a stub.

You can download a build of Chichi from the [releases page of the repository](https://github.com/open2b/chichi/releases) or from the [Chichi's website](https://example.com).

Then you can proceed with the [configuration](#configuration).

### Build from Git source

Besides the requirements listed at the beginning of this page, for this installation method you also need to have:

* [Git](https://git-scm.com/)
* [Go v1.22](https://go.dev/doc/install)

#### Steps

1. **Clone the repository**

    ```sh
   git clone https://github.com/open2b/chichi
   ```

2. **Change into the chichi/cmd/chichi directory**

    ```sh
   cd chichi/cmd/chichi
   ```

3. **Generate the assets**

   Use the following command to bundle and compress the assets, which will be embedded into the executable:

   ```sh
   go generate
   ```

   It must be re-executed if you pull a new version of Chichi.

4. **Build the executable**

    ```sh
    go build
    ```

   Verify that the executable `chichi` (or `chichi.exe` on Windows) has been created in the current directory.

Then you can proceed with the [configuration](#configuration).

### Build with Your Custom Connectors

Besides the requirements listed at the beginning of this page, for this installation method you will also need:

   * [Go v1.22](https://go.dev/doc/install)

#### Steps

1. **Create a new directory**

   ```sh
   mkdir chichi
   cd chichi 
   ```

2. **Copy the main.go file**

   Obtain the `main.go` file from [Chichi's GitHub repository](https://github.com/open2b/chichi/blob/main/cmd/chichi/main.go) and place it in the directory you just created.

3. **Import your connectors (Optional)**

   Modify the `main.go` file to include imports for each connector you wish to integrate into Chichi:

   ```go
   import (
      _ "github.com/example/connector"
   )
   ```

   Optionally, replace the default imports from [`github.com/open2b/chichi/connectors`](https://github.com/open2b/chichi/tree/main/connectors/connectors.go) with specific imports of the connectors you need.

4. **Initialize a Go module**

   ```sh
   go mod init chichi
   go mod tidy
   ```

5. **Generate the assets**

   Use the following command to bundle and compress the assets, which will be embedded into the executable:

   ```sh
   go generate
   ```

   Note: Re-execute `go generate` if you change Chichi module version.

6. **Build the executable**

   ```sh
   go build
   ```

   Check that the `chichi` executable (or `chichi.exe` on Windows) is created in the current directory.

Proceed with the [configuration](#configuration) after completing these steps.

### Build with your custom connectors

Besides the requirements listed at the beginning of this page, for this installation method you also need to have:

* [Go v1.22](https://go.dev/doc/install)

#### Steps

1. Create a new directory

   ```sh
   mkdir chichi
   cd chichi 
   ```

3. Copy the Chichi's `main.go` file
   
   > [github.com/open2b/chichi/cmd/chichi/main.go](https://github.com/open2b/chichi/blob/main/cmd/chichi/main.go)
   
4. Import your connectors (optional)

   Edit the `main.go` file adding an import for each connector that you want to build in Chichi:

   ```go
   import (
      _ "github.com/example/connector"
   )
   ```
   
   You can also replace the import of the package [`github.com/open2b/chichi/connectors`](https://github.com/open2b/chichi/tree/main/connectors/connectors.go) with its single imports of the connectors that you want to build into Chichi.

5. Initialize a Go module

   ```sh
   go mod init chichi
   go mod tidy
   ```

6. Get, bundle, and compress the assets

   The following commands get, bundle, and compress the assets and save them in a directory called `chichi-assets`. They will be automatically embedded into the executable.

   ```sh
   go get github.com/open2b/chichi/assets
   go generate
   ```

   `go generate` must be re-executed if you change the Chichi version used by this module.

7. Build the executable

    ```sh
   go build
   ```

   Verify that the executable `chichi` (or `chichi.exe` on Windows) has been created in the current directory.

Then you can proceed with the [configuration](#configuration).

## Configuration

Now that you have obtained the executable file of `chichi`, it is necessary to proceed with the configuration.

1. Choose a directory of the filesystem: it will be the directory in which you will start Chichi.
2. Take [the example configuration file `config.example.yaml`](https://github.com/open2b/chichi/blob/main/cmd/chichi/config.example.yaml) and copy it into the chosen directory in a file named `config.yaml`
3. Modify `config.yaml` according to your needs.

If the `https` configuration parameter is set to `true` in the configuration file, then proceed with the creation of the certificates; otherwise, proceed with the setup of the database.

## Certificates

In the directory you have chosen to start Chichi, add the certificate files `cert.pem` (for the domain) and `key.pem` (for its key).

Now proceed with the setup of the database.

## Setup the database

Set up the database you specified in the configuration file by executing the [`database/PostgreSQL.sql` script](https://github.com/open2b/chichi/blob/main/database/PostgreSQL.sql). This script is designed to configure the PostgreSQL database.

## Start Chichi

Run the `chichi` executable within the directory of your choice, containing the configuration file and the certificates.

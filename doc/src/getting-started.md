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

- [Downloading a pre-built release of Chichi](#downloading-a-pre-built-release-of-chichi). Recommended for those who want to start using Chichi immediately without needing of compiling or customize it.
- [Installing with `go install`](#installing-with-go-install). Recommended for those familiar with Go tools and want some degree of control over their builds.
- [Custom installation](#custom-installation). Recommended for those with advanced customization needs who want to configure the available connectors in Chichi.
- [Locally cloning the repository](#locally-cloning-the-repository). Recommended for those who want total control over their build, operating on the Chichi's source code in a local repository. This is also intended when *developing* Chichi.

### Downloading a pre-built release of Chichi

> 🚧 Releases are not available yet, so this section is just a stub.

You can download a build of Chichi from the [releases page of the repository](https://github.com/open2b/chichi/releases) or from the [Chichi's website](https://example.com).

Then you can proceed with the [configuration](#configuration).

### Installing with `go install`

> 🚧 This does not work due to the need to run `npm install` in the UI directory.

> 🚧 This does not work due to a problem with `replace` directives in Chichi's modules.

Execute the command:

```
go install github.com/open2b/chichi/cmd/chichi@latest
```

You should now have the `chichi` executable available in your system.

Then you can proceed with the [configuration](#configuration).

### Custom installation

> 🚧 This does not work due to the need to run `npm install` in the UI directory.

> 🚧 This does not work due to a problem with `replace` directives in Chichi's modules.

Besides the requirements listed at the beginning of this page, for this installation method you also need to have:

* Git
* Go v1.22
* Node.js 20 or higher
* npm

Steps:

1. Copy-paste the file [`cmd/chichi/main.go`](https://github.com/open2b/chichi/blob/main/cmd/chichi/main.go) into a file called `main.go` in an empty directory of your choice.

2. Initialize a Go module executing:

    ```console
    go mod init chichi
    ```

3. Add the Chichi module to your module executing:

    ```console
    go get github.com/open2b/chichi@latest
    ```

4. Just leave the import of the connectors you want to include in the build

5. Execute:

    ```
    go mod tidy
    ```
6. Install or simply build the `chichi` executable with:

   ```
   go build
   ```

   or

   ```
   go install
   ```

Then you can proceed with the [configuration](#configuration).

### Locally cloning the repository

Besides the requirements listed at the beginning of this page, for this installation method you also need to have:

* Git
* Go v1.22
* Node.js 20 or higher
* npm

Steps:

1. Clone the repository

    ```sh
   git clone https://github.com/open2b/chichi
   ```

2. Build the executable

    ```sh
    cd ui
    npm install 
    cd ../cmd/chichi
    go build -tags osusergo,netgo -trimpath
    ```

Then you can proceed with the [configuration](#configuration).

## Configuration

Now that you have obtained the executable file of `chichi`, it is necessary to proceed with the configuration.

1. Choose a directory of the filesystem: it will be the directory in which you will start Chichi.
2. Take [the example configuration file `config.example.yaml`](https://github.com/open2b/chichi/blob/main/config.example.yaml) and copy it into the chosen directory in a file named `config.yaml`
3. Modify `config.yaml` according to your needs.

Proceed now with the creation of the certificates.

## Certificates

In the directory you have chosen to start Chichi, add the certificate files `cert.pem` (for the domain) and `key.pem` (for its key).

Now proceed with the setup of the database.

## Setup the database

Set up the database you specified in the configuration file by executing the [`database/PostgreSQL.sql` script](https://github.com/open2b/chichi/blob/main/database/PostgreSQL.sql). This script is designed to configure the PostgreSQL database.

## Start Chichi

Run the `chichi` executable within the directory of your choice, containing the configuration file and the certificates.

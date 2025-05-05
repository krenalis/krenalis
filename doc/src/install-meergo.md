{% extends "/layouts/doc.html" %}
{% macro Title string %}Install Meergo{% end %}
{% Article %}

# Install Meergo

There are several ways to get started with Meergo:

* [Docker](#docker). This method is ideal for local development, testing, and prototyping. 
* [Pre-packaged binaries](#pre-packaged-binaries). A convenient method for quickly setting up Meergo without the need to compile from source.
* [Source code](#source-code). Recommended if you wish to customize the executable or contribute to the project by building Meergo directly from the source.

## Docker

> 🚧 It will be simplified in the future, as a version of the Meergo image will already need to be available on platforms like Docker Hub.

For this installation method you need to have [Git](https://git-scm.com/downloads) and [Docker](https://docs.docker.com/engine/install/).

1. Clone the Meergo's repository and enter into the repository's directory: 

   ```sh
   git clone https://github.com/meergo/meergo
   cd meergo
   ```

2. Build the Docker image of Meergo and launch it:

   ```sh
   docker build -t meergo:dev . --progress=plain
   docker compose up
   ```

> If you have previously started Meergo using Docker Compose and want to reset it, perhaps for a clean installation or because you are running a new version of Meergo, you just need to clear the Meergo Docker data by running `docker compose down -v` before starting Meergo with `docker compose up`.

3. Open the admin at

   [http://localhost:9090/admin/](http://localhost:9090/admin/)

Initially, login is not required with the Docker installation. To enable login, create a new member with their email and password. 

## Pre-packaged binaries

For this installation method you need to have [PostgreSQL](https://www.postgresql.org/download/) 13 or higher.

> 🚧 Releases are not available yet, so this section is just a stub.

You can download a build of Meergo from the [releases page of the repository](https://github.com/meergo/meergo/releases) or from the [Meergo's website](https://www.meergo.com).

Then you can proceed with the [configuration](#configuration).

## Source code

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

   Verify that the executable `meergo` (or `meergo.exe` on Windows) has been created in the current directory.

Then you can proceed with the [configuration](#configuration).

## Configuration

Once you have obtained the `meergo` executable, follow these steps to configure the application:

1. **Choose a directory** on your filesystem — this will be the working directory where you will run Meergo.
2. Download the example configuration file [`meergo.example.env`](https://github.com/meergo/meergo/blob/experiment-with-env/cmd/meergo/meergo.example.env) and copy it into the chosen directory as `.env`.
3. Edit `.env` to match your environment and requirements.

> The `.env` file contains the definition of environment variables and is sourced by Meergo at startup. It is therefore possible, alternatively and according to the needs of the environment in which Meergo is to be run, to define the environment variables before starting Meergo, without using the `.env` file.

Next, you’ll need to set up certificates (if using HTTPS), configure the database, and launch the application.

### Certificates

If you have enabled HTTPS by setting the `MEERGO_HTTP_TLS_ENABLED` environment variable to `true`, you must also specify the TLS certificate and private key files. To do this, set the following environment variables:

- `MEERGO_HTTP_TLS_CERT_FILE`: Path to the TLS certificate file.
- `MEERGO_HTTP_TLS_KEY_FILE`: Path to the corresponding private key file.

Make sure both files are accessible.

### Database setup

Meergo relies on PostgreSQL for its internal database. Note that this is not the same as the data warehouse you will configure later — this database is used exclusively for Meergo's own operational data and internal management.

To initialize it, execute the SQL script [`database/PostgreSQL.sql`](https://github.com/meergo/meergo/blob/main/database/PostgreSQL.sql), which will create the required schema and tables based on your configuration.

Make sure the database connection settings in specified with the environment variables match your PostgreSQL instance.

### Starting Meergo

Once the setup is complete, run the `meergo` executable (if you have a `.env` file, it must be in the same directory where Meergo is executed).

Meergo will launch using the configuration specified by the environment variables and will be ready for use.

## First login

When you start **Meergo** for the first time, you can access the admin using the default credentials:

- **Email:** `acme@open2b.com`
- **Password:** `foopass2`

After logging in, you’ll be prompted to create your first **workspace**.

Each workspace operates as an isolated environment with its own **data warehouse**, which stores user data, events, and is used for identity resolution and data unification.

> ⚠️ Once a data warehouse is linked to a workspace, it **cannot be changed** later.

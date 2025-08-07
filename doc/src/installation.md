{% extends "/layouts/doc.html" %}
{% macro Title string %}Installation{% end %}
{% Article %}

# Installation

There are several ways to install Meergo or simply try out its features. Choose the one that suits you best:

* [**Using Docker**](./using-docker). This method is ideal for local development, testing, and prototyping. 
* [**Pre-compiled binaries**](./pre-compiled-binaries). A convenient method for quickly setting up Meergo without the need to compile from source.
* [**From source**](./from-source). Recommended if you wish to customize the executable or contribute to the project by building Meergo directly from the source.

### Certificates

If you have enabled HTTPS by setting the `MEERGO_HTTP_TLS_ENABLED` environment variable to `true`, you must also specify the TLS certificate and private key files. To do this, set the following environment variables:

- `MEERGO_HTTP_TLS_CERT_FILE`: Path to the TLS certificate file.
- `MEERGO_HTTP_TLS_KEY_FILE`: Path to the corresponding private key file.

Make sure both files are accessible.

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

### Import and export local files with Docker

When running Meergo under Docker, for importing and exporting files locally, you can add a Filesystem connection whose Root Path is:

```plain
/bin/meergo-files/sample-filesystem
```

which is mapped to the directory:

```plain
<local Meergo repository>/docker-compose/sample-filesystem
```

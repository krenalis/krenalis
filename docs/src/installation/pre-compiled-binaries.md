{% extends "/layouts/doc.html" %}
{% macro Title string %}Install from pre-compiled binaries{% end %}
{% Article %}

# Install from pre-compiled binaries

Meergo is also available as pre-compiled binaries.

This is the recommended installation method for users who want more control over the execution and environment of Meergo compared to running it via Docker, but don't specifically need to compile Meergo from source.

## Before you begin

To install and use Meergo via pre-compiled binaries, you need:

* [PostgreSQL](https://www.postgresql.org/download/) version 13 or later.

## How to run Meergo from pre-compiled binaries

You can find Meergo binaries on the [GitHub Releases page](https://github.com/meergo/meergo/releases).

Once downloaded and extracted, verify the `meergo` binary works by running:

```sh
$ ./meergo --help
```

You should see an output like:

```sh
Usage of ./meergo:
  -help
    	print the help for meergo and exit
```

This confirms the Meergo binary is valid and executable.

You can now proceed with the [database setup](./database-setup).

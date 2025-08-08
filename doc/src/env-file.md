{% extends "/layouts/doc.html" %}
{% macro Title string %}.env file{% end %}
{% Article %}

# .env file

> 💡 This section applies if you're running Meergo after [downloading the binary from GitHub](pre-compiled-binaries), or if you [compiled Meergo from source](from-source).
> 
> If you're [running Meergo via Docker](using-docker), this section probably doesn't apply to you and you are just interested to the [environment variables reference documentation](configuration-reference).

Meergo installation configuration is done via environment variables. You can therefore:

* Pass environment variables directly to the `meergo` command when it is run.
* For convenience, use an `env` file, documented on this page.

## The .env file

Create a file called `.env` in the same directory where you run the Meergo executable. Meergo will read the `.env` file for configuration when it starts.

For an example `.env` file, see: https://github.com/meergo/meergo/blob/main/cmd/meergo/meergo.example.env

Once you understand the syntax of the `.env` file (shown [below](#env-file-syntax)), you can look at the [documentation for each configuration key](./configuration-reference) in detail and customize your `.env` file.

## .env file syntax

The `.env` file is a simple list of key-value pairs used to configure environment variables. Here's how it works:

* One variable per line: `KEY=VALUE`.
* Strings with spaces or special characters should be quoted with double quotes.
* Lines starting with `#` are comments and ignored.
* Empty lines are ignored.
* To insert a value that contains double quotes, enclose the entire string in single quotes, e.g.: `MY_VAR='"my-quoted-value"'`.
* Use `true` or `false` for booleans. Don't use `1` or `0`.
* File paths can be absolute or relative; relative paths are based on the working directory of the Meergo process.

An example of `.env` file syntax:

```ini
# Server settings
MEERGO_HTTP_HOST=127.0.0.1
MEERGO_HTTP_PORT=9090

# TLS settings
MEERGO_HTTP_TLS_ENABLED=true
MEERGO_HTTP_TLS_CERT_FILE="/etc/ssl/certs/my meergo cert.crt"
MEERGO_HTTP_TLS_KEY_FILE='/etc/ssl/private/meergo.key'

# Commented out value
#MEERGO_HTTP_PORT=8080
```

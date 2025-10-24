{% extends "/layouts/doc.html" %}
{% macro Title string %}The .env file{% end %}
{% Article %}

# The _.env_ file

In addition to setting environment variables before starting Meergo, you can place them in a _.env_ file located in the same directory where you run the executable. Meergo will read variables from this file, but it will never override values already defined in the environment. Environment variables always take precedence over those in the _.env_ file.

A ready-to-use _.env_ file is provided in the repository: [meergo.example.env](https://github.com/meergo/meergo/blob/main/cmd/meergo/meergo.example.env). You can rename this file to _.env_ and customize it as needed. It includes all [environment variables](environment-variables) supported by Meergo.

## Example

This is an example of _.env_ file:

```ini
# Server settings
MEERGO_HTTP_HOST=127.0.0.1
MEERGO_HTTP_PORT=2022

# TLS settings
MEERGO_HTTP_TLS_ENABLED=true
MEERGO_HTTP_TLS_CERT_FILE="/etc/ssl/certs/my meergo cert.crt"
MEERGO_HTTP_TLS_KEY_FILE='/etc/ssl/private/meergo.key' # key file

# Commented out value
#MEERGO_HTTP_PORT=8080

# With export
export MEERGO_HTTP_READ_TIMEOUT=10s
```

## Syntax

The _.env_ file is a simple list of key-value pairs used to configure environment variables. Here's how it works:

* One variable per line: `KEY=VALUE`.
* Lines starting with `#` or empty lines are ignored.
* Lines can start with `export `.
* Keys and values cannot contain `NUL` characters.
* Values can be:
    * **Unquoted** — spaces at the end are kept; inline comments start with ` #`.
    * **Double-quoted** — supports `\n`, `\r`, `\t`, `\\`, `\"`.
    * **Single-quoted** — supports only `\'`.
* Values after a closing quote must have only spaces or a comment.
* Variables in the file override existing environment values.
* Use `true` or `false` for booleans. Don't use `1` or `0`.
* File paths can be absolute or relative; relative paths are based on the working directory of the Meergo process.

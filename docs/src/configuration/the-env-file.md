{% extends "/layouts/doc.html" %}
{% macro Title string %}The .env file{% end %}
{% Article %}

# The _.env_ file

In addition to setting environment variables before starting Meergo, you can place them in a _.env_ file located in the same directory where you run the executable. Meergo will read variables from this file, but it will never override values already defined in the environment. Environment variables always take precedence over those in the _.env_ file.

A ready-to-use _.env_ file is provided in the repository: [meergo.example.env](https://github.com/meergo/meergo/blob/main/cmd/meergo/meergo.example.env). You can rename this file to _.env_ and customize it as needed. It includes all [startup settings](startup-settings) supported by Meergo.

## Syntax

The _.env_ file is a simple list of key-value pairs used to configure environment variables. Here's how it works:

* One variable per line: `KEY=VALUE`.
* Strings with spaces or special characters should be quoted with double quotes.
* Lines starting with `#` are comments and ignored.
* Empty lines are ignored.
* To insert a value that contains double quotes, enclose the entire string in single quotes, e.g.: `MY_VAR='"my-quoted-value"'`.
* Use `true` or `false` for booleans. Don't use `1` or `0`.
* File paths can be absolute or relative; relative paths are based on the working directory of the Meergo process.

An example of _.env_ file syntax:

```ini
# Server settings
MEERGO_HTTP_HOST=127.0.0.1
MEERGO_HTTP_PORT=2022

# TLS settings
MEERGO_HTTP_TLS_ENABLED=true
MEERGO_HTTP_TLS_CERT_FILE="/etc/ssl/certs/my meergo cert.crt"
MEERGO_HTTP_TLS_KEY_FILE='/etc/ssl/private/meergo.key'

# Commented out value
#MEERGO_HTTP_PORT=8080
```

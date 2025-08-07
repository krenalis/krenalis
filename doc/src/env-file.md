{% extends "/layouts/doc.html" %}
{% macro Title string %}.env file{% end %}
{% Article %}

# .env file

> This section applies if you configure Meergo via the .env file. **If you run Meergo with docker**, or **if you pass environment variables directly to the Meergo process**, you can **skip this section**.

Once you have obtained the `meergo` executable, follow these steps to configure the application:

1. **Choose a directory** on your filesystem — this will be the working directory where you will run Meergo.
2. Download the example configuration file [`meergo.example.env`](https://github.com/meergo/meergo/blob/main/cmd/meergo/meergo.example.env) and copy it into the chosen directory as `.env`.
3. Edit `.env` to match your environment and requirements.

> The `.env` file contains the definition of environment variables and is sourced by Meergo at startup. It is therefore possible, alternatively and according to the needs of the environment in which Meergo is to be run, to define the environment variables before starting Meergo, without using the `.env` file.

Next, you’ll need to set up certificates (if using HTTPS), configure the database, and launch the application.

## .env file syntax

The `.env` file is a simple list of key-value pairs used to configure environment variables. Here's how it works:

* One variable per line: `KEY=VALUE`.
* Strings with spaces or special characters should be quoted with double quotes.
* Lines starting with `#` are comments and ignored.
* Empty lines are ignored.
* To insert a value that contains double quotes, enclose the entire string in single quotes, e.g.: `MY_VAR='"my-quoted-value"'`.
* Use `true` or `false` for booleans. Don't use `1` or `0`.
* File paths can be absolute or relative; relative paths are based on the working directory of the Meergo process.

### Examples

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

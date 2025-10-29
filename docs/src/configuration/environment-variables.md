{% extends "/layouts/doc.html" %}
{% macro Title string %}Environment variables{% end %}
{% Article %}

# Environment variables

This section lists all environment variables that Meergo reads at startup. Meergo will also attempt to load variables from [the _.env_ file](the-env-file), if present.

## Database

Settings used to access the PostgreSQL server used by Meergo.

| Variable                    | Default     | Description                                                  |
|-----------------------------|-------------|--------------------------------------------------------------|
| `MEERGO_DB_HOST`            | `127.0.0.1` | Address of the PostgreSQL server. Example: `localhost`.      |
| `MEERGO_DB_PORT`            | `5432`      | Port number used by PostgreSQL.                              |
| `MEERGO_DB_USERNAME`        |             | PostgreSQL username.                                         |
| `MEERGO_DB_PASSWORD`        |             | PostgreSQL password.                                         |
| `MEERGO_DB_DATABASE`        |             | PostgreSQL database name.                                    |
| `MEERGO_DB_SCHEMA`          | `public`    | Schema within the PostgreSQL database to use.                |
| `MEERGO_DB_MAX_CONNECTIONS` | `8`         | Maximum number of connections to PostgreSQL. Must be `>= 2`. |

## HTTP server

| Variable                          | Default                  | Description                                                                                                                                                                                                                                                           |
|-----------------------------------|--------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_HTTP_HOST`                | `127.0.0.1`              | Server address to bind to. It can be an IPv4 address, an IPv6 address, or a hostname. Examples: `localhost`, `[::1]`                                                                                                                                                  |
| `MEERGO_HTTP_PORT`                | `2022`                   | Port number on which the server listens. Example: `443`.                                                                                                                                                                                                              |
| `MEERGO_HTTP_TLS_ENABLED`         | `false`                  | Enable or disable TLS (HTTPS). You can disable TLS if a reverse proxy or load balancer in front of the Meergo server is handling the TLS termination, as it will manage the encryption and decryption of traffic.                                                     |
| `MEERGO_HTTP_TLS_CERT_FILE`       |                          | Path to the TLS certificate file (e.g., `.crt` file). It is required if TLS is enabled.                                                                                                                                                                               |
| `MEERGO_HTTP_TLS_KEY_FILE`        |                          | Path to the private key file associated with the TLS certificate. It is required if TLS is enabled.                                                                                                                                                                   |
| `MEERGO_HTTP_EXTERNAL_URL`        | *(see&nbsp;description)* | Public address through which the server can be accessed from outside the internal network. If not set, it is determined by the combination of `MEERGO_HTTP_TLS_ENABLED`, `MEERGO_HTTP_HOST`, and `MEERGO_HTTP_PORT`. Example: `https://meergo.example.com:8080/`.     |
| `MEERGO_HTTP_EXTERNAL_EVENT_URL`  | *(see&nbsp;description)* | Public address through which the event ingestion endpoint (`/api/v1/events`) can be accessed from outside the internal network. If not set, it is the combination of the external URL and `/api/v1/events`. Example: `https://meergo.example.com:8080/api/v1/events`. |
| `MEERGO_HTTP_READ_HEADER_TIMEOUT` | `2s`                     | Max time to read request headers, including TLS handshake.                                                                                                                                                                                                            |
| `MEERGO_HTTP_READ_TIMEOUT`        | `5s`                     | Max time to read the full request (headers + body), starting from first byte.                                                                                                                                                                                         |
| `MEERGO_HTTP_WRITE_TIMEOUT`       | `30s`                    | Max time for handler execution and sending response. For TLS, includes handshake.                                                                                                                                                                                     |
| `MEERGO_HTTP_IDLE_TIMEOUT`        | `120s`                   | Max idle time between requests on keep-alive connections.                                                                                                                                                                                                             |

## SMTP

These settings are used to send transactional emails.

| Variable               | Default | Description          |
|------------------------|---------|----------------------|
| `MEERGO_SMTP_HOST`     |         | SMTP server address. |
| `MEERGO_SMTP_PORT`     |         | SMTP server port.    |
| `MEERGO_SMTP_USERNAME` |         | SMTP username.       |
| `MEERGO_SMTP_PASSWORD` |         | SMTP password.       |

## Member emails

Settings for emails that are sent to members.

| Variable                                    | Default | Description                                                                                                                                    |
|---------------------------------------------|---------|------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_MEMBER_EMAIL_VERIFICATION_REQUIRED` | `true`  | Require verification of a member's email address when creating a new member. Use `false` to allow adding members without verification.         |
| `MEERGO_MEMBER_EMAIL_FROM`                  |         | "From" address from which member emails are sent (mandatory to send emails to members). Example: `Org <org@example.com>` or `org@example.com`. |


## General

| Variabile                     | Default                                                                  | Description                                                                                                                                                                                                                                                                                             |
|-------------------------------|--------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_MAXMIND_DB_PATH`      |                                                                          | Path to the MaxMind database file (usually with extension '.mmdb') for automatically adding geolocation information to the events. If not set, no geolocation information are automatically added to the events by Meergo, so it is only possibile to provide location information explicitly.          |
| `MEERGO_TERMINATION_DELAY`    | no delay                                                                 | Delay time before gracefully shutting down the server. Example: `1s` (1 second), `200ms` (200 milliseconds).                                                                                                                                                                                            |
| `MEERGO_JAVASCRIPT_SDK_URL`   | `https://cdn.jsdelivr.net/npm/@meergo/javascript-sdk/dist/meergo.min.js` | URL that serves the JavaScript SDK.                                                                                                                                                                                                                                                                     |
| `MEERGO_TELEMETRY_LEVEL`      | `all`                                                                    | Level for telemetry data sent by Meergo: `none` (no telemetry data will be sent), `errors` (only telemetry data related to errors will be sent), `stats` (only telemetry data related to software usage statistics will be sent), `all` (both types of telemetry data (errors and stats) will be sent). |
| `MEERGO_EXTERNAL_ASSETS_URLS` |                                                                          | List of base URLs (comma-separated) from which Meergo retrieves external assets (as icons) related to connector and data warehouse brands. If an image is not available at the first URL, the second is called, and so on, until eventually a default image is used.                                    |

## Transformers

The following settings let you choose how transformation functions are executed. Meergo can run them either using AWS Lambda or locally. In production, you must use AWS Lambda only. The local mode is meant for testing or evaluating Meergo when running with Docker.

<!-- tabs Transformer variables -->

### AWS Lambda

| Variable                                       | Default | Description                                                    |
|------------------------------------------------|---------|----------------------------------------------------------------|
| `MEERGO_TRANSFORMERS_LAMBDA_ACCESS_KEY_ID`     |         | AWS access key ID for Lambda.                                  |
| `MEERGO_TRANSFORMERS_LAMBDA_SECRET_ACCESS_KEY` |         | AWS secret access key for Lambda.                              |
| `MEERGO_TRANSFORMERS_LAMBDA_REGION`            |         | AWS region where Lambda functions are deployed.                |
| `MEERGO_TRANSFORMERS_LAMBDA_ROLE`              |         | AWS IAM Role ARN to be assumed for executing Lambda functions. |
| `MEERGO_TRANSFORMERS_LAMBDA_NODE_RUNTIME`      |         | Node.js runtime version for AWS Lambda. Example: `nodejs22.x`. |
| `MEERGO_TRANSFORMERS_LAMBDA_NODE_LAYER`        |         | (Optional) ARN of a Lambda layer for Node.js functions.        |
| `MEERGO_TRANSFORMERS_LAMBDA_PYTHON_RUNTIME`    |         | Python runtime version for AWS Lambda. Example: `python3.13`.  |
| `MEERGO_TRANSFORMERS_LAMBDA_PYTHON_LAYER`      |         | (Optional) ARN of a Lambda layer for Python functions.         |

### Local

> ⚠️ Configuring transformers for local execution allows the code in transformation functions defined in Meergo to execute arbitrary code on the local machine. Therefore, use with caution and only in trusted contexts.

| Variable                                      | Default | Description                                                                                                                                                                                                                                                   |
|-----------------------------------------------|---------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_TRANSFORMERS_LOCAL_NODE_EXECUTABLE`   |         | Path to the Node.js executable. Example: `/usr/bin/node`.                                                                                                                                                                                                     |
| `MEERGO_TRANSFORMERS_LOCAL_PYTHON_EXECUTABLE` |         | Path to the Python executable. Example: `/usr/bin/python`.                                                                                                                                                                                                    |
| `MEERGO_TRANSFORMERS_LOCAL_FUNCTIONS_DIR`     |         | Directory where local transformation functions are stored (a subdirectory named `meergo-functions` will be created inside the specified path). This directory should be writable by the user executing the Meergo executable. Example: `/var/meergo-project`. |
| `MEERGO_TRANSFORMERS_LOCAL_SUDO_USER`         |         | System user under which to run local transformation function processes. Switching to this user is done in Meergo via `sudo`. If left blank, the current user is retained and `sudo` is not invoked.                                                           |

<!-- end tabs -->

## OAuth providers

Settings for OAuth integrations with external applications.

### HubSpot

| Variable                             | Default | Description                      |
|--------------------------------------|---------|----------------------------------|
| `MEERGO_OAUTH_HUBSPOT_CLIENT_ID`     |         | OAuth Client ID for HubSpot.     |
| `MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET` |         | OAuth Client Secret for HubSpot. |

### Mailchimp

| Variable                               | Default | Description                        |
|----------------------------------------|---------|------------------------------------|
| `MEERGO_OAUTH_MAILCHIMP_CLIENT_ID`     |         | OAuth Client ID for Mailchimp.     |
| `MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET` |         | OAuth Client Secret for Mailchimp. |

## Static connector settings

The following are low-level settings related to the Meergo installation that can be set only via environment variables.

### Filesystem

| Variable                                     | Default | Description                                                                                                                                                                                                                                      |
|----------------------------------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_CONNECTOR_FILESYSTEM_ROOT`           |         | Directory used as root by Filesystem connections. Mandatory when using Filesystem connections.                                                                                                                                                   |
| `MEERGO_CONNECTOR_FILESYSTEM_DISPLAYED_ROOT` |         | Directory displayed as root by Filesystem connections. This is purely visual, useful in cases where you want to display a different root than the actual one in the Meergo interface (e.g., symlinks or directories mounted on virtual volumes). |

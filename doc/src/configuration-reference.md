{% extends "/layouts/doc.html" %}
{% macro Title string %}Configuration reference{% end %}
{% Article %}

# Configuration reference

The environment variables used to configure your Meergo installation are documented here.

Note that this configuration applies to the entire Meergo installation; specific configuration for each workspace can be modified through the Meergo admin (GUI) or via the APIs.

> 💡 For convenience, instead of passing environment variables to the Meergo command, you can declare an .env file. See [the dedicated page](env-file).

Table of contents:

- [General](#general)
- [HTTP server](#http-server)
- [Database](#database)
- [Member emails](#member-emails)
- [SMTP](#smtp)
- [MaxMind](#maxmind)
- [Transformations](#transformations)
  - [AWS Lambda](#aws-lambda)
  - [Local execution](#local-execution)
- [OAuth providers](#oauth-providers)
  - [HubSpot](#hubspot)
  - [Mailchimp](#mailchimp)


## General

| Variabile                   | Default                                                                  | Description                                                                                                                                                                                                                                                                                             |
|-----------------------------|--------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_TERMINATION_DELAY`  | no delay                                                                 | Delay time before gracefully shutting down the server. Example: `1s` (1 second), `200ms` (200 milliseconds).                                                                                                                                                                                            |
| `MEERGO_JAVASCRIPT_SDK_URL` | `https://cdn.jsdelivr.net/npm/@meergo/javascript-sdk/dist/meergo.min.js` | URL that serves the JavaScript SDK.                                                                                                                                                                                                                                                                     |
| `MEERGO_TELEMETRY_LEVEL`    | `all`                                                                    | Level for telemetry data sent by Meergo: `none` (no telemetry data will be sent), `errors` (only telemetry data related to errors will be sent), `stats` (only telemetry data related to software usage statistics will be sent), `all` (both types of telemetry data (errors and stats) will be sent). |

## HTTP server

| Variable                          | Default                        | Description                                                                                                                                                                                                        |
|-----------------------------------|--------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_HTTP_HOST`                | `127.0.0.1`                    | Server address to bind to. It can be an IPv4 address, an IPv6 address, or a hostname. Examples: `localhost`, `[::1]`                                                                                               |
| `MEERGO_HTTP_PORT`                | `9090`                         | Port number on which the server listens. Example: `443`.                                                                                                                                                           |
| `MEERGO_HTTP_TLS_ENABLED`         | `false`                        | Enable or disable TLS (HTTPS). You can disable TLS if a reverse proxy or load balancer in front of the Meergo server is handling the TLS termination, as it will manage the encryption and decryption of traffic.  |
| `MEERGO_HTTP_TLS_CERT_FILE`       |                                | Path to the TLS certificate file (e.g., `.crt` file). It is required if TLS is enabled.                                                                                                                            |
| `MEERGO_HTTP_TLS_KEY_FILE`        |                                | Path to the private key file associated with the TLS certificate. It is required if TLS is enabled.                                                                                                                |
| `MEERGO_HTTP_EXTERNAL_URL`        |                                | publicly accessible URL of the server. If not provided, it is determined by the combination of `MEERGO_HTTP_TLS_ENABLED`, `MEERGO_HTTP_HOST`, and `MEERGO_HTTP_PORT`. Example: `https://meergo.example.com:8080/`. |
| `MEERGO_HTTP_EVENT_URL`           | `/api/v1/events` (same server) | URL of the endpoint receiving events. If not set, assumed to be `/api/v1/events` on the same server. Example: `https://meergo.example.com:8080/api/v1/events`.                                                     |
| `MEERGO_HTTP_READ_HEADER_TIMEOUT` | `2s`                           | Max time to read request headers (incl. TLS handshake).                                                                                                                                                            |
| `MEERGO_HTTP_READ_TIMEOUT`        | `5s`                           | Max time to read full request (headers + body) from first byte.                                                                                                                                                    |
| `MEERGO_HTTP_WRITE_TIMEOUT`       | `10s`                          | Max time for handler execution + sending response (TLS incl. handshake).                                                                                                                                           |
| `MEERGO_HTTP_IDLE_TIMEOUT`        | `120s`                         | Max idle time between requests on keep-alive connections.                                                                                                                                                          | 

## Database

Configuration used to access the PostgreSQL server used by Meergo.

| Variable                    | Default                   | Description                                             |
|-----------------------------|---------------------------|---------------------------------------------------------|
| `MEERGO_DB_HOST`            |                           | Address of the PostgreSQL server. Example: `127.0.0.1`. |
| `MEERGO_DB_PORT`            | `5432`                    | Port number used by PostgreSQL.                         |
| `MEERGO_DB_USERNAME`        |                           | PostgreSQL username.                                    |
| `MEERGO_DB_PASSWORD`        |                           | PostgreSQL password.                                    |
| `MEERGO_DB_DATABASE`        |                           | PostgreSQL database name.                               |
| `MEERGO_DB_SCHEMA`          |                           | Schema within the PostgreSQL database to use.           |
| `MEERGO_DB_MAX_CONNECTIONS` | `max(4,runtime.NumCPU())` | Max number of connections to PostgreSQL.                |

## Member emails

Configuration for emails that are sent to members.

| Variable                                | Default                         | Description                                                                                                                                    |
|-----------------------------------------|---------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_SKIP_MEMBER_EMAIL_VERIFICATION` | `false` (verification required) | Enable or disable the ability to add new members without requiring email verification.                                                         |
| `MEERGO_MEMBER_EMAIL_FROM`              |                                 | "From" address from which member emails are sent (mandatory to send emails to members). Example: `Org <org@example.com>` or `org@example.com`. |

## SMTP

These settings are used to send transactional emails.

| Variable               | Default | Description          |
|------------------------|---------|----------------------|
| `MEERGO_SMTP_HOST`     |         | SMTP server address. |
| `MEERGO_SMTP_PORT`     |         | SMTP server port.    |
| `MEERGO_SMTP_USERNAME` |         | SMTP username.       |
| `MEERGO_SMTP_PASSWORD` |         | SMTP password.       |


## MaxMind

| Variable                 | Default  | Description                                                                                                                                                                                                                                                                                    |
|--------------------------|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_MAXMIND_DB_PATH` |          | Path to the MaxMind database file (usually with extension '.mmdb') for automatically adding geolocation information to the events. If not set, no geolocation information are automatically added to the events by Meergo, so it is only possibile to provide location information explicitly. |


## Transformations

Configuration for executing transformation functions via AWS Lambda or locally. Local execution should only be used for testing and not in production.

> 💡 Note that this configuration does not concern the data transformations themselves, which are set up via the graphical interface for each workspace, but the general configuration provided by the Meergo installation to enable writing transformation functions.

You can configure Meergo to run transformation functions on one of the following:

* [**AWS Lambda**](#aws-lambda). Recommended for production.
* [**Local execution**](#local-execution). Recommended for testing Meergo locally.

### AWS Lambda

| Variable                                          | Default | Description                                                    |
|---------------------------------------------------|---------|----------------------------------------------------------------|
| `MEERGO_TRANSFORMATIONS_LAMBDA_ACCESS_KEY_ID`     |         | AWS access key ID for Lambda.                                  |
| `MEERGO_TRANSFORMATIONS_LAMBDA_SECRET_ACCESS_KEY` |         | AWS secret access key for Lambda.                              |
| `MEERGO_TRANSFORMATIONS_LAMBDA_REGION`            |         | AWS region where Lambda functions are deployed.                |
| `MEERGO_TRANSFORMATIONS_LAMBDA_ROLE`              |         | AWS IAM Role ARN to be assumed for executing Lambda functions. |
| `MEERGO_TRANSFORMATIONS_LAMBDA_NODE_RUNTIME`      |         | Node.js runtime version for AWS Lambda. Example: `nodejs22.x`. |
| `MEERGO_TRANSFORMATIONS_LAMBDA_NODE_LAYER`        |         | (Optional) ARN of a Lambda layer for Node.js functions.        |
| `MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_RUNTIME`    |         | Python runtime version for AWS Lambda. Example: `python3.13`.  |
| `MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_LAYER`      |         | (Optional) ARN of a Lambda layer for Python functions.         |

### Local execution

> ⚠️ Configuring transformations for local execution allows the code in transformation functions defined in Meergo to execute arbitrary code on the local machine. Therefore, use with caution and only in trusted contexts.

| Variable                                         | Default | Description                                                                                                                                                                 |
|--------------------------------------------------|---------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MEERGO_TRANSFORMATIONS_LOCAL_NODE_EXECUTABLE`   |         | Path to the Node.js executable. Example: `/usr/bin/node`.                                                                                                                   |
| `MEERGO_TRANSFORMATIONS_LOCAL_PYTHON_EXECUTABLE` |         | Path to the Python executable. Example: `/usr/bin/python`.                                                                                                                  |
| `MEERGO_TRANSFORMATIONS_LOCAL_FUNCTIONS_DIR`     |         | Directory where local transformation functions are stored. This directory should be writable by the user executing the Meergo executable. Example: `/var/meergo/functions`. |

## OAuth providers

Configuration for OAuth integrations with external applications.

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

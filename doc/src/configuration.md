{% extends "/layouts/doc.html" %}
{% macro Title string %}Configuration{% end %}
{% Article %}

# Configuration file

This document describes the available environment variables for configuring Meergo.

These variables can be provided to Meergo when it starts, or they can be declared in a `.env` file located in the same directory where Meergo is started. You can check the example file [`meergo.example.env`](https://github.com/meergo/meergo/blob/experiment-with-env/cmd/meergo/meergo.example.env).

## General settings

- **`MEERGO_ENCRYPTION_KEY`** \
  Key used for encrypting admin cookies and PostgreSQL notifications. It must be a randomly generated 64-byte sequence, encoded in Base64. \
  Example: `QTayM1ovJctpSidiXgqC8AeFsqYGbWTdwGykdJ2ll1fpxkzFI9FtNf+/FepV1c/Zu9g/y+0FsM8d2nust97tlw`

- **`MEERGO_TERMINATION_DELAY`** \
  Delay time before gracefully shutting down the server. If left empty, the server will initiate a graceful shutdown immediately after receiving the termination signal, without waiting for the specified delay. \
  Example: `1s` (1 second), `200ms` (200 milliseconds) 

- **`MEERGO_JAVASCRIPT_SDK_URL`** \
  The URL that serves the JavaScript SDK.

  Example `https://my.cdn.meergo.example.com/javascript-sdk/dist/meergo.min.js`.

  If not provided, it is assumed that the JavaScript SDK is served by the same server that Meergo runs on, at the path `/javascript-sdk/dist/meergo.min.js`.

- **`MEERGO_DISABLE_TELEMETRY`** \
  When this variable is set to "true", sending telemetry data (such as errors, crash reports) is disabled.

  By default, telemetry is enabled.

## HTTP server configuration

- **`MEERGO_HTTP_HOST`** \
  The server address to bind to. It can be an IPv4 address, an IPv6 address, or a hostname. \
  Examples: `127.0.0.1`, `[::1]`, `localhost`.

- **`MEERGO_HTTP_PORT`** \
  The port number on which the server listens. \
  Example: `9090`

- **`MEERGO_HTTP_TLS_ENABLED`** \
  Enable or disable TLS (HTTPS). You can disable TLS if a reverse proxy or load balancer in front of the Meergo server is handling the TLS termination, as it will manage the encryption and decryption of traffic.

- **`MEERGO_HTTP_TLS_CERT_FILE`** \
  Path to the TLS certificate file (e.g., `.crt` file). It is required if TLS is enabled.

- **`MEERGO_HTTP_TLS_KEY_FILE`** \
  Path to the private key file associated with the TLS certificate. It is required if TLS is enabled.

- **`MEERGO_HTTP_EXTERNAL_URL`** \
  The publicly accessible URL of the server. If not provided, it is determined by the combination of `MEERGO_HTTP_TLS_ENABLED`, `MEERGO_HTTP_HOST`, and `MEERGO_HTTP_PORT`. \
  Example: `https://meergo.example.com:8080/`

- **`MEERGO_HTTP_EVENT_URL`** \
  The URL of the endpoint that receives the events.

  Example: `https://meergo.example.com:8080/api/v1/events`

  If not provided, the event endpoint is assumed to be on the same server as Meergo at `/api/v1/events`.

## Database configuration

Configuration used to access the PostgreSQL server used by Meergo.

- **`MEERGO_DB_HOST`** \
  Address of the PostgreSQL database server. \
  Example: `127.0.0.1`

- **`MEERGO_DB_PORT`** \
  Port number used by PostgreSQL. \
  By default, the port is `5432`.

- **`MEERGO_DB_USERNAME`** \
  PostgreSQL database username.

- **`MEERGO_DB_PASSWORD`** \
  Password for the PostgreSQL user.

- **`MEERGO_DB_DATABASE`** \
  Name of the PostgreSQL database.

- **`MEERGO_DB_SCHEMA`** \
  Specific schema within the PostgreSQL database to use.

## SMTP configuration

These settings are used to send transactional emails.

- **`MEERGO_SMTP_HOST`** \
  SMTP server address.

- **`MEERGO_SMTP_PORT`** \
  SMTP server port number.

- **`MEERGO_SMTP_USERNAME`** \
  Username for SMTP authentication.

- **`MEERGO_SMTP_PASSWORD`** \
  Password for SMTP authentication.

## Transformations

Configuration for executing transformation functions via AWS Lambda or locally. Local execution should only be used for testing and not in production.

### AWS Lambda

- **`MEERGO_TRANSFORMATIONS_LAMBDA_ACCESS_KEY_ID`** \
  AWS access key ID for Lambda.

- **`MEERGO_TRANSFORMATIONS_LAMBDA_SECRET_ACCESS_KEY`** \
  AWS secret access key for Lambda.

- **`MEERGO_TRANSFORMATIONS_LAMBDA_REGION`** \
  AWS region where Lambda functions are deployed.

- **`MEERGO_TRANSFORMATIONS_LAMBDA_ROLE`** \
  AWS IAM Role ARN to be assumed for executing Lambda functions.

#### Node.js settings

- **`MEERGO_TRANSFORMATIONS_LAMBDA_NODE_RUNTIME`** \
  Node.js runtime version for AWS Lambda. \
  Example: `nodejs22.x`

- **`MEERGO_TRANSFORMATIONS_LAMBDA_NODE_LAYER`** \
  (Optional) ARN of a Lambda layer for Node.js functions.

#### Python settings

- **`MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_RUNTIME`** \
  Python runtime version for AWS Lambda. \
  Example: `python3.13`

- **`MEERGO_TRANSFORMATIONS_LAMBDA_PYTHON_LAYER`** \
  (Optional) ARN of a Lambda layer for Python functions.

### Local execution

- **`MEERGO_TRANSFORMATIONS_LOCAL_NODE_EXECUTABLE`** \
  Path to the Node.js executable. \
  Example: `/usr/bin/node`

- **`MEERGO_TRANSFORMATIONS_LOCAL_PYTHON_EXECUTABLE`** \
  Path to the Python executable. \
  Example: `/usr/bin/python`

- **`MEERGO_TRANSFORMATIONS_LOCAL_FUNCTIONS_DIR`** \
  Directory where local transformation functions are stored. This directory should be writable by the user executing the Meergo executable. \
  Example: `/var/meergo/functions`

## OAuth providers

Configuration for OAuth integrations with external applications.

### HubSpot

- **`MEERGO_OAUTH_HUBSPOT_CLIENT_ID`** \
  OAuth Client ID for HubSpot.

- **`MEERGO_OAUTH_HUBSPOT_CLIENT_SECRET`** \
  OAuth Client Secret for HubSpot.

### Mailchimp

- **`MEERGO_OAUTH_MAILCHIMP_CLIENT_ID`** \
  OAuth Client ID for Mailchimp.

- **`MEERGO_OAUTH_MAILCHIMP_CLIENT_SECRET`** \
  OAuth Client Secret for Mailchimp.

## Telemetry

- **`MEERGO_TELEMETRY_ENABLE`** \
  Enable or disable [telemetry](telemetry) to collect anonymous usage statistics.
  By default, the telemetry is disabled.

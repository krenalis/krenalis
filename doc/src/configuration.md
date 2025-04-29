{% extends "/layouts/doc.html" %}
{% macro Title string %}Configuration{% end %}
{% Article %}

# Configuration file

This document describes the available configuration options for the [`config.yaml`](https://github.com/meergo/meergo/blob/main/cmd/meergo/config.example.yaml) file.

## General settings

- **`encryptionKey`** \
  Key used for encrypting admin cookies and PostgreSQL notifications. It must be a randomly generated 64-byte sequence, encoded in Base64. \
  Example: `QTayM1ovJctpSidiXgqC8AeFsqYGbWTdwGykdJ2ll1fpxkzFI9FtNf+/FepV1c/Zu9g/y+0FsM8d2nust97tlw`

- **`terminationDelay`** \
  Delay time before gracefully shutting down the server. If left empty, the server will initiate a graceful shutdown immediately after receiving the termination signal, without waiting for the specified delay. \
  Example: `1s` (1 second), `200ms` (200 milliseconds) 

## HTTP server configuration (`http`)

- **`host`** \
  The server address to bind to. It can be an IPv4 address, an IPv6 address, or a hostname. \
  Examples: `127.0.0.1`, `[::1]`, `localhost`.

- **`port`** \
  The port number on which the server listens. \
  Example: `9090`

- **`tls`**
    - **`enabled`** \
      Enable or disable TLS (HTTPS). You can disable TLS if a reverse proxy or load balancer in front of the Meergo server is handling the TLS termination, as it will manage the encryption and decryption of traffic.
    - **`certFile`** \
      Path to the TLS certificate file (e.g., `.crt` file). It is required if TLS is enabled.
    - **`keyFile`** \
      Path to the private key file associated with the TLS certificate. It is required if TLS is enabled.

- **`externalURL`** \
  The publicly accessible URL of the server. If not provided, it is determined by the combination of `http.tls.enabled`, `http.host`, and `http.port`. \
  Example: `https://meergo.example.com:8080/`

- **`cdnURL`** \
  The URL of the CDN that serves the admin files and the Meergo JavaScript SDK.

  Example `https://my.cdn.meergo.example.com`.

  If not provided, it is assumed that these files are served from the same server that Meergo runs on.

- **`eventURL`** \
  The URL of the endpoint that receives the events.

  Example: `https://meergo.example.com:8080/api/v1/events`

  If not provided, the event endpoint is assumed to be on the same server as Meergo at `/api/v1/events`.

## Database configuration (`db`)

Configuration used to access the PostgreSQL server used by Meergo.

- **`host`** \
  Address of the PostgreSQL database server. \
  Example: `127.0.0.1`

- **`port`** \
  Port number used by PostgreSQL. \
  Default: `5432`

- **`username`** \
  PostgreSQL database username.

- **`password`** \
  Password for the PostgreSQL user.

- **`database`** \
  Name of the PostgreSQL database.

- **`schema`** \
  Specific schema within the PostgreSQL database to use.

## SMTP configuration (`smtp`)

These settings are used to send transactional emails.

- **`host`** \
  SMTP server address.

- **`port`** \
  SMTP server port number.

- **`user`** \
  Username for SMTP authentication.

- **`pass`** \
  Password for SMTP authentication.

## Transformations (`transformations`)

Configuration for executing transformation functions via AWS Lambda or locally. Local execution should only be used for testing and not in production.

### AWS Lambda (`lambda`)

- **`accessKeyID`** \
  AWS access key ID for Lambda.

- **`secretAccessKey`** \
  AWS secret access key for Lambda.

- **`region`** \
  AWS region where Lambda functions are deployed.

- **`role`** \
  AWS IAM Role ARN to be assumed for executing Lambda functions.

#### Node.js settings (`node`)

- **`runtime`** \
  Node.js runtime version for AWS Lambda. \
  Example: `nodejs22.x`

- **`layer`** \
  (Optional) ARN of a Lambda layer for Node.js functions.

#### Python settings (`python`)

- **`runtime`** \
  Python runtime version for AWS Lambda. \
  Example: `python3.13`

- **`layer`** \
  (Optional) ARN of a Lambda layer for Python functions.

### Local execution (`local`)

- **`nodeExecutable`** \
  Path to the Node.js executable. \
  Example: `/usr/bin/node`

- **`pythonExecutable`** \
  Path to the Python executable. \
  Example: `/usr/bin/python`

- **`functionsDir`** \
  Directory where local transformation functions are stored. This directory should be writable by the user executing the Meergo executable. \
  Example: `/var/meergo/functions`

## OAuth providers (`oauth`)

Configuration for OAuth integrations with external applications.

### HubSpot

- **`clientID`** \
  OAuth Client ID for HubSpot.

- **`clientSecret`** \
  OAuth Client Secret for HubSpot.

### Mailchimp

- **`clientID`** \
  OAuth Client ID for Mailchimp.

- **`clientSecret`** \
  OAuth Client Secret for Mailchimp.

## Telemetry (`telemetry`)

- **`enable`** \
  Enable or disable [telemetry](telemetry) to collect anonymous usage statistics.
  Default: `false`

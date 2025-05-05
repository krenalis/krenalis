{% extends "/layouts/doc.html" %}
{% macro Title string %}Telemetry{% end %}
{% Article %}

# Telemetry

Meergo supports the transmission of telemetry data to an **OpenTelemetry collector** of your choice. This capability allows developers to send traces, metrics, and logs, which are essential for feature development and performance monitoring. By leveraging telemetry, you can gain insights into the behavior of Meergo, identify bottlenecks, and enhance overall system performance.

## Enabling telemetry

By default, telemetry is disabled in Meergo to ensure that applications can run without the overhead of data collection unless explicitly configured. To enable telemetry, you need to set the environment variable `MEERGO_TELEMETRY_ENABLE` to `true`.

After enabling telemetry, it is crucial to configure the system to send the collected data to an OpenTelemetry collector. This step is necessary to ensure that the telemetry data is properly processed and stored for analysis.

## Configuring telemetry

The configuration of telemetry in Meergo is accomplished through the use of environment variables. These variables must be passed to the execution environment of Meergo, and they will be read by the **OpenTelemetry SDK** implemented in Go.

The specific environment variables required for configuration are documented in the [official OpenTelemetry documentation](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/). It is important to refer to this documentation to understand which variables you need to set and what values are appropriate for your use case. Common variables may include settings for the collector endpoint, sampling rates, and authentication credentials, among others.
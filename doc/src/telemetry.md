{% extends "/layouts/doc.html" %}
{% macro Title string %}Telemetry{% end %}
{% Article %}

# Telemetry

By default, Meergo sends telemetry data containing information about errors and crashes that occur. This helps Meergo developers identify issues and resolve them.

This behavior can be changed by setting the environment variable `MEERGO_TELEMETRY_LEVEL`. For more details, see the [configuration](configuration) section.

Meergo also supports, in a provisional and experimental way, sending telemetry data to an OpenTelemetry collector. This is a feature separate and independent from error telemetry.

## OpenTelemetry (experimental)

As an experimental, additional, and only partially implemented feature, Meergo supports the transmission of telemetry data to an **OpenTelemetry collector** of your choice.

This capability allows developers to send traces, metrics, and logs, which are essential for feature development and performance monitoring. By leveraging telemetry, you can gain insights into the behavior of Meergo, identify bottlenecks, and enhance overall system performance.

**Important**: sending data to the OpenTelemetry collector is an independent feature and separate from sending errors as telemetry data, which is discussed above. **So enabling, disabling or configuring OpenTelemetry does not impact the sending of telemetry information about crashes and errors, which are configured separately**.

### Enabling

By default, sending telemetry to an OpenTelemetry collector is disabled in Meergo. To enable this, you need to set the environment variable `MEERGO_OPEN_TELEMETRY_ENABLE` to `true`.

After enabling telemetry, it is crucial to configure the system to send the collected data to an OpenTelemetry collector. This step is necessary to ensure that the telemetry data is properly processed and stored for analysis.

### Configuration

The configuration of OpenTelemetry collector in Meergo is accomplished through the use of environment variables. These variables must be passed to the execution environment of Meergo, and they will be read by the **OpenTelemetry SDK** implemented in Go.

The specific environment variables required for configuration are documented in the [official OpenTelemetry documentation](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/). It is important to refer to this documentation to understand which variables you need to set and what values are appropriate for your use case. Common variables may include settings for the collector endpoint, sampling rates, and authentication credentials, among others.

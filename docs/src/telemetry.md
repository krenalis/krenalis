{% extends "/layouts/doc.html" %}
{% macro Title string %}Telemetry{% end %}
{% Article %}

# Telemetry

By default, Meergo sends telemetry data containing information about errors and crashes that occur. This helps Meergo developers identify issues and resolve them.

This behavior can be changed by setting the environment variable `MEERGO_TELEMETRY_LEVEL`. For more details, see the [configuration reference](configuration/reference#general-settings).

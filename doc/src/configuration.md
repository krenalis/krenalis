{% extends "/layouts/doc.html" %}
{% macro Title string %}Configuration{% end %}
{% Article %}

# Configuration

This document describes the available environment variables for configuring Meergo. In particular:

* **When running Meergo with Docker**, environment variables for configuration are defined inside the [`compose.yaml`](https://github.com/meergo/meergo/blob/main/compose.yaml) file, which provides a default to start trying out Meergo.
* **When running Meergo launching an executable**, these variables can be provided to Meergo when it starts, or they can be declared in a `.env` file located in the same directory where Meergo is started. You can check the example file [`meergo.example.env`](https://github.com/meergo/meergo/blob/main/cmd/meergo/meergo.example.env).

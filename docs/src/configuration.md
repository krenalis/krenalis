{% extends "/layouts/doc.html" %}
{% macro Title string %}Configuration{% end %}
{% Article %}

# Configuration

Most Meergo settings can be managed at runtime through the Admin console or the API. However, some settings are only read at startup from environment variables, such as those that configure the Meergo server, HTTPS certificates, the internal database, and other core infrastructure.

This section describes the [environment variables](configuration/environment-variables) and how to configure them.

* If you installed Meergo [using Docker](installation/using-docker), you can configure these variables in the _compose.yml_ file in your _meergo_ directory.

* If you installed Meergo [from pre-compiled binaries](installation/pre-compiled-binaries) or built it [from source](installation/from-source), you can configure them in [the _.env_ file](configuration/the-env-file).

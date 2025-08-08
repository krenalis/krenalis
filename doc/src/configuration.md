{% extends "/layouts/doc.html" %}
{% macro Title string %}Configuration{% end %}
{% Article %}

# Configuration

Meergo is configured at two distinct levels simultaneously:

* an **installation-wide configuration**, which is read when Meergo starts via environment variables. This is a more static form of configuration that is shared across all members and workspaces in the installation. This concerns the Meergo server, HTTP certificates, Meergo's internal database, etc...
* a **workspace-specific configuration**, which can be easily modified at any time via the GUI by the Meergo admin. This is the most dynamic form of configuration, which sets things like the *data warehouse*, *connections*, *actions*, *transformations*, *identity resolution*, etc.

**This section documents the first point, the installation-wide configuration**. The workspace configuration is documented in detail on other pages of this manual, because it is the most important configuration for making the best use of the software.

## Where to go from here?

* [**If you're running Meergo through Docker**](./using-docker**), you can probably skip directly to the [documentation for the individual environment variables](./configuration-reference), as these are declared in the `compose.yml` file.

* If you're **running Meergo from [pre-compiled binaries](./pre-compiled-binaries) or [from source](./from-source)**, you may find it helpful to see [the documentation for the `.env` file](env-file).

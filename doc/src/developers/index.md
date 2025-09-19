{% extends "/layouts/doc.html" %}
{% macro Title string %}Developers{% end %}
{% Article %}

# Developers

Learn how to create custom Meergo connectors with ease. 

### Building connectors

Meergo comes with several ready-to-use connectors, but if needed, you can create your own. A connector is basically a Go package that gets compiled along with the Meergo executable.

Creating a new connector involves:

1. Writing the package according to the documentation, based on the type of connector you want to create.

    - [App connectors](connectors/app)
    - [Database connectors](connectors/database)
    - [File connectors](connectors/file)
    - [File Storage connectors](connectors/file-storage)

2. Add the import of the package into the `cmd/meergo/main.go` file.
3. Build the Meergo executable.
4. Test the connector.

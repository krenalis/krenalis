{% extends "/layouts/doc.html" %}
{% macro Title string %}Developers{% end %}
{% Article %}

# Create new connector

Learn how to create Meergo connectors with ease. 

## Building connectors

Meergo comes with several ready-to-use connectors, but if needed, you can create your own. A connector is basically a Go package that gets compiled along with the Meergo executable.

Creating a new connector involves:

1. Writing the package according to the documentation, based on the type of connector you want to create.

    - Applications
        - [APIs](create-new-connector/apis)
        - [SDKs](create-new-connector/sdks)
    - [Databases](create-new-connector/databases)
    - [Files](create-new-connector/files)
    - [File storages](create-new-connector/file-storages)

2. Add the import of the package into the `cmd/meergo/main.go` file.
3. Build the Meergo executable.
4. Test the connector.

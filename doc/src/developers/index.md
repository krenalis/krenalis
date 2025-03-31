{% extends "/layouts/doc.html" %}
{% macro Title string %}Developers{% end %}
{% Article %}

# Developers

Learn how to integrate with Meergo: send events, manage APIs, and create custom connectors with ease. 

### Send events

<ul class="grid-list">
  <li><a href="javascript-sdk"> JavaScript SDK (Browser)</a></li>
  <li><a href="csharp-sdk"> C# SDK</a></li>
  <li><a href="android-sdk"> Android SDK</a></li>
  <li><a href="go-sdk"> Go SDK</a></li>
  <li><a href="java-sdk"> Java SDK</a></li>
  <li><a href="node-sdk"> Node SDK</a></li>
  <li><a href="python-sdk"> Python SDK</a></li>
</ul>

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

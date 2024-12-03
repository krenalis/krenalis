{% extends "/layouts/doc.html" %}
{% macro Title string %}Developers{% end %}
{% Article %}

# Developers

Learn how to integrate with Meergo: send events, manage APIs, and create custom connectors with ease. 

### Send events to Meergo:

<ul class="grid-list">
  <li><a href="javascript-sdk">{{ render "icons/javascript.md" }} JavaScript SDK (Browser)</a></li>
  <li><a href="csharp-sdk">{{ render "icons/dotnet.md" }} C# SDK</a></li>
  <li><a href="android-sdk">{{ render "icons/android.md" }} Android SDK</a></li>
  <li><a href="go-sdk">{{ render "icons/go.md" }} Go SDK</a></li>
  <li><a href="java-sdk">{{ render "icons/java.md" }} Java SDK</a></li>
  <li><a href="node-sdk">{{ render "icons/nodejs.md" }} Node SDK</a></li>
  <li><a href="python-sdk">{{ render "icons/python.md" }} Python SDK</a></li>
</ul>

### Manage APIs

### Create custom connectors:

- [App connectors](extend/connectors/app)
- [Database connectors](extend/connectors/database)
- [File connectors](extend/connectors/file)
- [File Storage connectors](./extend/connectors/file-storage)

{% extends "/layouts/doc.html" %}
{% macro Title string %}Introduction{% end %}
{% Article %}

# API reference

Meergo's API is a REST API that allows you to manage all resources within Meergo.

Requests are made via HTTP to specific endpoints, and arguments are passed using query strings and JSON in the request body. The responses are always returned in JSON format.

[Authentication](/api/authentication) is done through API Keys, which can be managed through the Meergo UI.

Since workspaces are isolated from each other, you can create a workspace specifically for testing, without affecting other workspaces. Additionally, an API key can be restricted to a specific workspace, reducing the risk of accidentally making changes to other workspaces.

For event ingestion, two endpoints are available with specific keys ([write keys](/api/authentication#write-keys)). These keys limit usage exclusively to event ingestion for a specific connection.

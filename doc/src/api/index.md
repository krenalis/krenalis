{% extends "/layouts/doc.html" %}
{% macro Title string %}Introduction{% end %}
{% Article %}

# API reference

The Meergo API follows REST principles, providing a simple and intuitive design. All requests are sent over HTTPS to resource-specific URLs. Request and response bodies, including error messages, are encoded in JSON for consistency and ease of use.

The API employs standard HTTP status codes to represent errors. These errors are easy for developers to interpret and simple to handle programmatically.

[Authentication](/api/authentication) is done through API keys, which can be managed through the Meergo admin.

For testing purposes, you can create a dedicated workspace without affecting others, as workspaces are fully isolated. Additionally, API keys can be restricted to a specific workspace, eliminating the risk of unintended changes to production environments.

### Event ingestion endpoints

For event ingestion, two endpoints are available, each accessible with specific keys ([event write keys](/api/authentication#event-write-keys)) that are exclusively limited to event ingestion.

These two endpoints support cross-origin resource sharing, enabling secure interaction with the API from any website or client-facing application.

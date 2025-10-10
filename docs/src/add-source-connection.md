{% extends "/layouts/doc.html" %}
{% macro Title string %}Collect users{% end %}
{% Article %}

# Add a Source Connection

A source connection enables you to retrieve customer and event data from an external location, then transform and consolidate it within your data warehouse. Meergo supports multiple sources connections, enabling you to collect data and events from different systems.

Each source connection represents an external system (e.g., a CRM, website, app, or database).
Within each connection, you define actions that determine what type of data to import (user data or event data) and how it should be transformed before being stored in your company’s data warehouse.

You can create a source connection from the **Meergo Admin console**, on the **Connections** page of a workspace, by clicking the **Add a new source ⊕** button. See the [documentation for specific source connectors](integrations/sources) for detailed instructions.
You can also create a source connection from the **Sources** page.

Alternatively, you can create a source connection via the [Create connection API endpoint](/api/connections#create-connection).

Next: [Add a connection action](/add-connection-action)

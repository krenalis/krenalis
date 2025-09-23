{% extends "/layouts/doc.html" %}
{% macro Title string %}Connections{% end %}
{% Article %}

# Connections

A connection enables Meergo to retrieve customer and event data from an external source location or send them to an external destination location.

An external location can be:

* Your website, mobile app, or server that collects user events.
* An application capable of receiving events.
* An application that stores data of your customers.
* A database or file containing your customer data.

To connect Meergo to an external location, add a connection. Choose a **source connection** if you want to retrieve data or events, or a **destination connection** if you want to send data or events.

There are many connection types that you can add based on the type of the external location. For example, add a source JavaScript connection if you want to receive events from your website, and add a destination HubSpot connection if you want to update your customers on HubSpot with the unified customer data in your data warehouse

### Sources 

A source connection enables you to retrieve customer and event data from an external location, then transform and consolidate it within your data warehouse. Meergo allows you to add multiple sources from which to receive data and events.

You can create a source in the "connections" page of a workspace clicking on the **Add a new source ⊕** button.

### Destinations

A destination connection enables you to send customer data, consolidated in your data warehouse, and collected events, possibly after transforming them, to an external location. Meergo allows you to add various destinations to which you can send data and events.

You can create a destination in the "connections" page of a workspace clicking on the **Add new destination ⊕** button.

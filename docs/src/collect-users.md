{% extends "/layouts/doc.html" %}
{% macro Title string %}Collect users{% end %}
{% Article %}

# Collect users

Meergo allows you to collect users in **batch** from apps, databases, and files, as well as in **real time** from events from your websites and applications.

- Import users in batch
- Import users in real time from events
- Ensure data quality and schema validation at the source, reducing errors, minimizing cleanup, and simplifying compliance
- Transform and enrich user data with mappings or function transformations in JavaScript and Python
- Store transformed user data in your data warehouse

Whether you want to collect users in batch or in real time, you need to create a connection using one of the available connectors and then add a connection's action.

Unlike batch imports, real-time imports from events require integration with the websites, mobile applications, ans servers from which events are collected. To facilitate this, Meergo provides [SDKs](integrations/#send-events) tailored to different platforms and programming languages.

## Add a source connection

A source connection enables you to retrieve customer and event data from an external location, then transform and consolidate it within your data warehouse. Meergo allows you to add multiple data sources to receive data and events.

You can create a source connection from the Meergo Admin console on the **Connections** page of a workspace by clicking the **Add a new source ⊕** button. See the [documentation for a specific source connector](integrations/sources) for detailed instructions.

You can also create a connection via the [Create connection endpoint](/api/connections#create-connection) of the API.

## Add a connection's action

Once a source connection is created, you need to add an **action** to the connection to collect users. 

- For files, the file format, name, and any relevant settings
- For databases, the query to execute for retrieving user data
- Filters to determine which users to import
- For batch imports, whether to perform incremental imports or not
- For files and databases, the columns to use as identity and as last modification time
- How to transform user data, either through mappings or transformation functions

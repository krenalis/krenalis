{% extends "/layouts/doc.html" %}
{% macro Title string %}Collect users{% end %}
{% Article %}

# Collect users

Meergo allows you to collect users in **batch** (from apps, databases, or files) and in **real time** (from events from your websites and applications).

- Import users in batch
- Import users in real time from events
- Ensure data quality and schema validation at the source, reducing errors, minimizing cleanup, and simplifying compliance
- Transform and enrich user data with mappings or function transformations in JavaScript or Python
- Store transformed user data directly in your data warehouse

Whether you collect users in batch or in real time, you need to **create a connection** using one of the available connectors and then **add an action** to that connection.

Unlike batch imports, **real-time imports** from events require integration with the websites, mobile applications, and servers from which events are collected. To facilitate this, Meergo provides [SDKs](integrations/#send-events) tailored to different platforms and programming languages.

Next: [Add a source connection](/add-source-connection)
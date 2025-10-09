{% extends "/layouts/doc.html" %}
{% macro Title string %}Transformations{% end %}
{% Article %}

# Transformations

**Meergo Transformations** let you write custom mappings and functions to cover specific scenarios:

* Transform imported user data (from apps, files, databases, or events) to fit your **customer model** schema before loading it into your data warehouse.
* Transform unified user data during export so it matches the target app or database schema.
* Transform real-time events from source systems into the correct format and fields required by destination apps.

In short, Meergo Transformations give you full control over how data moves, adapts, and stays consistent across every system.

Transformations can be written either through field [mappings](transformations/mapping) between schemas or by using custom functions written in [JavaScript](transformations/javascript) or [Python](transformations/python). They can be defined directly from the Admin console or through the [API endpoints for actions](/api/actions).

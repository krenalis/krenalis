{% extends "/layouts/doc.html" %}
{% macro Title string %}Altering User Schema{% end %}
{% Article %}

# Altering User Schema

Once the user model is created, it can be altered with operations such as adding, removing, and modifying properties to reflect the model that best represents the users you want to manage in Meergo.

The user schema can be altered with the UI.

## Supported operations

When altering the user schema, these operations are supported:

* **adding** properties **at any level**
* **dropping** properties **at any level**
* **renaming** properties **at any level**
* **reordering** properties **at top level** and **within object properties**
* **changing labels** and **descriptions** of properties **at any level**

Any other operation (as changing a property type) is not supported.

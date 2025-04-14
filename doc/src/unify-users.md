{% extends "/layouts/doc.html" %}
{% macro Title string %}Unify users{% end %}
{% Article %}

# Unify users

Meergo resolves customer identities and unifies user data from various sources (apps, databases, files, events) to provide a 360-degree view of all your customers. The data is stored and unified directly in your data warehouse, serving as a single source of truth for customer data. Meergo currently supports PostgreSQL and Snowflake as data warehouses. 

## How it works

- User data, retrieved through connections, is stored as identities in the data warehouse.
- Based on these identities, Meergo builds the identity graph.
- From the graph, it constructs customer profiles by unifying them based on identifiers.
- Stores these profiles in the customers table of the data warehouse.

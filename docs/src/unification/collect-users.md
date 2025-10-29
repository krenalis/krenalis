{% extends "/layouts/doc.html" %}
{% macro Title string %}Collect users{% end %}
{% Article %}

# Collect users
## Learn how to collect user data from any source.

Meergo lets you gather user information from multiple sources: real-time event streams from your websites and apps, and batch data from databases, files, or SaaS applications. Collecting users is the foundation for building a unified customer profile used for analytics, segmentation, and activation.

## Available sources

Each source has a detailed guide with configuration examples and supported options.

{{ render "_includes/sources-cards.html" }}

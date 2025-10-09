{% extends "/layouts/doc.html" %}
{% macro Title string %}Collect users{% end %}
{% Article %}

# Map and Harmonize your data

Meergo ensures that data imported from different systems is consistent, reliable, and ready for unification, directly at the connection action level.
During the configuration of a user import action (including those that collect users from events), you can harmonize incoming data by creating mappings between the source fields and the Customer Model properties defined in Meergo.

Mapping can be done in two ways:

**Visually**, by matching source fields with Customer Model properties in the Meergo interface.

**Programmatically**, using JavaScript or Python transformations to adjust, combine, or reformat input data before storage.

By managing harmonization at this stage, Meergo provides three critical benefits:

**Standardization** - Meergo transforms all incoming data into a normalized structure and format. This ensures that later processes (identity resolution, deduplication, analytics, segmentation, etc.) work on clean, consistent data, ensuring data uniformity across all sources.

**Integration** - Each connection action maps and aligns fields from its source system to the **Customer Model** defined in Meergo. This schema-level integration ensures that data from different systems follows a common structure, preserving source-level detail while enabling seamless downstream unification and analysis.

**Discrepancy Management** - Meergo validates incoming data within each connection action, identifying and resolving inconsistencies such as format mismatches, duplicates, or missing fields. By ensuring clean, reliable data at the source, Meergo improves the quality and stability of your unified customer profiles downstream.

> Note on Mapping:
Mapping and harmonization are applied only to user import actions - even when users are collected via real-time events. These processes ensure that user attributes conform to the Customer Model, supporting standardization, schema integration, and discrepancy management.
Event-only actions, on the other hand, do not use mapping. Event data can be filtered and transformed, but it is not aligned to the Customer Model at this stage.

This harmonization process guarantees that all data entering your workspace adheres to a consistent structure, forming a solid foundation for identity resolution, profile unification, and data activation across your entire ecosystem.
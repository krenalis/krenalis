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

## Add a Source Connection

A source connection enables you to retrieve customer and event data from an external location, then transform and consolidate it within your data warehouse. Meergo supports multiple sources connections, enabling you to collect data and events from different systems.

Each source connection represents an external system (e.g., a CRM, website, app, or database).
Within each connection, you define actions that determine what type of data to import (user data or event data) and how it should be transformed before being stored in your company’s data warehouse.

You can create a source connection from the **Meergo Admin console**, on the **Connections** page of a workspace, by clicking the **Add a new source ⊕** button. See the [documentation for specific source connectors](integrations/sources) for detailed instructions.
You can also create a source connection from the **Sources** page.

Alternatively, you can create a connection via the [Create connection API endpoint](/api/connections#create-connection).

## Add a Connection Action

Once a source connection has been created, you need to add one or more actions to define what kind of data the connection will collect.
Each action specifies how Meergo should import, transform, and store data from the source.

Meergo supports two main types of connection actions:

**User Imports** – for collecting customer profile data (attributes, identifiers, metadata). User Import Actions populate the customer schema, forming the basis for identity resolution and profile unification.

**Event Imports** – for collecting behavioral data (events, actions, page views, transactions, etc.). Event Import Actions enrich profiles with real-time behavioral data that can be used for analytics, segmentation, and activation.

The type and configuration of available actions depend on the selected connection source.
For example, a database connection might allow you to define a SQL query and identity columns, while a file connection will require specifying a file format and structure. Similarly, real-time event connections (via SDKs or APIs) include options for event filtering and schema mapping. The event source (e.g., website SDK, mobile SDK, or server-side API) is specify at Source Connection level.


### Add a connection action to import users

A user import action defines how user profile data is retrieved and stored in your data warehouse.

When configuring a user import action, you can specify:

- For files: the file format, name, and any relevant settings

- For databases: the query used to retrieve user data

- Filters to determine which users to import

- For batch imports: whether to perform incremental imports

- For files and databases: the columns used as identity keys and last modification time

- How to transform user data, either through mappings or transformation functions

User import actions are typically used to collect user attributes such as email, name, customer ID, signup date and so on, which contribute to the customer schema and Golden Record.

### Add a connection action to import events

An event import action defines how user activity and behavioral events are collected, transformed, and delivered to your data warehouse.

When configuring an event import action, you can specify:

- The event types or names to capture (e.g., page_view, add_to_cart, purchase)

- Filters to include or exclude specific events

- Event timestamp and user identity fields

- How to transform event data using mappings or transformation functions

Event import actions are typically used to track customer interactions in real time, enriching the user profiles with behavioral context for analytics, segmentation, and personalization.

## Harmonize and standardize your data

One of the key strengths of Meergo is its ability to harmonize data across multiple systems directly at the connection action level.
When configuring an import action, you can define mappings between the incoming data fields and the Customer Model you have created in Meergo. This mapping process ensures that all data—regardless of its source—fits into a consistent, unified schema.

By managing harmonization at this stage, Meergo provides three critical benefits:

Standardization – Meergo automatically enforces consistent formats for common fields such as dates, currencies, and structured attributes, ensuring data uniformity across all sources.

Integration – Fields from different systems are mapped and merged into a single, coherent schema, creating a unified view of the customer without losing source-level granularity.

Discrepancy Management – Meergo identifies and resolves inconsistencies or data conflicts early in the process, improving the accuracy and reliability of downstream identity resolution and analytics.

This harmonization process guarantees that all data entering your workspace adheres to a consistent structure, forming a solid foundation for identity resolution, profile unification, and data activation across your entire ecosystem.
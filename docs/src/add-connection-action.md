{% extends "/layouts/doc.html" %}
{% macro Title string %}Collect users{% end %}
{% Article %}


# Add a Connection Action

Once a source connection has been created, you need to add one or more actions to define what kind of data the connection will collect.
Each action specifies how Meergo should import, transform, and store data from the source.

You can create a connection action from the **Meergo Admin console**, directly inside the **Source Connection** page, by clicking **Add a new action ⊕** button.

Meergo supports two main types of connection actions:

**User Imports** – for collecting customer profile data (attributes, identifiers, metadata). User Import Actions populate the customer schema, forming the basis for identity resolution and profile unification.

**Event Imports** – for collecting behavioral data (events, actions, page views, transactions, etc.). Event Import Actions enrich profiles with real-time behavioral data that can be used for analytics, segmentation, and activation.

The type and configuration of available actions depend on the selected connection source.
For example, a database connection might allow you to define a SQL query and identity columns, while a file connection will require specifying a file format and structure. Similarly, real-time event connections (via SDKs or APIs) include options for event filtering and schema mapping. 

> Note: The event source (e.g., website SDK, mobile SDK, or server-side API) is specify at Source Connection level.


## Add a connection action to import users

A user import action defines how user profile data is retrieved and stored in your data warehouse.

When configuring a user import action, you can specify:

- For files: the file format, name, and any relevant settings

- For databases: the query used to retrieve user data

- Filters to determine which users to import

- For batch imports: whether to perform incremental imports

- For files and databases: the columns used as identity keys and last modification time

- How to transform user data, either through mappings or transformation functions

User import actions are typically used to collect user attributes such as email, name, customer ID, signup date and so on, which contribute to the customer schema and Golden Record.

## Add a connection action to import events

An event import action defines how user activity and behavioral events are collected, transformed, and delivered to your data warehouse.

When configuring an event import action, you can specify:

- The event types or names to capture (e.g., page_view, add_to_cart, purchase)

- Filters to include or exclude specific events

- Event timestamp and user identity fields

- How to transform event data using mappings or transformation functions

Event import actions are typically used to track customer interactions in real time, enriching the user profiles with behavioral context for analytics, segmentation, and personalization.
Only one action to import events can be configured for each connection.

Next: [Map and Harmonize](map-and-harmonize)

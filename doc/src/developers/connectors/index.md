{% extends "/layouts/doc.html" %}
{% macro Title string %}Build a Connector{% end %}
{% Article %}

# Build a Connector

Meergo is easy to add new features to through connectors. Connectors are like plugins that help Meergo work with different kinds of programs and tools. These include:

   * Programs like HubSpot, Mixpanel, Stripe, and more.
   * Databases like PostgreSQL, MySQL, Snowflake, and more.
   * Different types of files like Excel, CSV, Parquet, and more.
   * Places where files are stored like S3, SFTP, HTTP, and more.
   * Streams like Kafka, RabbitMQ, and more.

Meergo can also connect with websites, mobile apps, and servers.

To use Meergo, you should be good at the Go programming language.

That's all you need to get started with Meergo!

# Connector Types

   - [App Connectors](./app.md)
   - [Database Connectors](./database.md)
   - [File Connectors](./file.md)
   - [File Storage Connectors](./file-storage.md)
   - [Settings and UI](./settings-and-ui.md)
   - [Data Values](./data-values.md)

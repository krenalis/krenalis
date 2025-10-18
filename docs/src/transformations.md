{% extends "/layouts/doc.html" %}
{% macro Title string %}Transformations{% end %}
{% Article %}

# Transformations

Transformations are used in actions to process user data during import and export operations, as well as to build the events sent to destination apps based on the events received.

You can set up transformations in two ways:

* **Mapping:** use simple expressions to assign values to specific fields.
* **Functions:** for more advanced scenarios, you can write custom transformation logic in **JavaScript** or **Python**.

Each action that requires a transformation lets you choose the method that best fits your needs.

### How transformations apply to users and events

Depending on **what** is being transformed (users or events) and **when** the transformation occurs (import, export, or event sending), there are four main types:

* **User import transformations:** Applied when importing users from apps, database, or files. These transformations convert user data from the source schema (app, database, or file) into your customer model schema.

* **User export transformations:** Applied when exporting users to apps or databases. These transformations convert user data from your customer model schema into the destination schema expected by the app or database table.

* **User traits transformations:** Applied when users are created or updated from received events. These transformations convert the user traits in the event into your customer model schema.

* **Event sending transformations:** Applied when sending events to apps. These transformations convert the received events into the parameters required to build the outgoing events for destination apps.

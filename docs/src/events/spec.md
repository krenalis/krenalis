{% extends "/layouts/doc.html" %}
{% macro Title string %}Learn events spec{% end %}
{% Article %}

# Learn events spec

The _Meergo Event Specs_ explains what to send when you track events. SDKs handle delivery, but this spec keeps names, fields, and types consistent across web, mobile, and backend so your data stays clean and comparable. Read it to keep your events consistent, debug faster, and let downstream tools work without extra fixes. The rules apply no matter how you send data—via SDKs, backend services, batch jobs, or direct API calls—because the schema is the same in every case.

Calls are requests to the standard event methods:

* [**Page**](spec/page)\
  Answers: *Which web page did they view?*\
  Use when: a web route or URL loads or changes on the client.

* [**Screen**](spec/screen)\
  Answers: *Which app screen did they view?*\
  Use when: a native view opens on iOS, Android, desktop, or TV apps.

* [**Track**](spec/track)\
  Answers: *What did the user do?*\
  Use when: meaningful business actions. Examples: "Product Viewed", "Checkout Started", "Order Completed".

* [**Identify**](spec/identify)\
  Answers: *Who is the user?*\
  Use when: login, signup, profile updates, plan changes, consent updates.

* [**Group**](spec/group)\
  Answers: *Which account or organization is the user part of?*\
  Use when: the user is linked to a company, workspace, or team, or their roles within the group change.

## Event schema

The [event schema](specs/event-schema) defines the structure of an event, similar to how the user schema (Customer Model Schema) defines the structure of a user.
Unlike the user schema, which can be customized for each organization, the event schema is predefined. See more about the [event schema](specs/event-schema).

{% extends "/layouts/doc.html" %}
{% macro Title string %}Start collecting{% end %}
{% Article %}

# Start collecting events
## Use Meergo to start collecting behavioral events.

Mergo allows you to collect user behavior events directly from your websites, mobile and desktop apps, or server applications through SDKs and APIs, and receive them in a single unified schema across all platforms, compatible with Segment and RudderStack. This guide will help you understand how to collect, test, debug, and monitor them.

Before getting started, make sure you have [installed and started](installation) Meergo. Once done, you can manage the sources from which to collect events through the Admin console.

## Source connections

For each website, mobile app, or server where you collect events, you can add a source connection from the Admin console. Creating a separate connection for each source gives you greater control with dedicated authentication keys, its own testing and debugging environment, and its own monitoring system, as well as the ability to independently decide how to process events (load into the data warehouse and send to apps).

### Add a source connection

Procedi come segue per aggiungere una source connection:

1. 

Gli eventi hanno un unico schema indipendentemente da dove e con quale modalità sono stati raccolti. 

The _Meergo Event Spec_ explains what to send when you track events. SDKs handle delivery, but this spec keeps names, fields, and types consistent across web, mobile, and backend so your data stays clean and comparable. Read it to keep your events consistent, debug faster, and let downstream tools work without extra fixes. The rules apply no matter how you send data—via SDKs, backend services, batch jobs, or direct API calls—because the schema is the same in every case.

Calls are requests to the standard event methods:

* [**Page**](specs/page)\
  Answers: *Which web page did they view?*\
  Use when: a web route or URL loads or changes on the client.

* [**Screen**](specs/screen)\
  Answers: *Which app screen did they view?*\
  Use when: a native view opens on iOS, Android, desktop, or TV apps.

* [**Track**](specs/track)\
  Answers: *What did the user do?*\
  Use when: meaningful business actions. Examples: "Product Viewed", "Checkout Started", "Order Completed".

* [**Identify**](specs/identify)\
  Answers: *Who is the user?*\
  Use when: login, signup, profile updates, plan changes, consent updates.

* [**Group**](specs/group)\
  Answers: *Which account or organization is the user part of?*\
  Use when: the user is linked to a company, workspace, or team, or their roles within the group change.

## Event schema

The [event schema](specs/event-schema) defines the structure of an event, similar to how the user schema (Customer Model Schema) defines the structure of a user.
Unlike the user schema, which can be customized for each organization, the event schema is predefined. See more about the [event schema](specs/event-schema).

{% extends "/layouts/doc.html" %}
{% import "external-image.md" %}
{% macro Title string %}Meergo CDP overview{% end %}
{% Article %}

# Overview

## What is Meergo

**Meergo** is a Customer Data Platform (CDP) that helps teams collect user data from multiple sources, unify it, and activate it efficiently and securely. With a developer-first approach and native data warehouse compatibility, Meergo enables you to build flexible data pipelines that integrate seamlessly into your existing tech stack, without duplicating data or creating vendor lock-in.

## Who it's for

Meergo CDP is designed for engineers, analysts, and growth teams, as well as marketing teams who work alongside them, providing complete control over user data infrastructure while making it easy to collect, unify, and activate data.

## Collect, unify, and activate Users

### Unification of user data

Meergo unifies data from multiple sources, creating a complete and consolidated profile, the Golden Record. By merging information from marketing, sales, support, and behavioral events, it delivers an authoritative and accurate view of each user.

### Single source of truth

Meergo acts as the central source of truth for user data, giving companies a 360-degree perspective. All user data and events are stored directly in your warehouse, available in both batch and real time.

### Identity resolution
With identity resolution across devices, channels, and systems, Meergo consolidates fragmented records, enabling businesses to accurately identify users and follow the complete journey across touchpoints.

### Data activation
Unified data becomes immediately actionable. Meergo delivers clean, structured data to marketing platforms, analytics tools, and engagement systems in real time, powering personalized campaigns and data-driven insights.

### Data validation
To ensure accuracy and consistency, Meergo applies schema-based validation at ingestion and delivery. This rigorous process guarantees that the data driving your decisions is always reliable.

## Collect and send Events
 
Use the following SDKs to collect and send events to Meergo. Events can be stored in your data warehouse or sent to applications for activation in real-time.

<ul class="grid-list">
  <li><a href="javascript-sdk">{{ ExternalImage("javascript.svg") }} JavaScript SDK (Browser)</a></li>
  <li><a href="csharp-sdk">{{ ExternalImage("dotnet.svg") }} C# SDK</a></li>
  <li><a href="android-sdk">{{ ExternalImage("android.svg") }} Android SDK</a></li>
  <li><a href="go-sdk">{{ ExternalImage("go.svg") }} Go SDK</a></li>
  <li><a href="java-sdk">{{ ExternalImage("java.svg") }} Java SDK</a></li>
  <li><a href="node-sdk">{{ ExternalImage("node.svg") }} Node SDK</a></li>
  <li><a href="python-sdk">{{ ExternalImage("python.svg") }} Python SDK</a></li>
</ul>

## Getting Started with Meergo

## Installing

There are several ways to get started with Meergo:

* [Using Docker](installation/using-docker). The recommended way to quickly try Meergo with a pre-configured local instance and data warehouse, customizable later.
* [From pre-compiled binaries](installation/pre-compiled-binaries). For more control, run the downloadable binary. Requires manual setup of PostgreSQL and a data warehouse.
* [From source](installation/from-source). The most advanced method, offering full control and flexibility. Recommended for customizing the executable or contributing by building Meergo from source.

Ready to dive in? Check out [Installation](installation) to start using Meergo.

### Connect a warehouse

Once installed, connect Meergo to your data warehouse. Meergo stores user data and events, seamlessly unifying them directly within your data warehouse.

<ul class="grid-list">
  <li><a href="connect-a-warehouse#snowflake">{{ ExternalImage("postgresql.svg")}} PostgreSQL</a></li>
  <li><a href="connect-a-warehouse#postgresql">{{ ExternalImage("snowflake.svg")}} Snowflake</a></li>
</ul>

## Need help?

You can always contact us via support@meergo.com or ..............

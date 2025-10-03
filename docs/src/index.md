{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo CDP overview{% end %}
{% Article %}

# Overview

**Meergo** is a Customer Data Platform (CDP) designed to help teams collect, unify, and activate customer data efficiently and securely. Built with a developer-first approach and native data warehouse compatibility, Meergo empowers you to create flexible data pipelines that fit seamlessly into your existing tech stack. [More about what is Meergo CDP](what-is-meergo).

## Installing

There are several ways to get started with Meergo:

* [Using Docker](installation/using-docker). The recommended way to quickly try Meergo with a pre-configured local instance and data warehouse, customizable later.
* [From pre-compiled binaries](installation/pre-compiled-binaries). For more control, run the downloadable binary. Requires manual setup of PostgreSQL and a data warehouse.
* [From source](installation/from-source). The most advanced method, offering full control and flexibility. Recommended for customizing the executable or contributing by building Meergo from source.

## Connect a warehouse

Meergo stores user data and events, seamlessly unifying them directly within your data warehouse. You can have a data warehouse for each workspace.

<ul class="grid-list">
  <li><a href="connect-a-warehouse#postgresql"> PostgreSQL</a></li>
  <li><a href="connect-a-warehouse#snowflake"> Snowflake</a></li>
</ul>

## Collect, unify, and activate users

Meergo [connectors](connectors/) allows to import and export users from and to apps, databases, storages, and files:

* [Collect users](collect-users)
* Unify users
* Activate users

## Collect and send events
 
Use the following SDKs to collect and send events to Meergo. Events can be stored in your data warehouse or sent to applications for activation in real-time.

<ul class="grid-list">
  <li><a href="javascript-sdk"> JavaScript SDK (Browser)</a></li>
  <li><a href="csharp-sdk"> C# SDK</a></li>
  <li><a href="android-sdk"> Android SDK</a></li>
  <li><a href="go-sdk"> Go SDK</a></li>
  <li><a href="java-sdk"> Java SDK</a></li>
  <li><a href="node-sdk"> Node SDK</a></li>
  <li><a href="python-sdk"> Python SDK</a></li>
</ul>

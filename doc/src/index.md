{% extends "/layouts/doc.html" %}
{% macro Title string %}Meergo CDP overview{% end %}
{% Article %}

# Overview

**Meergo** is a Customer Data Platform (CDP) designed to help teams collect, unify, and activate customer data efficiently and securely. Built with a developer-first approach and native data warehouse compatibility, Meergo empowers you to create flexible data pipelines that fit seamlessly into your existing tech stack. [More about what is Meergo CDP](what-is-meergo).

## Installing

There are several ways to get started with Meergo:

* [Docker](install-meergo#docker). This method is ideal for local development, testing, and prototyping.
* [Pre-packaged binaries](install-meergo#pre-packaged-binaries). A convenient method for quickly setting up Meergo without the need to compile from source.
* [Source code](install-meergo#source-code). Recommended if you wish to customize the executable or contribute to the project by building Meergo directly from the source.

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

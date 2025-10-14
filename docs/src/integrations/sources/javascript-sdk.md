{% extends "/layouts/doc.html" %}
{% macro Title string %}JavaScript SDK (Source){% end %}
{% Article %}

# JavaScript SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_JavaScript_SDK-blue?logo=github)](https://github.com/open2b/analytics-javascript)

Use the JavaScript SDK to integrate event tracking and data analytics into your web application. The SDK lets you send tracking data to your data warehouse and multiple analytics tools through a single interface, so you don't need separate integrations.

This integration simplifies implementation, enabling teams to gain comprehensive insights into user behavior and optimize their applications with ease.

### Key features

* **Track user actions:** Record user interactions such as clicks, page views, and custom events within your application.
* **Connect to analytics tools:** Send tracked data to various analytics tools like Google Analytics, Mixpanel, and others.
* **Manage data easily:** Decide which data goes to which analytics tools, making it simple to organize and change tools later.
* **User data extraction:** Retrieve user traits from tracked events and load them into the data warehouse, where they are unified through Meergo's identity resolution.
* **Lightweight:** No external dependencies and only **16 kB** (compressed), reducing network overhead.
* **Compatible** with the Segment and RudderStack SDKs.

## Guides

| Guide                                                       | What will you learn?                                                      |
|-------------------------------------------------------------|---------------------------------------------------------------------------|
| [Getting&nbsp;started](javascript-sdk/getting-started)      | How to install the SDK in browsers and import it into applications.       |
| [Options](javascript-sdk/options)                           | The available options when installing the SDK.                            |
| [Methods&nbsp;of&nbsp;the&nbsp;SDK](javascript-sdk/methods) | The methods of the SDK, including the methods to send events.             |
| [Storage&nbsp;locations](javascript-sdk/storage-locations)  | Where the users' data is stored and how to manage its storage location.   |
| [Querystring&nbsp;API](javascript-sdk/querystring-api)      | How to activate `identify` and `track` events using the URL query string. |
| [FAQ](javascript-sdk/faq)                                   | Frequently Asked Questions about the JavaScript SDK.                      |

## Minimum supported browsers

Here is a list of the minimum browser versions required to run the JavaScript SDK:

* Chrome 23
* Edge 80
* Safari 7
* Firefox 21
* Opera 14
* IE 11

## SDK source code

The source code of the Meergo JavaScript SDK is [available on GitHub](https://github.com/open2b/analytics-javascript) and distributed under the **MIT license**.

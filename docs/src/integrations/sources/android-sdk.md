{% extends "/layouts/doc.html" %}
{% macro Title string %}Android SDK (Source){% end %}
{% Article %}

# Android SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_Kotlin_SDK-blue?logo=github)](https://github.com/open2b/analytics-kotlin)

Use the Android SDK to add event tracking and analytics to your Android app with minimal setup. The SDK lets you send tracking data to your data warehouse and multiple analytics tools through a single interface, removing the need for separate integrations.

This integration simplifies development and helps teams understand user behavior and improve their apps efficiently.

### Key features

* **Track user actions:** Capture user interactions such as taps, screen views, and custom events.
* **Connect to analytics tools:** Send data to analytics tools like Google Analytics, Mixpanel, and others.
* **Manage data easily:** Choose which data goes to each analytics tool for simple organization and future changes.
* **Respect user privacy:** Comply with privacy standards like GDPR by giving users control over tracking and anonymizing data.
* **Extract user data:** Retrieve user traits from events and load them into your data warehouse for unified analysis.
* **Compatible:** Works seamlessly with the Segment Kotlin SDK.

## Guides

| Guide                                               | What will you learn?                                            |
|-----------------------------------------------------|-----------------------------------------------------------------|
| [Getting&nbsp;started](android-sdk/getting-started) | How to install the SDK and import it into Android applications. |
| [Options](android-sdk/options)                      | The available options when installing the SDK.                  |
| [Methods&nbsp;of&nbsp;SDK](android-sdk/methods)     | The methods of the SDK, including the methods to send events.   |

## SDK source code

The source code of the Meergo Kotlin SDK is [available on GitHub](https://github.com/open2b/analytics-kotlin) and distributed under the MIT license.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Android SDK (Source){% end %}
{% Article %}

# Android SDK (Source)

[![GitHub Repo](https://img.shields.io/badge/Github-Meergo_Kotlin_SDK-blue?logo=github)](https://github.com/open2b/analytics-kotlin)

The Android SDK empowers developers to effortlessly integrate robust event tracking and advanced data analytics directly into their Android applications. Developers can seamlessly transmit tracking data to both the data warehouse and a multiple analytics tools, all without the hassle of individual integrations.

This streamlined approach simplifies the development process, enabling teams to gain comprehensive insights into user behavior and optimize their applications with ease.

### Key features

* Track User Actions: Keep tabs on user actions like taps, screen views, and more on your Android application.
* Connect to Analytics Tools: Send tracked data to various analytics tools like Google Analytics, Mixpanel, and others.
* Manage Data Easily: Decide which data goes to which analytics tools, making it simple to organize and change tools later.
* Respect User Privacy: Follow privacy rules like GDPR by giving users control over tracking and anonymizing data.
* User Data Extraction: Extract user data from events for importing and identity resolution, enabling a deeper understanding of individual users.
* Compatible with the Segment Kotlin SDK.

## Guides

| Guide                                               | What will you learn?                                            |
|-----------------------------------------------------|-----------------------------------------------------------------|
| [Getting&nbsp;started](android-sdk/getting-started) | How to install the SDK and import it into Android applications. |
| [Options](android-sdk/options)                      | The available options when installing the SDK.                  |
| [Methods&nbsp;of&nbsp;SDK](android-sdk/methods)     | The methods of the SDK, including the methods to send events.   |

## SDK source code

The source code of the Meergo Kotlin SDK is [available on GitHub](https://github.com/open2b/analytics-kotlin).

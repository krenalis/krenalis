{% extends "/layouts/doc.html" %}
{% macro Title string %}JavaScript SDK{% end %}
{% Article %}

<span>Send events</span>
# JavaScript SDK

The JavaScript SDK empowers developers to effortlessly integrate robust event tracking and advanced data analytics directly into their web applications. Developers can seamlessly transmit tracking data to both the data warehouse and a multiple analytics tools, all without the hassle of individual integrations.

This streamlined approach simplifies the development process, enabling teams to gain comprehensive insights into user behavior and optimize their applications with ease.

### Key features

* Track User Actions: Keep tabs on user actions like clicks, page views, and more on your website.
* Connect to Analytics Tools: Send tracked data to various analytics tools like Google Analytics, Mixpanel, and others.
* Manage Data Easily: Decide which data goes to which analytics tools, making it simple to organize and change tools later.
* Respect User Privacy: Follow privacy rules like GDPR by giving users control over tracking and anonymizing data.
* User Data Extraction: Extract user data from events for importing and identity resolution, enabling a deeper understanding of individual users.
* Compatible with the Segment and RudderStack SDKs.

## Guides

| Guide                                                      | What will you learn?                                                      |
|------------------------------------------------------------|---------------------------------------------------------------------------|
| [Getting&nbsp;started](javascript-sdk/getting-started)     | How to install the SDK in browsers and import it into applications.       |
| [Options](javascript-sdk/options)                          | The available options when installing the SDK.                            |
| [Methods&nbsp;of&nbsp;SDK](javascript-sdk/methods)         | The methods of the SDK, including the methods to send events.             |
| [Storage&nbsp;locations](javascript-sdk/storage-locations) | Where the users' data is stored and how to manage its storage location.   |
| [Querystring&nbsp;API](javascript-sdk/querystring-api)     | How to activate `identify` and `track` events using the URL query string. |
| [FAQ](javascript-sdk/faq)                                  | Frequently Asked Questions about the JavaScript SDK.                      |

## Minimum Supported Browsers

Here is a list of the minimum browser versions required to run the JavaScript SDK:

* Chrome 23
* Edge 80
* Safari 7
* Firefox 21
* Opera 14
* IE 11
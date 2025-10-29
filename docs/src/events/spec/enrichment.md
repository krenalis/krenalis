{% extends "/layouts/doc.html" %}
{% macro Title string %}Enrichment{% end %}
{% Article %}

# Enrichment

The following table indicates whether each event property is **automatically collected** by the API and SDKs.

<div id="automatically-collected-properties">

| Property                                    | API   | JavaScript SDK | Android SDK | Other SDKs |
|---------------------------------------------|-------|----------------|-------------|------------|
| `user`                                      | ✓[^1] | ✓[^1]          | ✓[^1]       | ✓[^1]      |
| `connectionId`                              | ✓[^2] | ✓[^1]          | ✓[^1]       | ✓[^1]      |
| `anonymousId`                               | ✓[^2] | ✓              | ✓           | ✓[^2]      |
| `channel`                                   |       |                |             |            |
| `category`                                  |       |                |             |            |
| [`context.app`](event-schema#app)           |       |                | ✓           |            |
| [`context.browser`](event-schema#browser)   |       | ✓[^1]          | ✓[^1]       |            |
| [`context.campaign`](event-schema#campaign) |       | ✓              |             |            |
| [`context.device`](event-schema#device)     |       |                | ✓           |            |
| `context.ip`                                | ✓[^2] | ✓[^2]          | ✓[^2]       | ✓[^2]      |
| [`context.library`](event-schema#library)   |       | ✓              | ✓           | ✓          |
| `context.locale`                            |       | ✓              | ✓           |            |
| [`context.location`](event-schema#location) |       | ✓[^2]          | ✓[^2]       |            |
| [`context.network`](event-schema#network)   |       | ✓              | ✓           |            |
| [`context.os`](event-schema#os)             |       | ✓[^1]          | ✓[^1]       |            |
| [`context.page`](event-schema#page)         |       | ✓              |             |            |
| [`context.screen`](event-schema#screen)     |       |                | ✓           |            |
| [`context.session`](event-schema#session)   |       | ✓              | ✓           |            |
| `context.timezone`                          |       | ✓              | ✓           |            |
| `context.userAgent`                         |       | ✓              | ✓           |            |
| `event`                                     |       |                |             |            |
| `groupId`                                   |       |                |             |            |
| `messageId`                                 | ✓[^2] | ✓              | ✓           | ✓          |
| `name`                                      |       |                |             |            |
| `properties`                                |       |                |             |            |
| `receivedAt`                                | ✓[^1] | ✓[^1]          | ✓[^1]       | ✓[^1]      |
| `sentAt`                                    | ✓[^2] | ✓              | ✓           | ✓          |
| `originalTimestamp`                         | ✓[^2] | ✓[^2]          | ✓[^2]       | ✓[^2]      |
| `timestamp`                                 | ✓[^2] | ✓              | ✓           | ✓          |
| `traits`                                    |       |                |             |            |
| `type`                                      |       | ✓              | ✓           | ✓          |
| `previuosId`                                |       |                |             |            |
| `userId`                                    |       |                |             |            |

[^1]: Collected server side by Meergo.
[^2]: Collected server side by Meergo if not provided.

</div>

# Handling event IPs

When using the [JavaScript SDK](/sources/javascript-sdk) and [Android SDK](/sources/android-sdk), the event's IP address is automatically derived from the HTTP request if it's not explicitly provided in the context.
This allows Meergo to automatically capture the browser or mobile device's IP address.

When using a different SDK or sending events directly through the API, and the IP address is not specified in the context, the event will not include an IP address.

### Special IPs and privacy

All SDKs and APIs support a set of special IP values that let you control how the event's IP address is determined or masked.
These values can be used either to adjust the default behavior or to respect user privacy when needed.

| Special IP      | Description                                                                                                                                        |
|-----------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| 255.255.255.255 | Uses the IP address from the HTTP request.                                                                                                         |
| 255.255.255.0   | Mask the request IP with 255.255.255.0, setting the last segment to 0 (e.g., 192.168.45.32 → 192.168.45.0). Use this for partial anonymization.    |
| 255.255.0.0     | Mask the request IP with 255.255.0.0, setting the last two segments to 0 (e.g., 192.168.45.32 → 192.168.0.0). Use this for stronger anonymization. |
| 0.0.0.0         | The event will not include an IP address. Use this when no IP is applicable or when it should not be associated for privacy reasons.               |

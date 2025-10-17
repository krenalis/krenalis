{% extends "/layouts/doc.html" %}
{% macro Title string %}Collected properties{% end %}
{% Article %}

# Collected properties

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

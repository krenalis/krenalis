{% extends "/layouts/doc.html" %}
{% macro Title string %}Screen Call{% end %}
{% Article %}

# Screen call

The screen call enables you to capture instances when a user views a screen and record associated properties or details about that particular screen.

For tracking website page views, it's recommended to utilize the specific [page](page) call.

## When to use the screen call

Use the screen call when there is a change in the displayed content or when a user transitions from one screen to another within your application.

## How to make a screen call

To make a screen call, you can use a Meergo SDK. Refer to its documentation for more details.

The following is an example of how a screen call would appear in Meergo once received and processed:

```json
{
  "anonymousId": "4z7a3b8c1f5e207d6e9b0c4f58",
  "channel": "mobile",
  "context": {
    "ip": "172.16.254.1"
  },
  "messageId": "1b3ff72a-ccbd-22f5-9abc-bb09c6d2a8ef",
  "receivedAt": "2024-01-16T09:03:34.917Z",
  "sentAt": "2024-01-16T09:03:34.781Z",
  "timestamp": "2024-01-16T09:03:34.781Z",
  "name": "Product",
  "properties": {
    "name": "Product Details",
    "category": "E-commerce",
    "label": "Product ABC",
    "additionalInfo": {
      "price": 19.99,
      "color": "green"
    }
  },
  "type": "screen",
  "userId": "123456",
  "version": "1.0"
}
```

## Properties

Properties are like extra details that tell you more about the screen. You can include whatever information you need.

They are accessible in Meergo within a property of the event called `properties` as a JSON value.

### Prefilled properties

Meergo doesn't automatically forward an event's properties to a destination. Instead, the control lies in your hands to choose and specify the properties you want to send when configuring the mapping.

However, when you set up a mapping, Meergo automatically fills in the mapping expressions with the following properties when applicable:

| Property   | JSON&nbsp;Type | Description                                      |
|------------|----------------|--------------------------------------------------|
| `name`     | `String`       | Page name designated as reserved for future use. |

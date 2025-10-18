{% extends "/layouts/doc.html" %}
{% macro Title string %}Page call{% end %}
{% Article %}

# Page

The page call allows you to capture when a user views a page on your website, including any extra details about that specific page.

For tracking app screen views, it's recommended to utilize the specific [screen](screen) call.

## When to use the page call

The page call is used when loading a new page. In Single Page Applications (SPAs), it's also used for logical changes in the page content or transitions, like moving to a different view or route.

## How to make a page call

To make a page call, you can use a Meergo SDK:

<!-- codeblocks sync:sdk Page -->

```javascript
meergo.page('Product Detail', {
    sku: 'SKU-12345',
    name: 'White T-Shirt',
    price: 19.99,
    currency: 'USD'
});
```
Refer to the [JavaScript SDK](/integrations/sources/javascript-sdk) for more details.

```python
analytics.page(
    user_id="user-123",
    name="Product Detail",
    properties={
        "sku": "SKU-12345",
        "name": "White T-Shirt",
        "price": 19.99,
        "currency": "USD",
    },
)
```
Refer to the [Python SDK](/integrations/sources/python) for more details.

```go
client.Enqueue(analytics.Page{
    UserId: "user-123",
    Name:   "Product Detail",
    Properties: map[string]any{
        "sku":      "SKU-12345",
        "name":     "White T-Shirt",
        "price":    19.99,
        "currency": "USD",
    },
})
```
Refer to the [Go SDK](/integrations/sources/go) for more details.

```nodejs
analytics.page({
    userId: 'user-123',
    name: 'Product Detail',
    properties: {
        sku: 'SKU-12345',
        name: 'White T-Shirt',
        price: 19.99,
        currency: 'USD'
    }
});
```
Refer to the [Node.js SDK](/integrations/sources/nodejs) for more details.

```java
analytics.enqueue(PageMessage.builder("Product Detail")
    .userId("user-123")
    .properties(new com.meergo.analytics.messages.Properties()
        .putValue("sku", "SKU-12345")
        .putValue("name", "White T-Shirt")
        .putValue("price", 19.99)
        .putValue("currency", "USD"))
);
```
Refer to the [Java SDK](/integrations/sources/java) for more details.

```csharp
Analytics.Client.Page("user-123", "Product Detail", new Properties {
    { "sku", "SKU-12345" },
    { "name", "White T-Shirt" },
    { "price", 19.99 },
    { "currency", "USD" }
});
```
Refer to the [.Net SDK](/integrations/sources/dotnet) for more details.

<!-- end codeblocks -->

The argument, `'Product Detail'`, is the name of the page. The second argument contains the properties of the event, offering additional context to track page views. This extra information enhances the comprehension of your users' actions and is stored along with the event.

The following is an example of how the previous page call would appear in Meergo once received and processed:
```json
{
  "connectionId": 129402661,
  "anonymousId": "f8d886bf-e1a6-484c-9ded-ac789ec4146b",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Macintosh) Chrome/138 Safari/537.36"
  },
  "messageId": "0a2dd91e-bbac-11e4-8dfc-aa07a5b093db",
  "receivedAt": "2025-10-13T18:07:33.288Z",
  "sentAt": "2025-10-13T18:07:33.051Z",
  "timestamp": "2025-10-13T18:07:33.051Z",
  "properties": {
    "sku": "SKU-12345",
    "name": "White T-Shirt",
    "price": 19.99,
    "currency": "USD",
    "title": "White T-Shirt",
    "url": "https://example.com"
  },
  "traits": {},
  "type": "page",
  "userId": "user-123"
}
```

As you can see, there is much more information than what is provided in SDK examples. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

## Properties

Properties are like extra details that tell you more about the page. You can include whatever information you need.

They are accessible in Meergo within a property of the event called `properties` as a JSON value.

### Prefilled properties

Meergo doesn't automatically forward an event's properties to a destination. Instead, the control lies in your hands to choose and specify the properties you want to send when configuring the mapping.

However, when you set up a mapping, Meergo automatically fills in the mapping expressions with the following properties when applicable:

| Property   | JSON&nbsp;Type      | Description                                                                                                                                                                                                                                |
|------------|---------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `keywords` | `Array` of `String` | List of keywords that outlines the content of the page, similar to those used in HTML meta tags for SEO. This feature is mainly used by content publishers who heavily depend on tracking pageviews, and it is not gathered automatically. |
| `name`     | `String`            | Page name designated as reserved for future use.                                                                                                                                                                                           |
| `path`     | `String`            | Path segment in the page's URL corresponds to the canonical path, which is derived from the default `location.pathname` in JavaScript.                                                                                                     |
| `referrer` | `String`            | complete url of the page the user visited before the current one; corresponding to `document.referrer` in JavaScript.                                                                                                                      |
| `search`   | `String`            | Query string of the URL; corresponding to `location.search` in JavaScript.                                                                                                                                                                 |
| `title`    | `String`            | Title; corresponding to `document.title` in JavaScript.                                                                                                                                                                                    |
| `url`      | `String`            | Complete URL of the page, corresponding to the page's canonical URL, and in case it does not exist, the JavaScript `location.href` is utilized.                                                                                            |

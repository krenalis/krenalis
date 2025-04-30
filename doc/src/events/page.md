{% extends "/layouts/doc.html" %}
{% macro Title string %}Page Call{% end %}
{% Article %}

# Page call

The page call allows you to capture when a user views a page on your website, including any extra details about that specific page.

For tracking app screen views, it's recommended to utilize the specific [screen](screen) call.

## When to use the page call

The page call is used when loading a new page. In Single Page Applications (SPAs), it's also used for logical changes in the page content or transitions, like moving to a different view or route.

## How to make a page call

To make a page call, you can use a Meergo SDK. Refer to its documentation for more details.

For example, with the [JavaScript SDK](/developers/javascript-sdk) in the browser, you can make a page call (apart from the automatic page call triggered by the snippet on page load) in the following way:

```javascript
meergo.page('Product View', {
    productId: 308263,
});
```

The argument, `'Sign Up'`, is the name of the page. The second argument contains the properties of the event, offering additional context to track page views. This extra information enhances the comprehension of your users' actions and is stored along with the event.

The following is an example of how the previous page call would appear in Meergo once received and processed:
```json
{
  "anonymousId": "3a8b2c9f6e107d5e8b1c0f47",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.100 Safari/537.36"
  },
  "messageId": "0a2dd91e-bbac-11e4-8dfc-aa07a5b093db",
  "receivedAt": "2024-01-14T18:07:33.288Z",
  "sentAt": "2024-01-14T18:07:33.051Z",
  "timestamp": "2024-01-14T18:07:33.051Z",
  "properties": {
    "productId": 308263,
    "title": "Acme Sign Up",
    "url": "http://www.example.com"
  },
  "type": "page",
  "version": "1.0"
}
```

As you can see, there is much more information than what is provided in the JavaScript example. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

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


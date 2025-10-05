{% extends "/layouts/doc.html" %}
{% macro Title string %}Track call{% end %}
{% Article %}

# Track

The track call is used to send specific events or actions, and associated properties, that occur when users interact with your application or website.

For more effective tracking of page and screen views, it's recommended to utilize the specific [page](page) and [screen](screen) calls.

## When to use the track call

Here are some common scenarios when you might want to use a track call:

* **Tracking Important Events:** Use track calls to record significant events like page views, button clicks, form submissions, or any other actions you want to monitor.

* **Monitoring User Engagement:** Measure user engagement by recording time spent on an interaction, the number of actions performed, or other behaviors indicating interest or participation.

* **Confirming Conversions:** When a user completes a desired action such as a purchase or registration, send a track call to confirm the conversion and track the success of your initiatives.

* **User Segmentation:** Use track calls to record specific events and create user segments based on their behavior. This helps customize marketing strategies or provide personalized experiences based on user actions.

## How to make a track call

To make a track call, you can use a Meergo SDK. Refer to its documentation for more details. For example, with the [JavaScript SDK](/integrations/sources/javascript-sdk) in the browser, you can make a track call in the following way:

```javascript
meergo.track('Product Viewed', {
    productId: 'ABC123',
    category: 'Electronics'
});
```

The first argument, `'Product Viewed'`, represents the event, which is essentially the user's action. The second argument contains the properties of the event, offering additional context to track events. This extra information enhances the comprehension of your users' actions and is stored along with the event.

The following is an example of how the previous track call would appear in Meergo once received and processed:

```json
{
  "anonymousId": "3a8b2c9f6e107d5e8b1c0f47",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.100 Safari/537.36"
  },
  "messageId": "522cc01f-f4cf-22e7-1e8b-cc81d2c005dd",
  "receivedAt": "2024-01-15T12:23:51.801Z",
  "sentAt": "2024-01-15T12:23:52.061Z",
  "timestamp": "2024-01-15T12:23:52.061Z",
  "properties": {
    "productId": "ABC123",
    "category": "Electronics"
  },
  "type": "track",
  "userId": "59a20n37ec82",
  "version": "1.0"
}
```

As you can see, there is much more information than what is provided in the JavaScript example. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

## Properties

As seen earlier, properties are pieces of information about the user's action that you wish to store along with the event. You can include whatever information you need.

They are accessible in Meergo within a property of the event called `properties` as a JSON value.

### Prefilled properties

Meergo doesn't automatically forward the properties of an event to a destination location. Instead, the control lies in your hands to choose and specify the properties you want to send when configuring the mapping.

However, when you set up a mapping, Meergo automatically fills in the mapping expressions with the following properties when applicable:

| Property   | JSON&nbsp;Type | Description                                                                                                                                                            |
|------------|----------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `currency` | `String`       | Currency for the event's resulting revenue, specified in ISO 4127 format, must be set; otherwise, the revenue is assumed to be in US dollars.                          |
| `revenue`  | `Number`       | Income generated from the event. For instance, a $24.99 shirt would bring in a revenue of `24.99` dollars.                                                             |
| `value`    | `String`       | Abstract value of the event, emphasized when it doesn't directly generate income but is important for marketing goals, like boosting brand visibility on social media. |

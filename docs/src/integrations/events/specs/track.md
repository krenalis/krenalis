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

To make a track call, you can use a Meergo SDK.

<!-- codeblocks sync:sdk Track -->

```javascript
meergo.track('Workout Completed', {
  workout_type: 'Cardio',
  duration_minutes: 45,
  calories_burned: 380,
  device: 'Smartwatch'
});
```

Refer to the [JavaScript SDK](/integrations/sources/javascript-sdk) for more details.

```python
analytics.track(
    user_id="user-123",
    event="Workout Completed",
    properties={
        "workout_type": "Cardio",
        "duration_minutes": 45,
        "calories_burned": 380,
        "device": "Smartwatch",
    },
)
```

Refer to the [Python SDK](/integrations/sources/python) for more details.

```go
client.Enqueue(analytics.Track{
    UserId: "user-123",
    Event:  "Workout Completed",
    Properties: map[string]any{
        "workout_type":     "Cardio",
        "duration_minutes": 45,
        "calories_burned":  380,
        "device":           "Smartwatch",
    },
})
```

Refer to the [Go SDK](/integrations/sources/go) for more details.

```nodejs
analytics.track({
  userId: 'user-123',
  event: 'Workout Completed',
  properties: {
    workout_type: 'Cardio',
    duration_minutes: 45,
    calories_burned: 380,
    device: 'Smartwatch'
  }
});
```

Refer to the [Node.js SDK](/integrations/sources/nodejs) for more details.

```java
analytics.enqueue(TrackMessage.builder("Workout Completed")
    .userId("user-123")
    .properties(new com.meergo.analytics.messages.Properties()
        .putValue("workout_type", "Cardio")
        .putValue("duration_minutes", 45)
        .putValue("calories_burned", 380)
        .putValue("device", "Smartwatch"))
);
```

Refer to the [Java SDK](/integrations/sources/java) for more details.

```csharp
Analytics.Client.Track("user-123", "Workout Completed", new Properties {
    { "workout_type", "Cardio" },
    { "duration_minutes", 45 },
    { "calories_burned", 380 },
    { "device", "Smartwatch" }
});
```

Refer to the [.Net SDK](/integrations/sources/dotnet) for more details.

```kotlin
analytics.track(
    "Workout Completed",
    properties = buildJsonObject {
        put("workout_type", "Cardio")
        put("duration_minutes", 45)
        put("calories_burned", 380)
        put("device", "Smartwatch")
    }
)
```
Refer to the [Android SDK](/integrations/sources/android-sdk) for more details. You can also use the **Java** language with the Android SDK.

<!-- end codeblocks -->

The following is an example of how the previous track call would appear in Meergo once received and processed:

```json
{
  "anonymousId": "f8d886bf-e1a6-484c-9ded-ac789ec4146b",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Macintosh) Chrome/138 Safari/537.36"
  },
  "messageId": "522cc01f-f4cf-22e7-1e8b-cc81d2c005dd",
  "receivedAt": "2025-10-13T12:23:51.801Z",
  "sentAt": "2025-10-13T12:23:52.061Z",
  "timestamp": "2025-10-13T12:23:52.061Z",
  "properties": {
    "workout_type": "Cardio",
    "duration_minutes": 45,
    "calories_burned": 380,
    "device": "Smartwatch"
  },
  "traits": {},
  "type": "track",
  "userId": "user-123"
}
```

As you can see, there is much more information than what is provided in SDK examples. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

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

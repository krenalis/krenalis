{% extends "/layouts/doc.html" %}
{% macro Title string %}Screen call{% end %}
{% Article %}

# Screen

The screen call enables you to capture instances when a user views a screen and record associated properties or details about that particular screen.

For tracking website page views, it's recommended to utilize the specific [page](page) call.

## When to use the screen call

Use the screen call when there is a change in the displayed content or when a user transitions from one screen to another within your application.

## How to make a screen call

To make a screen call, you can use a Meergo SDK.

<!-- codeblocks sync:sdk Screen -->

```javascript
meergo.screen('Workout Summary', {
    workout_type: 'Cardio',
    duration_minutes: 45,
    calories_burned: 380
});
```
Refer to the [JavaScript SDK](/integrations/sources/javascript-sdk) for more details.

```python
analytics.screen('user-123', 'Workout Summary', {
    'workout_type': 'Cardio',
    'duration_minutes': 45,
    'calories_burned': 380
})
```
Refer to the [Python SDK](/integrations/sources/python) for more details.

```go
client.Enqueue(analytics.Screen{
    UserId: "user-123",
    Name:   "Workout Summary",
    Properties: map[string]any{
        "workout_type":     "Cardio",
        "duration_minutes": 45,
        "calories_burned":  380,
    },
})
```
Refer to the [Go SDK](/integrations/sources/go) for more details.

```nodejs
analytics.screen({
    userId: 'user-123',
    name: 'Workout Summary',
    properties: {
        workout_type: 'Cardio',
        duration_minutes: 45,
        calories_burned: 380
    }
});
```
Refer to the [Node.js SDK](/integrations/sources/nodejs) for more details.

```java
analytics.enqueue(ScreenMessage.builder("Workout Summary")
    .userId("user-123")
    .properties(new com.meergo.analytics.messages.Properties()
        .putValue("workout_type", "Cardio")
        .putValue("duration_minutes", 45)
        .putValue("calories_burned", 380))
);
```
Refer to the [Java SDK](/integrations/sources/java) for more details, or use Java with the [Android SDK](/integrations/sources/android-sdk).

```kotlin
analytics.screen(
    "Workout Summary",
    properties = buildJsonObject {
        put("workout_type", "Cardio")
        put("duration_minutes", 45)
        put("calories_burned", 380)
    }
)
```
Refer to the [Android SDK](/integrations/sources/android-sdk) for more details. You can also use the **Java** language with the Android SDK.

```csharp
Analytics.Client.Screen("user-123", "Workout Summary", new Properties {
    { "workout_type", "Cardio" },
    { "duration_minutes", 45 },
    { "calories_burned", 380 }
});
```
Refer to the [.Net SDK](/integrations/sources/dotnet) for more details.

<!-- end codeblocks -->

The following is an example of how a screen call would appear in Meergo once received and processed:

```json
{
  "anonymousId": "f8d886bf-e1a6-484c-9ded-ac789ec4146b",
  "channel": "mobile",
  "context": {
    "ip": "172.16.254.1"
  },
  "messageId": "1b3ff72a-ccbd-22f5-9abc-bb09c6d2a8ef",
  "receivedAt": "2025-10-13T09:03:34.917Z",
  "sentAt": "2025-10-13T09:03:34.781Z",
  "timestamp": "2025-10-13T09:03:34.781Z",
  "name": "Workout Summary",
  "properties": {
    "name": "Workout Summary",
    "workout_type": "Cardio",
    "duration_minutes": 45,
    "calories_burned": 380
  },
  "traits": {},
  "type": "screen",
  "userId": "user-123"
}
```

As you can see, there is much more information than what is provided in SDK examples. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

## Properties

Properties are like extra details that tell you more about the screen. You can include whatever information you need.

They are accessible in Meergo within a property of the event called `properties` as a JSON value.

### Prefilled properties

Meergo doesn't automatically forward an event's properties to a destination. Instead, the control lies in your hands to choose and specify the properties you want to send when configuring the mapping.

However, when you set up a mapping, Meergo automatically fills in the mapping expressions with the following properties when applicable:

| Property   | JSON&nbsp;Type | Description                                        |
|------------|----------------|----------------------------------------------------|
| `name`     | `String`       | Screen name designated as reserved for future use. |

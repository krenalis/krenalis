{% extends "/layouts/doc.html" %}
{% macro Title string %}Android SDK Methods{% end %}
{% Article %}

# Android SDK methods

The Android SDK is equipped to handle all essential event calls, including `screen`, `track`, `identify`, and `group`. Furthermore, it offers functionalities to efficiently manage session information.

With these capabilities, you can seamlessly track and analyze user interactions, facilitating a comprehensive understanding of user behavior and engagement.

Below the Android SDK methods:

- [screen](#screen)
- [track](#track)
- [identify](#identify)
- [group](#group)
- [userId](#userid)
- [traits](#traits)
- [alias](#alias)
- [anonymousId](#anonymousid)
- [reset](#reset)
- [getSessionId](#getsessionid)
- [startSession](#startsession)
- [endSession](#endsession)

## screen

The screen method implements the [screen call](../../events/screen).

The screen call enables you to capture instances when a user views a screen and record associated properties or details about that particular screen.

#### Syntax

```kotlin
screen(title: String, properties: JsonObject = emptyJsonObject, category: String = ""): Unit
```

#### Parameters

| Name         | Type         | Required                               | Description                      |
|--------------|--------------|----------------------------------------|----------------------------------|
| `title`      | `String`     | <div style="text-align:center">✓</div> | Title of the screen.             |
| `properties` | `JsonObject` |                                        | Properties of the screen.        |
| `category`   | `String`     |                                        | Category to describe the screen. |

#### Example

Kotlin
```kotlin
analytics.screen("Order completed", buildJsonObject {
    put("items", 3)
    put("total", 274.99)
})
```

Java
```java
analytics.screen("Order completed", Builders.buildJsonObject(o -> {
    o.put("items", 3)
      .put("total", 274.99);
}));
```

## track

The track method implements the [track call](../../events/track).

The track call is used to send specific events or actions, and associated properties, that occur when users interact with your application or website.

#### Syntax

```kotlin
track(name: String, properties: JsonObject = emptyJsonObject): Unit
```

#### Parameters

| Name         | Type         | Required                               | Description              |
|--------------|--------------|----------------------------------------|--------------------------|
| `name`       | `String`     | <div style="text-align:center">✓</div> | Name of the event.       |
| `properties` | `JsonObject` |                                        | Properties of the event. |

#### Example

Kotlin
```kotlin
analytics.track("Product added to cart", buildJsonObject {
    put("id", 12345)
})
```

Java
```java
analytics.track("Product added to cart", Builders.buildJsonObject(o -> {
    o.put("id", 12345);
}));
```

## identify

The identify method implements the [identify call](../../events/identify).

Through an identify call, you can connect previous and upcoming events to a recognized user and save details about them along with their events, such as name and email. The user information can also be utilized to update and enhance unified data from other sources.

#### Syntax

```kotlin
identify(userId: String, traits: JsonObject = emptyJsonObject): Unit
```
#### Parameters

| Name         | Type         | Required                               | Description                         |
|--------------|--------------|----------------------------------------|-------------------------------------|
| `userId`     | `String`     | <div style="text-align:center">✓</div> | Identifier of the user.             |
| `traits`     | `JsonObject` |                                        | Traits to add to the user's traits. |

#### Example

Kotlin
```kotlin
analytics.identify("59a20n37ec82", buildJsonObject {
    put("firstName", "Emily")
    put("lastName", "Johnson")
    put("email", "emma.johnson@example.com")
})
```

Java
```java
analytics.identify("59a20n37ec82", Builders.buildJsonObject(o -> {
    o.put("firstName", "Emily")
      .put("lastName", "Johnson")
      .put("email", "emma.johnson@example.com");
}));
```

## group

The `group` method implements the [group call](../../events/group). 

The group call provides a way to associate individual users with groups, such as a company, organization, team, association, or initiative. A user who has been identified can be associated with several groups.

#### Syntax

```kotlin
group(groupId: String, traits: JsonObject = emptyJsonObject): Unit
```

#### Parameters

| Name         | Type         | Required                               | Description              |
|--------------|--------------|----------------------------------------|--------------------------|
| `groupId`    | `String`     | <div style="text-align:center">✓</div> | Identifier of the group. |
| `traits`     | `JsonObject` |                                        | Traits of the group.     |

#### Example

Kotlin
```kotlin
analytics.group("84s76y49tb28v1jxq", buildJsonObject {
    put("name", "AcmeTech")
    put("industry", "Technology")
    put("employeeCount", 100)
})
```

Java
```java
analytics.group("84s76y49tb28v1jxq", Builders.buildJsonObject(o -> {
    o.put("name", "AcmeTech")
      .put("industry", "Technology")
      .put("employeeCount", 100);
}));
```

## userId

The `userId` method is used to get the identifier of the user. It always returns the user's identifier, or `null` if there is no identifier.

To modify the user's identifier, use the [`identify`](#identify) method or the [`reset`](#reset) method.

#### Syntax

```kotlin
userId(): String?
```

#### Parameters

There are no parameters.

#### Examples

Kotlin
```kotlin
val userId = analytics.userId()
```

Java
```java
String userId = analytics.userId();
```

## traits

The `traits` method is used to retrieve a user's traits. These traits are for the anonymous user if the user is anonymous, and for the non-anonymous user if non-anonymous.

To modify the user's traits, use the [`identify`](methods#identify) method or the [`reset`](methods#reset) method.

#### Syntax

```kotlin
traits(): jsonObject?
```

#### Parameters

There are no parameters.

#### Examples

Kotlin
```kotlin
val traits = analytics.traits()
```

Java
```java
JsonObject traits = analytics.traits();
```

## alias

The `alias` method is used to merge two user identities, effectively connecting two sets of user data as one. This method is applicable when the event is dispatched to a destination, such as Mixpanel.  

> In Meergo, user merging is handled by Meergo's Identity Resolution. Therefore this method is not utilized in this process. 

#### Syntax

```kotlin
alias(newId: String): Unit
```

#### Parameters

| Name         | Type     | Required                               | Description                                      |
|--------------|----------|----------------------------------------|--------------------------------------------------|
| `newId`      | `String` | <div style="text-align:center">✓</div> | The new ID you want to alias the existing ID to. |

#### Example

Kotlin
```kotlin
analytics.alias("12r60m18ff04")
```

Java
```java
analytics.alias("12r60m18ff04");
```

## anonymousId

The `anonymousId` method is used to retrieve the Anonymous ID.

To modify the Anonymous ID, use the [`identify`](methods#identify) method or the [`reset`](methods#reset) method.

#### Syntax

```kotlin
anonymousId(): String
```

#### Parameters

There are no parameters.

#### Examples

Kotlin
```kotlin
val anonymousId = analytics.anonymousId()
```

Java
```java
String anonymousId = analytics.anonymousId();
```

## reset

The `reset` method resets the user identifier, and updates or removes the Anonymous ID and traits according to the strategy (as detailed in the table below). If `all` is true it always resets the Anonymous ID by generating a new one, and ends the session if one exists, regardless of the strategy.

| Strategy     | Behavior of `reset()`                                                                                                          |
|--------------|--------------------------------------------------------------------------------------------------------------------------------|
| Conversion   | Removes User ID and user traits, and changes Anonymous ID and session.                                                         |
| Fusion       | Removes User ID and user traits. Does not change Anonymous ID or session.                                                      |
| Isolation    | Removes User ID and user traits and changes Anonymous ID and session.                                                          |
| Preservation | Removes User ID. Restores Anonymous ID, user traits and session to their state before the latest [`identify`](#identify) call. |


#### Syntax

```kotlin
reset(all: Boolean = false): Unit
```

#### Parameters

| Name       | Type      | Required | Description                                                                              |
|------------|-----------|----------|------------------------------------------------------------------------------------------|
| `all`      | `Boolean` |          | Indicates if the Anonymous ID and the session must be reset, regardless of the strategy. |

#### Example

Kotlin
```kotlin
analytics.reset()
```

Java
```java
analytics.reset();
```

#### Segment Compatibility

To align with Segment's `reset()` behavior, choose the "Conversion" or "Isolation" strategy in Meergo. Note that `reset(true)` is not available in Segment.

#### RudderStack Compatibility

To match RudderStack's `reset()` behavior, choose the "Conversion" or "Isolation" strategy in Meergo. In Meergo, `reset(true)` works the same way as it does in RudderStack for all strategies.

## getSessionId

The `getSessionId` method returns the current session identifier. It returns `null` if there is no session.

#### Syntax

```kotlin
getSessionId(): Long?
```

#### Parameters

There are no parameters.

#### Example

Kotlin
```kotlin
val sessionId = analytics.getSessionId()
```

Java
```java
long sessionId = analytics.getSessionId();
```

## startSession

The `startSession` method starts a new session using the provided identifier. If the identifier provided is `null`, it generates one automatically. The provided session Id, if not `null`, must be a positive Long.

#### Syntax

```kotlin
startSession(id: Long?): Unit
```

#### Parameters

| Name | Type    | Required                               | Description                           |
|------|---------|----------------------------------------|---------------------------------------|
| `id` | `Long?` | <div style="text-align:center">✓</div> | Session identifier. Must be positive. |

#### Example

Kotlin
```kotlin
analytics.startSession(123456789L)
```

Java
```java
analytics.startSession(123456789L);
```

## endSession

The `endSession` method ends the session.

#### Syntax

```kotlin
endSession(): Unit
```

#### Parameters

There are no parameters.

#### Example

Kotlin
```kotlin
analytics.endSession()
```

Java
```java
analytics.endSession();
```

 # UserInfo Class

The `UserInfo` class represents a user.

## userId

The `userId` method is used to get the identifier of the user. It always returns the user's identifier, or `null` if there is no identifier.

To modify the user's identifier, use the [`identify`](methods.md#identify) method or the [`reset`](methods.md#reset) method.

#### Syntax

```kotlin
userId(): String?
```

#### Examples

Kotlin
```kotlin
val userId = analytics.userId()
```

Java
```java
String userId = analytics.userId();
```

## anonymousId

The `anonymousId` method is used to retrieve the Anonymous ID.

To modify the Anonymous ID, use the [`identify`](methods.md#identify) method or the [`reset`](methods.md#reset) method.

#### Syntax

```kotlin
anonymousId(): String
```

#### Examples

Kotlin
```kotlin
val anonymousId = analytics.anonymousId()
```

Java
```java
String anonymousId = analytics.anonymousId();
```

## traits

The `traits` method is used to retrieve a user's traits. These traits are for the anonymous user if the user is anonymous, and for the non-anonymous user if non-anonymous.

To modify the user's traits, use the [`identify`](methods.md#identify) method or the [`reset`](methods.md#reset) method.

#### Syntax

```kotlin
traits(): jsonObject?
```

#### Examples

Kotlin
```kotlin
val traits = analytics.traits()
```

Java
```java
JsonObject traits = analytics.traits();
```

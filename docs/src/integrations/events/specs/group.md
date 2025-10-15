{% extends "/layouts/doc.html" %}
{% macro Title string %}Group call{% end %}
{% Article %}

# Group

The group call provides a way to associate individual users with groups, such as a company, organization, team, association, or initiative. A user who has been identified can be associated with several groups.

It also provides the ability to store custom traits related to that group, such as organization name and industry sector, or account type and annual revenue.

## When to use the group call

For best results, it is recommended to make a group call in the following scenarios:

* When a new user signs up or onboards to your website or application.
* Whenever there are changes in group-specific traits (e.g., industry, employee count).
* In situations where users can dynamically switch between different groups during their session.

## How to make a group call

To make a group call, you can use a Meergo SDK.

<!-- codeblocks sync:sdk Group -->

```javascript
meergo.group('group-890', {
    name: 'AcmeTech',
    industry: 'Technology',
    employeeCount: 100
});
```
Refer to the [JavaScript SDK](/integrations/sources/javascript-sdk) for more details.

```python
analytics.group(
    user_id="user-123",
    group_id="group-890",
    traits={
        "name": "AcmeTech",
        "industry": "Technology",
        "employeeCount": 100,
    },
)
```
Refer to the [Python SDK](/integrations/sources/python) for more details.

```go
client.Enqueue(analytics.Group{
    UserId: "user-123",
    GroupId: "group-890",
    Traits: map[string]any{
        "name":          "AcmeTech",
        "industry":      "Technology",
        "employeeCount": 100,
    },
})
```
Refer to the [Go SDK](/integrations/sources/go) for more details.

```nodejs
analytics.group({
    userId: 'user-123',
    groupId: 'group-890',
    traits: {
        name: 'AcmeTech',
        industry: 'Technology',
        employeeCount: 100
    }
});
```
Refer to the [Node.js SDK](/integrations/sources/nodejs) for more details.

```java
analytics.enqueue(GroupMessage.builder("group-890")
    .userId("user-123")
    .traits(new com.meergo.analytics.messages.Properties()
        .putValue("name", "AcmeTech")
        .putValue("industry", "Technology")
        .putValue("employeeCount", 100))
);
```
Refer to the [Java SDK](/integrations/sources/java) for more details.

```csharp
Analytics.Client.Group("user-123", "group-890", new Properties {
    { "name", "AcmeTech" },
    { "industry", "Technology" },
    { "employeeCount", 100 }
});
```
Refer to the [.Net SDK](/integrations/sources/dotnet) for more details.

```kotlin
analytics.group(
    userId = "user-123",
    groupId = "group-890",
    traits = buildJsonObject {
        put("name", "AcmeTech")
        put("industry", "Technology")
        put("employeeCount", 100)
    }
)
```

Refer to the [Android SDK](/integrations/sources/android-sdk) for more details. You can also use the **Java** language with the Android SDK.

<!-- end codeblocks -->

The following is an example of how the previous group call would appear in Meergo once received and processed:

```json
{
  "connectionId": 129402661,
  "anonymousId": "f8d886bf-e1a6-484c-9ded-ac789ec4146b",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Macintosh) Chrome/138 Safari/537.36"
  },
  "messageId": "0a2ef1d3-ebad-41b2-9c8a-7b58d8e1a8f9",
  "receivedAt": "2025-10-13T09:42:51.477Z",
  "sentAt": "2025-10-13T09:42:51.210Z",
  "timestamp": "2025-10-13T09:42:51.210Z",
  "traits": {
    "name": "AcmeTech",
    "industry": "Technology",
    "employeeCount": 100
  },
  "type": "group",
  "userId": "user-123",
  "groupId": "group-890"
}
```

As you can see, there is much more information than what is provided in SDK examples. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

## Traits

As seen earlier, traits are pieces of information about the event's group that you wish to store along with the event. You can include whatever information you need.

They are accessible in Meergo within a property of the event called `traits` as a JSON value.

### Prefilled traits

Meergo doesn't automatically forward a group's traits to a destination. Instead, the control lies in your hands to choose and specify the traits you want to send when configuring the mapping.

However, when you set up a mapping, Meergo automatically fills in the mapping expressions with the following traits when applicable:

| Trait         | JSON&nbsp;Type | Description                                                                                                          |
|---------------|----------------|----------------------------------------------------------------------------------------------------------------------|
| `address`     | `Object`       | Address. Can include the following `String` fields `"street"`, `"city"`, `"state"`, `"postalCode"`, and `"country"`. |
| `avatar`      | `String`       | URL of the avatar image.                                                                                             |
| `createdAt`   | `String`       | Date of account creation in the ISO 8601 format.                                                                     |
| `description` | `String`       | Description of the group.                                                                                            |
| `email`       | `String`       | Email address of the group.                                                                                          |
| `employees`   | `String`       | Number of employees, commonly referred to when the group is a company.                                               |
| `id`          | `String`       | Unique identifier of the group.                                                                                      |
| `industry`    | `String`       | industry where the group belongs or where the user works.                                                            |
| `name`        | `String`       | Name of the group.                                                                                                   |
| `phone`       | `String`       | Phone number.                                                                                                        | 
| `website`     | `String`       | URL of the group's website.                                                                                          |
| `plan`        | `String`       | Plan that the group is enrolled in.                                                                                  |

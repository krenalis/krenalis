{% extends "/layouts/doc.html" %}
{% macro Title string %}Group Call{% end %}
{% Article %}

# Group call

The group call provides a way to associate individual users with groups, such as a company, organization, team, association, or initiative. A user who has been identified can be associated with several groups.

It also provides the ability to store custom traits related to that group, such as organization name and industry sector, or account type and annual revenue.

## When to use the group call

For best results, it is recommended to make a group call in the following scenarios:

* When a new user signs up or onboards to your website or application.

* Whenever there are changes in group-specific traits (e.g., industry, employee count).
 
* In situations where users can dynamically switch between different groups during their session.

## How to make a group call

To make a group call, you can use a Meergo SDK. Refer to its documentation for more details. For example, with the [JavaScript SDK](../javascript-sdk) in the browser, you can make a group call in the following way:

```javascript
meergo.group('84s76y49tb28v1jxq', {
	name: "AcmeTech",
	industry: "Technology",
	employeeCount: 100
});
```

The first argument, `'84s76y49tb28v1jxq'`, is the **Group ID**, which uniquely identifies the group in your website's database. The second argument consists of the **group's traits**, which are pieces of information you want to store with the event.

The following is an example of how the previous group call would appear in Meergo once received and processed:
```json
{
  "anonymousId": "3a8b2c9f6e107d5e8b1c0f47",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.100 Safari/537.36"
  },
  "messageId": "0a2ef1d3-ebad-41b2-9c8a-7b58d8e1a8f9",
  "receivedAt": "2024-01-16T09:42:51.477Z",
  "sentAt": "2024-01-16T09:42:51.210Z",
  "timestamp": "2024-01-16T09:42:51.210Z",
  "traits": {
    "name": "AcmeTech",
    "industry": "Technology",
    "employeeCount": 100
  },
  "type": "group",
  "groupId": "84s76y49tb28v1jxq",
  "version": "1.0"
}
```

As you can see, there is much more information than what is provided in the JavaScript example. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

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

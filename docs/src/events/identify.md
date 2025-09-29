{% extends "/layouts/doc.html" %}
{% macro Title string %}Identify Call{% end %}
{% Article %}

# Identify call

Through an identify call, you can connect previous and upcoming events to a recognized user and save details about them along with their events, such as name and email. The user information can also be utilized to update and enhance unified data from other sources.

## When to use the identify call

For optimal use, Meergo suggests making an identify call in the following situations:

* Right after a user registers for the first time.

* Once a user successfully logs in.

* When there's an update to the user's information, such as a change in address or the addition of a new one.

## How to make an identify call

To make an identify call, you can use a Meergo SDK. Refer to its documentation for more details. For example, with the [JavaScript SDK](/developers/javascript-sdk) in the browser, you can make an identify call in the following way:

```javascript
meergo.identify('59a20n37ec82', {
    firstName: 'Emily',
    lastName: 'Johnson',
    email: 'emma.johnson@example.com',
    address: {
        street: "123 Main Street",
        city: "San Francisco",
        state: "CA",
        postalCode: "94104",
        country: "USA"
    }
});
```

The first argument, `'59a20n37ec82'`, is the **User ID**, which uniquely identifies the user in your website's database. The second argument consists of the **user's traits**, which are pieces of information you want to store with the event and potentially unify with customer data taken from other sources.

The following is an example of how the previous identify call would appear in Meergo once received and processed:

```json
{
  "anonymousId": "3a8b2c9f6e107d5e8b1c0f47",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.100 Safari/537.36"
  },
  "messageId": "022bb90c-bbac-11e4-8dfc-aa07a5b093db",
  "receivedAt": "2024-01-15T11:54:08.391Z",
  "sentAt": "2024-01-15T11:54:08.206Z",
  "timestamp": "2024-01-15T11:54:08.206Z",
  "traits": {
    "firstName": "Emily",
    "lastName": "Johnson",
    "email": "emma.johnson@example.com",
    "address": {
      "street": "123 Main Street",
      "city": "San Francisco",
      "state": "CA",
      "postalCode": "94104",
      "country": "USA"
    }
  },
  "type": "identify",
  "userId": "59a20n37ec82",
  "version": "1.0"
}
```

As you can see, there is much more information than what is provided in the JavaScript example. This is because both the SDK used to make the call and Meergo enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

## Traits

As seen earlier, traits are pieces of information about the event's user that you wish to store along with the event, and possibly integrate with customer data obtained from other sources. You can include whatever information you need.

The traits can be passed to Meergo through the identify and track calls. They are accessible in Meergo within a property of the event called `traits` as a JSON value.

### Prefilled traits

Meergo doesn't automatically forward a user's traits to a destination. Instead, the control lies in your hands to choose and specify the traits you want to send when configuring the mapping.

However, when you set up a mapping, Meergo automatically fills in the mapping expressions with the following traits when applicable:

| Trait         | JSON&nbsp;Type | Description                                                                                                                                                                                    |
|---------------|----------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `address`     | `Object`       | Address. Can include the following `String` fields `"street"`, `"city"`, `"state"`, `"postalCode"`, and `"country"`.                                                                           |
| `age`         | `Number`       | Age.                                                                                                                                                                                           |
| `avatar`      | `String`       | URL of the avatar image.                                                                                                                                                                       |
| `birthday`    | `String`       | Birthday in the ISO 8601 format.                                                                                                                                                               |
| `company`     | `Object`       | Company. Can include the following fields: `"id"` as a `String` or `Number`, `"name"` as a `String`, `"industry"` as a `String`, `"employee_count"` as a `Number`, and `"plan"` as a `String`. |
| `createdAt`   | `String`       | Date of account creation in the ISO 8601 format.                                                                                                                                               |
| `description` | `String`       | Description of the user.                                                                                                                                                                       |
| `email`       | `String`       | Email address.                                                                                                                                                                                 |
| `firstName`   | `String`       | First name.                                                                                                                                                                                    |
| `gender`      | `String`       | Gender.                                                                                                                                                                                        |
| `id`          | `String`       | Unique identifier of the user.                                                                                                                                                                 |
| `lastName`    | `String`       | Last name.                                                                                                                                                                                     |
| `name`        | `String`       | Full name. It is automatically filled from `firstName` and `lastName` if they are present.                                                                                                     |
| `phone`       | `String`       | Phone number.                                                                                                                                                                                  |
| `title`       | `String`       | Title. For example, concerning her role within the company.                                                                                                                                    |
| `username`    | `String`       | Username. Unique for each user.                                                                                                                                                                |
| `website`     | `String`       | URL of the user's website.                                                                                                                                                                     |

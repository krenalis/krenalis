# Anonymize Call

The anonymize call serves the purpose of anonymizing a previously identified user. By invoking this function, traits associated with the identified individual, such as name and email, are removed. This action ensures a user's transition from a known, identified state to an anonymous one, allowing for privacy considerations and data protection.

## When to Use the Anonymize Call

To safeguard user privacy and adhere to data protection regulations, it is advisable to invoke the anonymize call in the following scenarios:

* When the user logs out.

* In the event of the user deleting their registration.

## How to Make an Anonymize Call

To make an anonymize call, you can use a Chichi SDK. Refer to its documentation for more details. For example, with the [JavaScript SDK](../javascript-sdk.md) in the browser, you can make an anonymize call in the following way:

```javascript
chichiAnalytics.anonymize();
```

The following is an example of how the previous anonymize call would appear in Chichi once received and processed:

```json
{
  "anonymousId": "3a8b2c9f6e107d5e8b1c0f47",
  "channel": "browser",
  "context": {
    "ip": "172.16.254.1",
    "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.100 Safari/537.36"
  },
  "messageId": "686bc063-6ec9-40f1-a16d-64b20a920485",
  "receivedAt": "2024-01-15T13:12:49.214Z",
  "sentAt": "2024-01-15T13:12:49.015Z",
  "timestamp": "2024-01-15T13:12:49.015Z",
  "type": "anonymize",
  "version": "1.0"
}
```

As you can see, there is much more information than what is provided in the JavaScript example. This is because both the SDK used to make the call and Chichi enrich the information by extracting it from the device where the event occurs. Refer to the SDK documentation for more details.

## Reset vs Anonymize 

The `reset` function shares similarities with the `anonymize` call, but there is a key difference – it doesn't generate an event. Additionally, it uniformly removes the Anonymous ID and ends the current session if one exists. While the `anonymize` call performs these actions exclusively for the "AB-C," "A-B-C," and "AC-B" strategies, for the "ABC" strategy, it keeps the Anonymous ID and the session.
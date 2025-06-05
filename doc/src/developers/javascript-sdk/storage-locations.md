{% extends "/layouts/doc.html" %}
{% macro Title string %}Storage Locations{% end %}
{% Article %}

# Storage locations

The JavaScript SDK is designed to store user data directly within the user's browser. Through the [`storage`](options#storage-option) option, you can control which storage locations the data will be saved to, or choose not to save them at all. This capability is particularly important in ensuring compliance with privacy regulations, as selecting the appropriate storage method can significantly impact the level of user data protection and privacy afforded by the application.

Below are the browser storage locations that the JavaScript SDK can use to store data:

- cookie
- localStorage
- sessionStorage
- memory

And below are the types of information that the JavaScript SDK stores in these storage locations:

- Anonymous ID
- Session
- User ID
- User traits
- Group ID
- Group traits
- Leader information
- Event queues

### Data encoding

Data is encoded in Base64, unless otherwise specified. Notably, in Internet Explorer 11, data is encoded in Base64 starting from UTF-16 rather than UTF-8, distinguished by an underscore ('_') prefix.

## cookie

Below are the cookies that the SDK stores in the user browser.

All cookie names are prefixed with "`meergo.<writeKey>.`" where `<writeKey>` represents the first seven characters of the Write Key.

| Name          | Description                         |
|---------------|-------------------------------------|
| `anonymousId` | Anonymous ID.                       |
| `groupId`     | Group ID.                           |
| `groupTraits` | Group traits.                       |
| `session`     | Session ID and its expiration time. |
| `userId`      | User ID.                            |
| `userTraits`  | User traits.                        |

Using the Write Key in the cookie names lets you use two different Write Keys at the same time on one page. For instance, if the Write Key is "`z43tavAOsBB8RY50nAtItXMMIipGKEOC`", the cookie names would be "`meergo.z43tavA.anonymousId`", "`meergo.z43tavA.groupId`", and so on.

When the SDK persists user data in cookies, you can use the [`cookie`](options#cookie-option) option to control some specific settings.

## localStorage

Below are the items that the SDK stores in the localStorage of the browser.

All keys are prefixed with "`meergo.<writeKey>.`" where `<writeKey>` represents the first seven characters of the Write Key.

| Key               | Description                                                                                                      |
|-------------------|------------------------------------------------------------------------------------------------------------------|
| `anonymousId`     | Anonymous ID.                                                                                                    |
| `groupId`         | Group ID.                                                                                                        |
| `groupTraits`     | Group traits.                                                                                                    |
| `leader.beat`     | Browser tab designated as the leader, along with its expiration time, is updated by the tab leader every second. |
| `leader.election` | A browser tab that participated in a leader election.                                                            |
| `session`         | Session ID and its expiration time.                                                                              |
| `userId`          | User ID.                                                                                                         |
| `userTraits`      | User traits.                                                                                                     |
| `<tabId>.queue`   | Tracks the events in the queue of the tab with identifier `<tabId>`. `<tabId>` is a UUID v4.                     |

The values of the leader keys are not Base64 encoded.

## sessionStorage

Below are the items that the SDK stores in the sessionStorage of the browser.

All keys are prefixed with "`meergo.<writeKey>.`" where `<writeKey>` represents the first seven characters of the Write Key.

| Key           | Description                         |
|---------------|-------------------------------------|
| `anonymousId` | Anonymous ID.                       |
| `groupId`     | Group ID.                           |
| `groupTraits` | Group traits.                       |
| `session`     | Session ID and its expiration time. |
| `userId`      | User ID.                            |
| `userTraits`  | User traits.                        |

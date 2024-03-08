# Data Storages

The JavaScript SDK is designed to store user data directly within the user's browser. Through the [`store.type`](options.md#storagetype-option) option, you can control which storages the data will be saved to, or choose not to save them at all. This capability is particularly important in ensuring compliance with privacy regulations, as selecting the appropriate storage method can significantly impact the level of user data protection and privacy afforded by the application.

Below are the browser storages that the JavaScript SDK can use to store user data:

- cookies
- localStorage
- sessionStorage
- memory

And below are the types of information that the JavaScript SDK stores in these storages:

- Anonymous ID
- Session
- User ID
- User traits
- Group ID
- Group traits
- Leader information
- Event queues

## cookies

Below are the cookies that the SDK stores in the user browser.

All cookie names are prefixed with "`chichi.<writeKey>.`" where `<writeKey>` represents the first seven characters of the Write Key.

| Name          | Description                         |
|---------------|-------------------------------------|
| `anonymousId` | Anonymous ID.                       |
| `groupId`     | Group ID.                           |
| `groupTraits` | Group traits.                       |
| `session`     | Session Id and its expiration time. |
| `userId`      | User ID.                            |
| `userTraits`  | User traits.                        |

Using the Write Key in the cookie names lets you use two different Write Keys at the same time on one page. For instance, if the Write Key is `z43tavAOsBB8RY50nAtItXMMIipGKEOC`, the cookie names would be `chichi.z43tavA.anonymousId`, `chichi.z43tavA.groupId`, and so on. Cookie values are Base64 encoded.

When the SDK persists user data in cookies, you can use the [`store.cookie`](options.md#storagecookie-option) option to control some specific settings.

## localStorage

Below are the items that the SDK stores in the localStorage of the browser.

All keys are prefixed with "`chichi.<writeKey>.`" where `<writeKey>` represents the first seven characters of the Write Key.

| Key               | Description                                                                                                      |
|-------------------|------------------------------------------------------------------------------------------------------------------|
| `anonymousId`     | Anonymous ID.                                                                                                    |
| `groupId`         | Group ID.                                                                                                        |
| `groupTraits`     | Group traits.                                                                                                    |
| `leader.beat`     | Browser tab designated as the leader, along with its expiration time, is updated by the tab leader every second. |
| `leader.election` | A browser tab that participated in a leader election.                                                            |
| `session`         | Session Id and its expiration time.                                                                              |
| `userId`          | User ID.                                                                                                         |
| `userTraits`      | User traits.                                                                                                     |
| `<tabId>.queue`   | Tracks the events in the queue of the tab with identifier `<tabId>`. `<tabId>` is a UUID v4.                     |

Apart from the leader keys, the values are Base64 encoded.

## sessionStorage

Below are the items that the SDK stores in the sessionStorage of the browser.

All keys are prefixed with "`chichi.<writeKey>.`" where `<writeKey>` represents the first seven characters of the Write Key.

| Key           | Description                         |
|---------------|-------------------------------------|
| `anonymousId` | Anonymous ID.                       |
| `groupId`     | Group ID.                           |
| `groupTraits` | Group traits.                       |
| `session`     | Session Id and its expiration time. |
| `userId`      | User ID.                            |
| `userTraits`  | User traits.                        |

The values are Base64 encoded.
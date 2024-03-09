# JavaScript SDK Methods

The JavaScript SDK is equipped to handle all essential event calls, including `page`, `screen`, `track`, `identify`, `anonymize`, and `group`. Furthermore, it offers functionalities to efficiently manage sessions and user/group information.

With these capabilities, you can seamlessly track and analyze user interactions across different platforms, facilitating a comprehensive understanding of user behavior and engagement.

Below the javaScript SDK methods:

- [page](#page)

- [screen](#screen)
 
- [track](#track)

- [identify](#identify)

- [anonymize](#anonymize)

- [group](#group)

- [user](#user)

- [getSessionId](#getsessionid)

- [startSession](#startsession)

- [endSession](#endsession)

- [ready](#ready)

- [reset](#reset)

- [debug](#debug)

- [close](#close)

> The JavaScript SDK also supports the `getAnonymousId` and `setAnonymousId` methods of the RudderStack SDK that can be used as an alternative to the [`user().anonymousId`](user-class.html#anonymousid) method.

## page

The page method implements the [page call](../events/page.md).

The page call allows you to capture when a user views a page on your website, including any extra details about that specific page.

#### Syntax

```javascript
page(category, name, properties, options, callback)
```

#### Parameters 

All parameters are optional. A single `String` parameter indicates `name`, while a single `Object` parameter signifies `properties`. 

| Name         | Type       | Required | Description                                         |
|--------------|------------|----------|-----------------------------------------------------|
| `category`   | `String`   |          | Category of the page. Can be useful to group pages. |
| `name`       | `String`   |          | Name of the page.                                   |
| `properties` | `Object`   |          | Properties of the page.                             |
| `options`    | `Object`   |          | Options of the page.                                |
| `callback`   | `Function` |          | A function called when the event has been queued.   |

It returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`. 

#### Example

```javascript
chichianalytics.page('Shirt', {
    productId: 308263,
}).then(() => console.log('event queued'));
```

## screen

The screen method implements the [screen call](../events/screen.md).

The screen call enables you to capture instances when a user views a screen and record associated properties or details about that particular screen.

#### Syntax

```javascript
screen(category, name, properties, options, callback)
```

#### Parameters

All parameters are optional. A single `String` parameter indicates `name`, while a single `Object` parameter signifies `properties`.

| Name         | Type       | Required | Description                                             |
|--------------|------------|----------|---------------------------------------------------------|
| `category`   | `String`   |          | Category of the screen. Can be useful to group screens. |
| `name`       | `String`   |          | Name of the screen.                                     |
| `properties` | `Object`   |          | Properties of the screen.                               |
| `options`    | `Object`   |          | Options of the screen.                                  |
| `callback`   | `Function` |          | A function called when the event has been queued.       |

It returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

```javascript
chichianalytics.screen('Order Completed', {
	items: 3,
    total: 274.99,
}).then(() => console.log('event queued'));
```

## track

The track method implements the [track call](../events/track.md).

The track call is used to send specific events or actions, and associated properties, that occur when users interact with your application or website.

#### Syntax

```javascript
track(name, properties, options, callback)
```

#### Parameters

Only `name` is required, the other parameters are optional. A single `Object` parameter signifies `properties`.

| Name         | Type       | Required                               | Description                                       |
|--------------|------------|----------------------------------------|---------------------------------------------------|
| `name`       | `String`   | <div style="text-align:center">✓</div> | Name of the event.                                |
| `properties` | `Object`   |                                        | Properties of the event.                          |
| `options`    | `Object`   |                                        | Options of the event.                             |
| `callback`   | `Function` |                                        | A function called when the event has been queued. |

It returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

```javascript
chichianalytics.screen('Order Completed', {
	items: 3,
	total: 274.99,
});
```

## identify

The identify method implements the [identify call](../events/identify.md).

Through an identify call, you can connect previous and upcoming events to a recognized user and save details about them along with their events, such as name and email. The user information can also be utilized to update and enhance unified data from other sources.

#### Syntax

```javascript
identify(userId, traits, options, callback)
```

#### Parameters

All parameters are optional. A single `Object` parameter signifies `traits`.

| Name       | Type       | Required | Description                                       |
|------------|------------|----------|---------------------------------------------------|
| `userId`   | `String`   |          | Identifier of the user.                           |
| `traits`   | `Object`   |          | Traits of the user.                               |
| `options`  | `Object`   |          | Options of the event.                             |
| `callback` | `Function` |          | A function called when the event has been queued. |

It returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

```javascript
identify('59a20n37ec82', {
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

## anonymize

The anonymize method implements the [anonymize call](../events/anonymize.md).

The anonymize call serves the purpose of anonymizing a previously identified user. By invoking this function, traits associated with the identified individual, such as name and email, are removed. This action ensures a user's transition from a known, identified state to an anonymous one, allowing for privacy considerations and data protection.

#### Syntax

```javascript
anonymize()
```

#### Parameters

There is no parameters.

It returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

```javascript
anonymize().then(() => console.log('user has been anonymized'));
```

## group

The `group` method, when called without arguments, returns an instance of the [Group](group-class.md) class representing the group.

When called with arguments, implements the [group call](../events/group.md). The group call provides a way to associate individual users with groups, such as a company, organization, team, association, or initiative. A user who has been identified can be associated with several groups.

#### Syntax

```javascript
// returns a Group instance.
group()
```

```javascript
// implements the group call.
group(groupId, traits, options, callback)
```

#### Parameters

All parameters are optional. A single `Object` parameter signifies `traits`.

| Name       | Type       | Required | Description                                       |
|------------|------------|----------|---------------------------------------------------|
| `groupId`  | `String`   |          | Identifier of the group.                          |
| `traits`   | `Object`   |          | Traits of the group.                              |
| `options`  | `Object`   |          | Options of the event.                             |
| `callback` | `Function` |          | A function called when the event has been queued. |

If no arguments are provided, it returns an instance of the [Group](#group-class) class. Otherwise, it returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

```javascript
const groupId = chichianalytics.group().id();
```

```javascript
chichianalytics.group('84s76y49tb28v1jxq', {
	name: "AcmeTech",
	industry: "Technology",
	employeeCount: 100
});
```

## user

The `user` method returns an instance of the [`User`](user-class.md) class to read and set the identifier, Anonymous ID, and traits of the user.

#### Syntax

```javascript
user()
```

#### Parameters

There are no parameters.

Returns an instance of the [`User`](#user-class) class representing the user.

#### Example

```javascript
const traits = chichianalytics.user().traits();
```

## getSessionId

The `getSessionId` method returns the current session identifier.

#### Syntax

```javascript
getSessionId()
```

#### Parameters

There are no parameters. Returns a `Number` representing the current session identifier, or `null` if there is no session.  

#### Example

```javascript
const sessionId = chichianalytics.getSessionId();
```

## startSession

The `startSession` method starts a new session using the provided identifier. If no identifier is provided, it generates one automatically. 

#### Syntax

```javascript
startSession(id)
```

#### Parameters

| Name | Type       | Required | Description         |
|------|------------|----------|---------------------|
| `id` | `String`   |          | Session identifier. |

#### Example

```javascript
chichianalytics.startSession();
```

## endSession

The `endSession` method ends the session.

#### Syntax

```javascript
endSession()
```

#### Parameters

There are no parameters.

#### Example

```javascript
chichianalytics.endSession();
```

## ready

The `ready` method calls a callback after the Analytics finishes initializing. If promises are supported, it also returns a promise.

#### Syntax

```javascript
ready(callback)
```

#### Parameters

| Name       | Type       | Required | Description                                            |
|------------|------------|----------|--------------------------------------------------------|
| `callback` | `Function` |          | Callback to call when Analytics finishes initializing. |

It returns a `Promise` that resolves or rejects when Analytics finishes initializing. If the browser does not support promises and no polyfill has been installed, it returns `undefined`.       

#### Example

```javascript
chichianalytics.ready(() => console.log('Analytics has been inizialized'));
```

```javascript
import Analytics from 'chichi-javascript-sdk';
const analytics = new Analytics('<write key>', '<endpoint>');
await analytics.ready();
```

## reset

The `reset` method resets the user and group identifiers, and traits removing them from the storage. It also resets the Anonymous ID by generating a new one, and ends the session if one exists.

> See also the [anonymize](#anonymize) method and the [differences between reset and anonymize](../events/anonymize.md#reset-vs-anonymize). 

#### Syntax

```javascript
reset()
```

#### Parameters

There are no parameters.

#### Example

```javascript
chichianalytics.reset();
```

## debug

The `debug` method toggles debug mode.

#### Syntax

```javascript
debug(on)
```

#### Parameters

| Name       | Type      | Required                               | Description                            |
|------------|-----------|----------------------------------------|----------------------------------------|
| `on`       | `Boolean` | <div style="text-align:center">✓</div> | Indicates if the debug mode is active. |


#### Example

```javascript
chichianalytics.debug(true);
```

## close

The `close` method closes the Analytics instance. It tries to preserve the queue in the localStorage before returning.

#### Syntax

```javascript
close()
```

#### Parameters

There are no parameters.

#### Example

```javascript
chichianalytics.close();
```

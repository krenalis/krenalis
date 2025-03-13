{% extends "/layouts/doc.html" %}
{% macro Title string %}JavaScript SDK Methods{% end %}
{% Article %}

# JavaScript SDK methods

The JavaScript SDK is equipped to handle all essential event calls, including `page`, `screen`, `track`, `identify`, and `group`. Furthermore, it offers functionalities to efficiently manage sessions and user/group information.

With these capabilities, you can seamlessly track and analyze user interactions across different platforms, facilitating a comprehensive understanding of user behavior and engagement.

Below the JavaScript SDK methods:

- [page](#page)
- [screen](#screen)
- [track](#track)
- [identify](#identify)
- [group](#group)
- [user](#user)
- [setAnonymousId](#setanonymousid)
- [getSessionId](#getsessionid)
- [startSession](#startsession)
- [endSession](#endsession)
- [ready](#ready)
- [reset](#reset)
- [debug](#debug)
- [close](#close)

> The JavaScript SDK also supports the `getAnonymousId` method of the RudderStack SDK that can be used as an alternative to the [`user().anonymousId`](user-class#anonymousid).

## page

The page method implements the [page call](../../events/page).

The page call allows you to capture when a user views a page on your website, including any extra details about that specific page.

#### Syntax

```javascript
page(category, name, properties, options, callback)
```
<details>
<summary>TypeScript syntax</summary>

```typescript
page(
    name: string,
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>

page(
    category: string,
    name: string,
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>

page(
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>
```

</details>

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
meergo.page('Shirt', {
    productId: 308263,
}).then(() => console.log('event queued'));
```

## screen

The screen method implements the [screen call](../../events/screen).

The screen call enables you to capture instances when a user views a screen and record associated properties or details about that particular screen.

#### Syntax

```javascript
screen(category, name, properties, options, callback)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
screen(
    name: string,
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>

screen(
    category: string,
    name: string,
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>

screen(
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>
```

</details>


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
meergo.screen('Order Completed', {
	items: 3,
    total: 274.99,
}).then(() => console.log('event queued'));
```

## track

The track method implements the [track call](../../events/track).

The track call is used to send specific events or actions, and associated properties, that occur when users interact with your application or website.

#### Syntax

```javascript
track(name, properties, options, callback)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
track(
    name: string,
    properties?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>
```

</details>

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
meergo.screen('Order Completed', {
	items: 3,
	total: 274.99,
});
```

## identify

The identify method implements the [identify call](../../events/identify).

Through an identify call, you can connect previous and upcoming events to a recognized user and save details about them along with their events, such as name and email. The user information can also be utilized to update and enhance unified data from other sources.

#### Syntax

```javascript
identify(userId, traits, options, callback)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
identify(
    userId: string,
    traits?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>

identify(
    traits?: string,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>
```

</details>

#### Parameters

All parameters are optional. A single `Object` parameter signifies `traits`.

| Name       | Type       | Required | Description                                                                              |
|------------|------------|----------|------------------------------------------------------------------------------------------|
| `userId`   | `String`   |          | Identifier of the user.                                                                  |
| `traits`   | `Object`   |          | Traits to add to the user's traits. Pass the `undefined` value for a trait to remove it. |
| `options`  | `Object`   |          | Options of the event.                                                                    |
| `callback` | `Function` |          | A function called when the event has been queued.                                        |

It returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

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

For the previous example, we can have three scenarios:

* If the user is currently anonymous, they become known with the given User ID and traits.
* If the user is currently non-anonymous but has a different User ID, their User ID is changed, and all traits are replaced with the provided ones.
* If the user is currently non-anonymous and has the same User ID as the provided one, the provided traits are added to the current traits. If a provided trait has the value `undefined`, the corresponding trait is removed.

> To completely replace the current user's traits, regardless of whether the user is anonymous or non-anonymous, and without making an [`identify`](../../events/identify) call, use the [`user().traits`](user-class#traits) method.  

## group

The `group` method, when called without arguments, returns an instance of the [Group class](group-class) representing the group.

When called with arguments, implements the [group call](../../events/group). The group call provides a way to associate individual users with groups, such as a company, organization, team, association, or initiative. A user who has been identified can be associated with several groups.

#### Syntax

```javascript
// returns a Group instance
group()

// implements the group call
group(groupId, traits, options, callback)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
// returns a Group instance
group(): Group

// implements the group call
group(
    groupId: string,
    traits?: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>

// implements the group call
group(
    traits: Record<string, unknown>,
    options?: Record<string, unknown>,
    callback?: () => void
): Promise<SentEvent>
```

</details>

#### Parameters

All parameters are optional. A single `Object` parameter signifies `traits`.

| Name       | Type       | Required | Description                                       |
|------------|------------|----------|---------------------------------------------------|
| `groupId`  | `String`   |          | Identifier of the group.                          |
| `traits`   | `Object`   |          | Traits of the group.                              |
| `options`  | `Object`   |          | Options of the event.                             |
| `callback` | `Function` |          | A function called when the event has been queued. |

If no arguments are provided, it returns an instance of the [Group class](group-class). Otherwise, it returns a `Promise` that resolve when the event has queued. If the browser does not support promises and a polyfill has not been installed, it returns `undefined`.

#### Example

```javascript
const groupId = meergo.group().id();
```

```javascript
meergo.group('84s76y49tb28v1jxq', {
	name: "AcmeTech",
	industry: "Technology",
	employeeCount: 100
});
```

## user

The `user` method returns an instance of the [`User class`](user-class) to read and set the identifier, Anonymous ID, and traits of the user.

#### Syntax

```javascript
user()
```

<details>
<summary>TypeScript syntax</summary>

```typescript
user(): User
```

</details>

#### Parameters

There are no parameters.

Returns an instance of the [`User class`](user-class) representing the user.

#### Example

```javascript
const traits = meergo.user().traits();
```

## setAnonymousId

The `setAnonymousId` method is used to set the Anonymous ID. It's necessary because you can't call the `user().anonymousId` method before `Meergo` is ready. If you need to set the Anonymous ID before `Meergo` is ready, using the `setAnonymousId` method is your only option.

Call the `setAnonymousId` method with an argument:

- to reset the Anonymous ID with a newly generated value, pass `null`.

- to set the Anonymous ID with a specified value, pass a non-empty `String` or a `Number` (the number will be converted to a `String`).

If it is called after `Meergo` is ready, it also returns the Anonymous ID.

#### Syntax

```javascript
setAnonymousId(id)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
setAnonymousId(id?: string): string | undefined
```

</details>

#### Parameters

| Name | Type       | Required | Description                                            |
|------|------------|----------|--------------------------------------------------------|
| `id` | `String`   |          | Anonymous ID to set. If it is missing it does nothing. |

It returns the Anonymous ID, if called after `Meergo` is ready.

#### Example

```javascript
meergo.setAnonymousId('cd320a46-0642-468c-9f03-c8647faa8ac4');
```

## getSessionId

The `getSessionId` method returns the current session identifier.

#### Syntax

```javascript
getSessionId()
```

<details>
<summary>TypeScript syntax</summary>

```typescript
getSessionId(): number
```

</details>

#### Parameters

There are no parameters. Returns a `Number` representing the current session identifier, or `null` if there is no session.  

#### Example

```javascript
const sessionId = meergo.getSessionId();
```

## startSession

The `startSession` method starts a new session using the provided identifier. If no identifier is provided, it generates one automatically. 

#### Syntax

```javascript
startSession(id)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
startSession(id?: number): void
```

</details>

#### Parameters

| Name | Type     | Required | Description         |
|------|----------|----------|---------------------|
| `id` | `Number` |          | Session identifier. |

#### Example

```javascript
meergo.startSession();
```

## endSession

The `endSession` method ends the session.

#### Syntax

```javascript
endSession()
```

<details>
<summary>TypeScript syntax</summary>

```typescript
endSession(): void
```

</details>

#### Parameters

There are no parameters.

#### Example

```javascript
meergo.endSession();
```

## ready

The `ready` method calls a callback after the Meergo finishes initializing. If promises are supported, it also returns a promise.

#### Syntax

```javascript
ready(callback)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
ready(callback?: () => void): Promise<void>
```

</details>

#### Parameters

| Name       | Type       | Required | Description                                            |
|------------|------------|----------|--------------------------------------------------------|
| `callback` | `Function` |          | Callback to call when Meergo finishes initializing. |

It returns a `Promise` that resolves or rejects when Meergo finishes initializing. If the browser does not support promises and no polyfill has been installed, it returns `undefined`.       

#### Example

```javascript
meergo.ready(() => console.log('Meergo has been inizialized'));
```

```javascript
import Meergo from 'meergo-javascript-sdk';
const meergo = new Meergo('<write key>', '<endpoint>');
await meergo.ready();
```

## reset

The `reset` method resets the user and group identifiers, and updates or removes the Anonymous ID and traits according to the strategy (as detailed in the table below). If `all` is true it always resets the Anonymous ID by generating a new one, and ends the session if one exists, regardless of the strategy.

| Strategy     | Behavior of `reset()`                                                                                                                               |
|--------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| Conversion   | Removes User ID, Group ID, user and group traits, and changes Anonymous ID and session.                                                             |
| Fusion       | Removes User ID, Group ID, and user and group traits. Does not change Anonymous ID or session.                                                      |
| Isolation    | Removes User ID, Group ID, user and group traits, and changes Anonymous ID and session.                                                             |
| Preservation | Removes User ID. Restores Anonymous ID, Group ID, user and group traits, and session to their state before the latest [`identify`](#identify) call. |


#### Syntax

```javascript
reset(all)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
reset(all?: boolean): void
```

</details>

#### Parameters

| Name       | Type      | Required | Description                                                                              |
|------------|-----------|----------|------------------------------------------------------------------------------------------|
| `all`      | `Boolean` |          | Indicates if the Anonymous ID and the session must be reset, regardless of the strategy. |

#### Example

```javascript
meergo.reset();  // same as meergo.reset(false)
```

#### Segment Compatibility

To align with Segment's `reset()` behavior, choose the "Conversion" or "Isolation" strategy in Meergo. Note that `reset(true)` is not available in Segment.

#### RudderStack Compatibility

To match RudderStack's `reset()` behavior, choose the "Conversion" or "Isolation" strategy in Meergo. In Meergo, `reset(true)` works the same way as it does in RudderStack for all strategies.

## debug

The `debug` method toggles debug mode.

#### Syntax

```javascript
debug(on)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
debug(on: boolean): void
```

</details>

#### Parameters

| Name       | Type      | Required                               | Description                            |
|------------|-----------|----------------------------------------|----------------------------------------|
| `on`       | `Boolean` | <div style="text-align:center">✓</div> | Indicates if the debug mode is active. |


#### Example

```javascript
meergo.debug(true);
```

## close

The `close` method closes the Meergo instance. It tries to preserve the queue in the localStorage before returning.

#### Syntax

```javascript
close()
```

<details>
<summary>TypeScript syntax</summary>

```typescript
close(): void
```

</details>

#### Parameters

There are no parameters.

#### Example

```javascript
meergo.close();
```

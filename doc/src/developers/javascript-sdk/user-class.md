{% extends "/layouts/doc.html" %}
{% macro Title string %}User Class{% end %}
{% Article %}

# User class

The `User` class represents a user. An instance representing the current user is returned calling the [`user`](methods#user) method of `Meergo`. For example:

```javascript
const userId = meergo.user().id();
```

## id

The `id` method is used to get and set the identifier of the user. It always returns the user's identifier, or `null` if there is no identifier.

To set the user's identifier, call the `id` method with an argument:

- to remove the identifier, pass a `null` argument.
- to change the identifier and resets the Anonymous ID, pass a non-empty `String` or a `Number` (the number will be converted to a `String`). If the passed identifier is the same of the current identifier, it does nothing.

#### Syntax

```javascript
id(id)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
id(id?: string | null): string | null
```

</details>

#### Parameters

| Name | Type                           | Required | Description                                                                |
|------|--------------------------------|----------|----------------------------------------------------------------------------|
| `id` | `String`&nbsp;or&nbsp;`Number` |          | User identifier to set. If it is `null`, the user's identifier is removed. |


#### Examples

```javascript
const userId = meergo.user().id();
```

```javascript
meergo.user().id(null);
```

```javascript
meergo.user().id('509284521');
```

## anonymousId

The `anonymousId` method is used to retrieve and modify the Anonymous ID. It consistently returns the Anonymous ID.

To modify the Anonymous ID, call the `anonymous` method with an argument:

- to reset the Anonymous ID with a newly generated value, pass `null`.
- to set the Anonymous ID with a specified value, pass a non-empty `String` or a `Number` (the number will be converted to a `String`).

#### Syntax

```javascript
anonymousId(id)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
anonymousId(id?: string | null): string
```

</details>

#### Parameters

| Name | Type                           | Required | Description          |
|------|--------------------------------|----------|----------------------|
| `id` | `String`&nbsp;or&nbsp;`Number` |          | Anonymous ID to set. |


#### Examples

```javascript
const anonymousId = meergo.user().anonymousId();
```

```javascript
meergo.user().anonymousId(null);
```

```javascript
meergo.user().anonymousId('e2984831-431d-44ad-b1ec-4b901392fb67');
```

## traits

The `traits` method is used to access and modify a user's traits. These traits are for the anonymous user if the user is anonymous, and for the non-anonymous user if non-anonymous. It consistently returns the user's traits.

To modify the user's traits, call the `traits` method with an argument:

- To remove all traits, pass a `null` argument.
- To update the traits, provide a non-null `Object`. Since traits are serialized with `JSON.stringify`, they must consist only of serializable values and should not contain cyclic references. In case of serialization errors, a warning will be logged in the console. The provided traits will completely replace the current traits of the user, if any.

#### Syntax

```javascript
traits(traits)
```

<details>
<summary>TypeScript syntax</summary>

```typescript
traits(traits?: Record<string, unknown> | null): Record<string, unknown>
```

</details>

#### Parameters

| Name     | Type      | Required | Description                                        |
|----------|-----------|----------|----------------------------------------------------|
| `traits` | `Object`  |          | User's traits to set. `null` to remove all traits. |


#### Examples

```javascript
const traits = meergo.user().traits();
```

```javascript
meergo.user().traits(null);
```

```javascript
meergo.user().traits({ firstName: 'Emily', lastName: 'Johnson' });
```


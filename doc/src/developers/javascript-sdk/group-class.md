{% extends "/layouts/doc.html" %}
{% macro Title string %}Group Class{% end %}
{% Article %}

# Group class

The `Group` class represents a group. An instance representing the current group is returned calling  the [`group`](methods#group) method of `Meergo`. For example:

```javascript
const groupId = meergo.group().id();
```

## id

The `id` method is used to get and set the identifier of the group. It always returns the group's identifier, or `null` if there is no identifier.

To set the group's identifier, call the `id` method with an argument:

- to remove the identifier, pass a `null` argument.
- to change the identifier, pass a non-empty `String` or a `Number` (the number will be converted to a `String`). If the passed identifier is the same of the current identifier, it does nothing.

#### Syntax

```javascript
id(id)
```

<details class="typescript">
<summary><span>TypeScript</span></summary>

```typescript
id(id?: string | null): string | null
```

</details>

#### Parameters

| Name | Type                           | Required | Description                                                                  |
|------|--------------------------------|----------|------------------------------------------------------------------------------|
| `id` | `String`&nbsp;or&nbsp;`Number` |          | Group identifier to set. If it is `null`, the group's identifier is removed. |


#### Examples

```javascript
const groupId = meergo.group().id();
```

```javascript
meergo.group().id(null);
```

```javascript
meergo.group().id('acme');
```

## traits

The `traits` method is utilized for accessing and modifying the group's traits. It consistently returns the group's traits.

To modify the group's traits, utilize the `traits` method with an argument:

- To remove all traits, pass a `null` argument.
- To update the traits, provide a non-null `object`. Since traits are serialized with `JSON.stringify`, they must consist only of serializable values and should not contain cyclic references. In case of serialization errors, a warning will be logged in the console.

#### Syntax

```javascript
traits(traits)
```

<details class="typescript">
<summary><span>TypeScript</span></summary>

```typescript
traits(traits?: Record<string, unknown> | null): Record<string, unknown>
```

</details>

#### Parameters

| Name     | Type      | Required | Description                                         |
|----------|-----------|----------|-----------------------------------------------------|
| `traits` | `Object`  |          | Group's traits to set. `null` to remove all traits. |


#### Examples

```javascript
const traits = meergo.group().traits();
```

```javascript
meergo.group().traits(null);
```

```javascript
meergo.group().traits({ name: 'Acme Inc.' });
```


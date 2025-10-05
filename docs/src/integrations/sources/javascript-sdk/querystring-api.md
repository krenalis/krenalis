{% extends "/layouts/doc.html" %}
{% macro Title string %}Querystring API{% end %}
{% Article %}

# Querystring API

The Querystring API of the SDK enables the activation of tracking and identification events using the URL query string. This practical feature is particularly valuable for monitoring email click-throughs, social media clicks, and digital advertising engagements.

The Querystring API is enabled by default but can be turned off using the [`useQueryString`](options#usequerystring-option) option.

Below is a list of the query string parameters:

| Parameter             | Description                                                                                                                                                   |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ajs_aid`             | Anonymous ID. The current Anonymous ID is updated with the parameter's value and is passed to the `identify` and `track` calls if triggered.                  |
| `ajs_uid`             | User ID. The current User ID is updated with the parameter's value, and it triggers the `identify` call, and it is passed as argument.                        |
| `ajs_trait_<trait>`   | User trait. The current user traits are updated with the parameter's name and value, and the resulting traits are passed to the `identify` call if triggered. |
| `ajs_event`           | Event name. It triggers the `track` call and it is passed as argument.                                                                                        |
| `ajs_prop_<property>` | Event property. It is passed to the `track` call if triggered.                                                                                                |

The `identity` call is made only when the `ajs_uid` parameter is present, while the `track` call is made only when the `ajs_event` parameter is present.

### Example

The following URL

```
https://example.com?ajs_aid=anon_3094671&ajs_uid=510375492&ajs_trait_name=Emily+Johnson&ajs_event=Subscribed&ajs_prop_edition=standard&ajs_prop_monthly=yes
```

automatically sets the Anonymous ID to `anon_3094671` and triggers the `identify` and `track` calls:

```javascript
meergo.user().anonymousId('anon_3094671');
meergo.identify('510375492', { name: 'Emily Johnson' });
meergo.track('Subscribed', { edition: 'standard', monthly: 'yes' });
```

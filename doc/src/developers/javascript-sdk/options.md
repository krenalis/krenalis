{% extends "/layouts/doc.html" %}
{% macro Title string %}Options{% end %}
{% Article %}

# Options

When initializing the JavaScript SDK, you have the flexibility to provide various initialization options. Let's start by understanding how to pass these options to the SDK.

### Using the snippet

When utilizing the snippet, locate the following line at the end of the snippet:

```javascript
meergo.load('<write key>', '<endpoint>');
```

Instead of `<write key>` and `<endpoint>`, you should already see your specific write key and endpoint, respectively. Following these two arguments, add a third argument of type object:

```javascript
meergo.load('<write key>', '<endpoint>', { <options> });
```

Where `<options>` represent one or more of the options shown below.

### Using `import` or `require`

When importing the package using `import` or `require`, pass an additional argument to the `Meergo` constructor.  

```javascript
const meergo = new Meergo('<write key>', '<endpoint>', { <options> });
```

Where `<options>` represent one or more of the options shown below.

## Options

Below are the options that control various aspects of the JavaScript SDK.

| Option                                     | Type                                 | Default | Description                                                                                                                                                      |
|--------------------------------------------|--------------------------------------|---------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [`cookie`](#cookie-option)                 | `Object`                             |         | Controls specific cookie settings when used as storage.                                                                                                          |
| `debug`                                    | `Boolean`                            | `false` | **debug mode**: when enabled status messages will appear on the console. You can also enable/disable debug mode later using the [`debug`](methods#debug) method. |
| [`group`](#group-option)                   | `Object`                             |         | Customize the storage priority for group data.                                                                                                                   |
| [`session`](#session-option)               | `Object`                             |         | Controls whether the [session tracking](../../events/session-tracking) is automatic or not, and sets its timeout.                                                   |
| [`storage`](#storage-option)               | `Object`                             |         | Customize the global storage priority.                                                                                                                           |
| [`useQueryString`](#usequerystring-option) | `Boolean`<center>or</center>`Object` | `true`  | Indicates whether to process query parameters using the [Querystring API](querystring-api), and if enforce validation rules.                                     |
| [`user`](#user-option)                     | `Object`                             |         | Customize the storage priority for user data.                                                                                                                    |

> The JavaScript SDK also supports the following options from the RudderStack SDK: `secureCookie`, `sameDomainCookiesOnly`, `sameSiteCookie`, `setCookieDomain`, `storage.cookie`, and `storage.type`.

### cookie option

The `cookie` option has the following sub-options:

| Option     | Type      | Default | Description                                                                                                                                                                                                 |
|------------|-----------|---------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `domain`   | `String`  | `null`  | The "domain" attribute of cookies. If it is an empty string, it refers to that of the current page, and if it is `null`, it refers to the top-level domain of the current page. See below for more details. |
| `maxage`   | `Number`  | 365     | The "maxage" attribute of cookies in days.                                                                                                                                                                  |
| `path`     | `String`  | `'/'`   | The "path" attribute of cookies.                                                                                                                                                                            |
| `sameSite` | `String`  | `'Lax'` | The "sameSite" attribute of cookies. It can be `'Lax'`, `'Strict'`, or `'None'`.                                                                                                                            |
| `secure`   | `Boolean` | `false` | The "secure" attribute of cookies.                                                                                                                                                                          |

#### domain

By default, the JavaScript SDK uses top-level domains (`domain` option defaults to `null`). This facilitates user identification across different subdomains. For instance, if you embed the JavaScript SDK into both the blog section (blog.example.com) and the store section (store.example.com) of a website, the SDK will store the cookie at example.com.

If you prefer a different approach, you have options. You can set the domain to an empty string to utilize the current page's domain, or you can specify the precise domain you wish to use.

#### Example:

```javascript
meergo.load('<write key>', '<endpoint>', {
    cookie: {
        domain: 'www.example.com',
        maxage: 365,
        path: '/',
        sameSite: 'Strict',
        secure: true
    }
});
```

### group option

The `group` option has the following sub-option:

| Option                       | Type     | Default | Description                                    |
|------------------------------|----------|---------|------------------------------------------------|
| [`storage`](#storage-option) | `Object` |         | Customize the storage priority for group data. |

#### Example:

```javascript
meergo.load('<write key>', '<endpoint>', {
	group: {
		storage: {
			stores: ['cookies', 'memory']
		}
	}
});
```

### session option

The `session` option has the following sub-options:

| Option      | Type      | Default    | Description                                                                                    |
|-------------|-----------|------------|------------------------------------------------------------------------------------------------|
| `autoTrack` | `Boolean` | `true`     | Indicates if the auto tracking is enabled. See [session tracking](../../events/session-tracking). |
| `timeout`   | `Number`  | 30 minutes | Timeout in milliseconds for session expiration due to inactivity.                              |

#### Example:

```javascript
meergo.load('<write key>', '<endpoint>', {
	session: {
		autoTrack: true, // enable the auto tracking
        timeout: 45 * 60 * 60000 // 45 minutes
    }
});
```

### useQueryString option

The `useQueryString` option provides control over how query parameters are handled, as defined in the [Querystring API](querystring-api). You can deactivate query string processing entirely by setting `useQueryString` to `false`.

Alternatively, you can keep it active but validate the Anonymous ID and User ID using regular expressions. To do this, set `useQueryString` to an object, instead of a boolean, with the following properties:

| Option | Type     | Default    | Description                                             |
|--------|----------|------------|---------------------------------------------------------|
| `aid`  | `RegExp` | `/\s\S/`   | Regular expression to validate the `ajs_aid` parameter. |
| `uid`  | `RegExp` | `/\s\S/`   | Regular expression to validate the `ajs_uid` parameter. |


#### Examples

```javascript
meergo.load('<write key>', '<endpoint>', {
	useQueryString: {
		aid: /^[a-z0-9]+$/
    }
});
```

### user option

The `user` option has the following sub-option:

| Option                       | Type     | Default | Description                                   |
|------------------------------|----------|---------|-----------------------------------------------|
| [`storage`](#storage-option) | `Object` |         | Customize the storage priority for user data. |

#### Example:

```javascript
meergo.load('<write key>', '<endpoint>', {
	user: {
		storage: {
			stores: ['cookies', 'memory']
		}
	}
});
```

### Storage option

The JavaScript SDK enables configuration of the preferred storage location. By default, it stores data in the browser's localStorage and cookies. You have the flexibility to modify this default setting globally or specifically for user or group data.

The `storage` option has the following sub-option:

| Option    | Type    | Default                     | Description                           |
|-----------|---------|-----------------------------|---------------------------------------|
| `stores`  | `Array` | `['localStorage','cookie']` | Storage locations used to store data. |

The `storage.stores` option specifies the preferred storage locations in order of priority: localStorage, sessionStorage, cookie, and memory. If `storage.stores` is empty, no data will be stored. 

See also [Storage&nbsp;Locations](storage-locations).

#### Example:

```javascript
meergo.load('<write key>', '<endpoint>', {
	// Global storage locations.
	storage: {
        stores: ['cookies', 'localStorage', 'memory'] 
    },
	// Storage locations to user data.
	user: {
		storage: {
			stores: ['cookies', 'memory'] 
		}
	},
	// Storage locations to group data.
	group: {
		storage: {
			stores: ['localStorage', 'memory'] 
		}
	}
});
```

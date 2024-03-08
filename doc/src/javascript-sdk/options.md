
# Options

When initializing the JavaScript SDK, you have the flexibility to provide various initialization options. Let's start by understanding how to pass these options to the SDK.

### Using the Snippet

When utilizing the snippet, locate the following line at the end of the snippet:

```javascript
a.load('<write key>', '<endpoint>');
```

Instead of `<write key>` and `<endpoint>`, you should already see your specific write key and endpoint, respectively. Following these two arguments, add a third argument of type object:

```javascript
a.load('<write key>', '<endpoint>', { <options> });
```

Where `<options>` represent one or more of the options shown below.

### Using `import` or `require`

When importing the package using `import` or `require`, pass an additional argument to the `Analytics` constructor.  

```javascript
const analytics = new Analytics('<write key>', '<endpoint>', { <options> });
```

Where `<options>` represent one or more of the options shown below.

## Options

Below are the options that control various aspects of the JavaScript SDK.

| Option                                     | Type                                 | Default | Description                                                                                                                                      |
|--------------------------------------------|--------------------------------------|---------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| `debug`                                    | `Boolean`                            | `false` | **debug mode**: when enabled status messages will appear on the console. You can also enable/disable debug mode later using the `debug` method.  |
| [`session`](#session-option)               | `Object`                             |         | Controls whether the [session tracking](../events/session-tracking.md) is automatic or not, and sets its timeout.                                |
| [`storage`](#storage-option)               | `Object`                             |         | Customizes various storage-related functionalities, such as selecting preferred storages and configuring cookie settings.                        |
| [`useQueryString`](#usequerystring-option) | `Boolean`<center>or</center>`Object` | `true`  | Indicates whether to process query parameters using the [Querystring API](querystring-api.md), and if enforce validation rules.                  |

> The JavaScript SDK also supports the following options that can be used when migrating from the RudderStack SDK: `secureCookie`, `sameDomainCookiesOnly`, `sameSiteCookie`, and `setCookieDomain`.

### session option

The `session` option has the following sub-options:

| Option      | Type      | Default    | Description                                                                                       |
|-------------|-----------|------------|---------------------------------------------------------------------------------------------------|
| `autoTrack` | `Boolean` | `true`     | Indicates if the auto tracking is enabled. See [session tracking](../events/session-tracking.md). |
| `timeout`   | `Number`  | 30 minutes | Timeout in milliseconds for session expiration due to inactivity.                                 |

#### Example:

```javascript
a.load('<write key>', '<endpoint>', {
	session: {
		autoTrack: true, // enable the auto tracking
        timeout: 45 * 60 * 60000 // 45 minutes
    }
});
```

### storage option

The `storage` option has the following sub-options:

| Option                            | Type     | Default          | Description                                                                                 |
|-----------------------------------|----------|------------------|---------------------------------------------------------------------------------------------|
| [`cookie`](#storagecookie-option) | `Object` |                  | Controls specific cookie settings when used as storage.                                     |
| [`type`](#storagetype-option)     | `String` | `"multiStorage"` | Determines witch storage to use for user data. See also [Data Storages](data-storages.md).  |

#### Example:

```javascript
a.load('<write key>', '<endpoint>', {
	storage: {
		cookie: {
			secure: true
		},
        type: "localStorage"
    }
});
```

### storage.cookie option

The `storage.cookie` option has the following sub-options:

| Option     | Type      | Default       | Description                                                                                                                                                                                                 |
|------------|-----------|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `domain`   | `String`  | `null`        | The "domain" attribute of cookies. If it is an empty string, it refers to that of the current page, and if it is `null`, it refers to the top-level domain of the current page. See below for more details. |
| `maxage`   | `Number`  | one&nbsp;year | The "maxage" attribute of cookies in milliseconds.                                                                                                                                                          |
| `path`     | `String`  | `"/"`         | The "path" attribute of cookies.                                                                                                                                                                            |
| `samesite` | `String`  | `"Lax"`       | The "samesite" attribute of cookies. It can be `"Lax"`, `"Strict"`, or `"None"`.                                                                                                                            |
| `secure`   | `Boolean` | `false`       | The "secure" attribute of cookies.                                                                                                                                                                          |


#### domain

By default, the JavaScript SDK uses top-level domains (`domain` option defaults to `null`). This facilitates User IDentification across different subdomains. For instance, if you embed the JavaScript SDK into both the blog section (blog.example.com) and the store section (store.example.com) of a website, the SDK will store the cookie at example.com.

If you prefer a different approach, you have options. You can set the domain to an empty string to utilize the current page's domain, or you can specify the precise domain you wish to use.

#### Example:

```javascript
a.load('<write key>', '<endpoint>', {
	storage: {
		cookie: {
			domain: "www.example.com",
            maxAge: 30 * 24 * 60 * 60 * 1000, // 30 days
			path: "/",
			sameSite: "Strict",
			secure: true
		}
    }
});
```

### storage.type option

Below are the possible values that can be used for the option `store.type`:

| Storage Type       | Description                                                                                                                                                                               |
|--------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `"multiStorage"`   | Attempt to store the data in both **cookies** and, depending on availability, in one of the following: **local storage**, **session storage**, and **memory**. This is the default value. |
| `"cookieStorage"`  | Prioritize storing user data in the following order: **cookies**, **local storage**, **session storage**, and **memory**, utilizing the first available storage option.                   |
| `"localStorage"`   | Prioritize storing user data in the following order: **local storage**, and **memory**, utilizing the first available storage option.                                                     |
| `"sessionStorage"` | Prioritize storing user data in the following order: **session storage**, and **memory**, utilizing the first available storage option.                                                   |
| `"memoryStorage"`  | Store user data in **memory**.                                                                                                                                                            |
| `"none"`           | Do not store user data.                                                                                                                                                                   |

See also [Data Storages](data-storages.md).

#### Example:

```javascript
a.load('<write key>', '<endpoint>', {
	storage: {
		type: "localStorage" 
    }
});
```

### useQueryString option

The `useQueryString` option provides control over how query parameters are handled, as defined in the [Querystring API](querystring-api.md). You can deactivate query string processing entirely by setting `useQueryString` to `false`.

Alternatively, you can keep it active but validate the Anonymous ID and User ID using regular expressions. To do this, set `useQueryString` to an object, instead of a boolean, with the following properties:

| Option | Type     | Default    | Description                                             |
|--------|----------|------------|---------------------------------------------------------|
| `aid`  | `RegExp` | `/\s\S/`   | Regular expression to validate the `ajs_aid` parameter. |
| `uid`  | `RegExp` | `/\s\S/`   | Regular expression to validate the `ajs_uid` parameter. |


#### Examples

```javascript
a.load('<write key>', '<endpoint>', {
	useQueryString: {
		aid: /^[a-z0-9]+$/
    }
});
```

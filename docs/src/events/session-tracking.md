{% extends "/layouts/doc.html" %}
{% macro Title string %}Session Tracking{% end %}
{% Article %}

# Session tracking

A user session is a series of interactions that a user has with a website or application within a specific time frame. All events recorded within a session will be associated with the same Session ID. Meergo provides the necessary feature to manage user sessions.

Session tracking is vital for understanding user behavior and optimizing product workflows. By combining event tracking metrics with session metadata, businesses can gather insights into the user's product journey, identify issues, and seize optimization opportunities, enabling data-driven decision-making for an enhanced online experience.

Meergo SDKs provide a comprehensive set of functions for session management:

* Automatic and manual session creation.
* Inclusion of a session identifier with each event.
* Automatic and manual session expiration.
* Configurable session duration.

Additionally, you can start a session with a specific identifier.

### Session in the JSON body

When a Meergo SDK sends an event, it includes two session fields. Here's an example of how a session appears in Meergo upon receiving an event:

```json
{
  ...
  "sessionId": 8745632109876543,
  "sessionStart": true,
  ...
}
```

`sessionId` denotes the Session ID, and `sessionStart` indicates whether the session started with this event.

> The Session ID is a 64-bit integer, meaning it can be represented within the range of [-9223372036854775808, 9223372036854775807]. However, certain SDKs may restrict the generated Session IDs for sessions to a smaller interval. For example, the [JavaScript SDK](/developers/javascript-sdk) generates Session IDs within the range [1, 9007199254740991]. 

### Use sessions in transformations

In transformations, access session information through the `context.session` object:

* `context.session.id` with type `int(64)`: representing the session identifier as sent by the website or application.
* `context.session.start` with type `boolean`: indicating whether the session started with this event.

### Data warehouse

In your data warehouse, Session ID and Session Start are stored respectively in the `context_session_id` and `context_session_start` columns of the `events` table.

## Website sessions

The session management functions depend on the specific SDK. The example below uses the [JavaScript SDK](/developers/javascript-sdk), but consult the SDK documentation for comprehensive details.

The Meergo [JavaScript SDK](/developers/javascript-sdk) automatically manages sessions unless specified otherwise. When a user visits your website for the first time, a session starts with a default timeout of 30 minutes.

Every time an event is recorded, the timeout resets, defaulting back to 30 minutes. However, if the timeout elapses, the current session expires. Upon the next event, a new session with a new Session ID will be created.

### End a session

To prematurely end a session, even before the timeout elapses, you can call the `endSession` function. The next session will automatically start with the subsequent event.

```javascript
meergo.endSession();
```

### Start a session

Alternatively, to promptly initiate a new session, expiring the current one if still ongoing, use the `startSession` function. Optionally, you can pass the desired Session ID as an argument to the function.

```javascript
meergo.startSession(sessionId);
```

The Session ID passed as an argument must be an integer and should have the `Number` type.  

### Get the session ID

Retrieve the Session ID of the current session with the `getSessionId` function.

```javascript
let sessionId = meergo.getSessionId();
```

### Disable automatic session tracking

With automatic session tracking, a session expires when the timeout period concludes, and a new session starts with the subsequent event.

This behavior is the default setting. To disable it, set the `autoTrack` option to `false` when loading the [JavaScript SDK](/developers/javascript-sdk):

```javascript
meergo.load(writeKey, endpoint, {
    sessions: {
        autoTrack: false // disable the automatic session tracking
    }
});
```
When automatic session tracking is disabled, you can still use the `startSession` and `endSession` functions to starts and end sessions.

### Change the session timeout

The default session timeout is 30 minutes. If you wish to set a different timeout, specify the `timeout` option with a value in milliseconds when loading the [JavaScript SDK](/developers/javascript-sdk):

```javascript
meergo.load(writeKey, endpoint, {
    sessions: {
        timeout: 15 * 60000 // set the timeout to 15 minutes
    }
});
```

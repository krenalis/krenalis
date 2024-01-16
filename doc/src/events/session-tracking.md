# Session Tracking

A user session is a series of interactions that a user has with a website or application within a specific time frame. All events recorded within a session will be associated with the same Session ID. Chichi provides the necessary feature to manage user sessions.

Session tracking is vital for understanding user behavior and optimizing product workflows. By combining event tracking metrics with session metadata, businesses can gather insights into the user's product journey, identify issues, and seize optimization opportunities, enabling data-driven decision-making for an enhanced online experience.

Chichi SDKs provide a comprehensive set of functions for session management:

* Automatic or manual session creation.
* Inclusion of a session identifier with each event.
* Automatic or manual session expiration.
* Configurable session duration.

### Use Sessions in Transformations

In transformations, access session information through the `context.session` object:

* `context.session.id` with type `Int(64)`: representing the session identifier as sent by the website or application.
* `context.session.start` with type `Boolean`: indicating whether the session started with this event.

### Data Warehouse

In your data warehouse, Session ID and Session Start are stored respectively in the `context_session_id` and `context_session_start` columns of the `events` table.

## Website sessions

The session management functions depend on the specific SDK. The example below uses the JavaScript SDK, but consult the SDK documentation for comprehensive details.

The Chichi JavaScript SDK automatically manages sessions unless specified otherwise. When a user visits your website for the first time, a session starts with a default timeout of 30 minutes.

Every time an event is recorded, the timeout resets, defaulting back to 30 minutes. However, if the timeout elapses, the current session expires. Upon the next event, a new session with a new Session ID will be created.

### End a Session

To prematurely end a session, even before the timeout elapses, you can call the `endSession` function. The next session will automatically start with the subsequent event.

```javascript
chichianalytics.endSession();
```

### Start a Session

Alternatively, to promptly initiate a new session, expiring the current one if still ongoing, use the `startSession` function. Optionally, you can pass the desired Session ID as an argument to the function.

```javascript
chichianalytics.startSession(sessionId);
```

#### Get the Session ID

Retrieve the Session ID of the current session with the `getSessionId` function.

```javascript
let sessionId = chichianalytics.getSessionId();
```

{% extends "/layouts/doc.html" %}
{% macro Title string %}Send Events{% end %}
{% Article %}

# Send events

Meergo makes it easy to send events to APIs that can receive them.

Here's how to get started with setting up your connector to send events:

```go
meergo.RegisterAPI(meergo.APISpec{
    ...
    AsDestination: &meergo.AsAPIDestination{
        ...
        Targets: meergo.TargetEvent | meergo.TargetUser,
        ...
    },
    ...
}, New)
```

This piece of code registers your connector, telling Meergo that it's ready to manage events (as well as users) when used as destination. Next, you'll need to implement the `EventSender` interface:

```go
// EventSender is implemented by API connectors that support event sending.
type EventSender interface {

	// EventTypeSchema returns the schema of the specified event type.
	//
	// The returned schema describes values required by the connector to send an
	// event of this type. Actions based on the specified event type will have a
	// transformation that, given the received event, provides the values required
	// by the connector. These values, along with the received event, are passed to
	// the connector's PreviewSendEvents and SendEvents methods.
	//
	// If no extra information is needed for the event type, the returned schema
	// is the invalid schema. If the event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	EventTypeSchema(ctx context.Context, eventType string) (types.Type, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)

	// PreviewSendEvents builds and returns the HTTP request that would be used to
	// send the given events to the API, without actually sending it.
	//
	// If any event type does not exist, it returns the ErrEventTypeNotExist error.
	//
	// Authentication data in the returned request is redacted (i.e., replaced with
	// "[REDACTED]"). If the destination action's identifier would appear in an
	// event identifier, it is replaced with "[ACTION]".
	//
	// This method is safe for concurrent use, on the same instance, by multiple
	// goroutines.
	PreviewSendEvents(ctx context.Context, events Events) (*http.Request, error)

	// SendEvents sends a non-empty sequence of events to an API.
	//
	// If any event type does not exist, it returns the ErrEventTypeNotExist error.
	//
	// If one or more events fail to be delivered, it returns an EventsError, which
	// includes a key for each failed event along with the corresponding error.
	//
	// If the returned error is not nil and not one of the above cases, it indicates
	// a failure in the request itself that cannot be retried.
	//
	// If all events are delivered successfully, it returns nil.
	//
	// This method is safe for concurrent use, on the same instance, by multiple
	// goroutines.
	SendEvents(ctx context.Context, events Events) error
}
```

Let's look more closely at what each part does.

## Event types

Your connector can be set up to handle different types of events. You'll use the `EventTypes` method of the `EventSender` interface to tell Meergo what these types are.

For every event type you support, you'll define a unique ID, a user-friendly name, and a description. Here's how you define an event type:

```go
type EventType struct {
    ID          string // unique identifier for the event type. Cannot be longer than 100 runes.
    Name        string // display name for the event type
    Description string // description of the event type
}
```

You have the freedom to decide on the identifiers, names, and descriptions, as long as each event type has a unique ID.

### Adding schema

Sometimes, the event might lack necessary information required for sending to the API. In such cases, the schema of the event type specifies the extra information needed.

Actions based on an event type involve a transformation that, given an event, provides the extra information required by the connector. This information, along with the event, is passed to the connector's `PreviewSendEvents` and `SendEvents` methods.

The schema of an event type is provided by the connector's `EventTypeSchema` method. If no extra information is needed for an event type, it must return the invalid schema.

For instance, if you need to send a ["share"](https://developers.google.com/analytics/devguides/collection/protocol/ga4/reference/events?hl=en#share) event to Google Analytics, you might require parameters like "method," "content_type," and "item_id," which could vary for each event. However, during the connector implementation stage, you might not have values for these parameters or know where to obtain them. In such cases, you can specify how to determine these parameters using a transformation in the action for the "share" event.

For example, the "share" event type might have the following schema:

```go
types.Object([]types.Property{
    {Name: "method", Type: types.Text()},
    {Name: "content_type", Type: types.Text()},
    {Name: "item_id", Type: types.Text()},
})
```

In the action of the "share" event type, if a mapping is chosen as a transformation, the schema of the event type would be represented as follows:

```
┌─────────────────────────────────┐
│                                 │ ->  method
└─────────────────────────────────┘
┌─────────────────────────────────┐
│                                 │ ->  content_type
└─────────────────────────────────┘
┌─────────────────────────────────┐
│                                 │ ->  item_id
└─────────────────────────────────┘
```

When sending the event, the `PreviewSendEvents` and `SendEvents` methods receives, as an argument, a sequence of events that should be sent. Each event of the sequence has the `Type.Values` field with the values of the three parameters "method," "content_type," and "item_id" conforming to the event type schema.

If a field in the schema is mandatory, set the `Required` field in the `types.Property` struct to `true`. Additionally, you can specify a placeholder using the `Placeholder` field for easier mapping compilation.

> When selecting a placeholder, consider that certain property names and traits hold specific meanings and can thus serve as suitable placeholders. Refer to the prefilled properties and traits sections within the events for further details:
>
>    - [page](/events/specs/page#prefilled-properties)
>    - [screen](/events/specs/screen#prefilled-properties)
>    - [track](/events/specs/track#prefilled-properties)
>    - [identify](/events/specs/identify#prefilled-traits)
>    - [group](/events/specs/group#prefilled-traits)

Now, let's move on to sending events to the API using the `SendEvents` method.

## Send events

Semplificata e fluida:

Finally, use the `SendEvents` method to send events to the API:


```go
SendEvents(ctx context.Context, events meergo.Events) error
```

The parameters are:

- `ctx`: The context.
- `events`: An iterator over the events.

The `events` parameter is a collection of events to send. **You don't need to process all the events in the collection at once.** Instead, handle only as many as can be sent in a single HTTP request to the API. Even if the API supports processing only one event per request, that's fine. Meergo will automatically call the method again for any events that remain unprocessed.

#### Key concept: processed events

Meergo considers an event processed as soon as it has been read from the `Events` collection. To better understand how this works, let’s first explore the methods provided by the `Events` interface. Afterward, we'll review how to use these methods effectively in various scenarios.

```go
// Events provides access to a non-empty sequence of events to be sent to an
// API.
//
// To iterate over events, call either All, SameUser, or First — only one of
// these can be used per Events value:
//   - All returns an iterator over all events.
//   - SameUser returns an iterator over events with the same user (events with
//     the same anonymous ID) as the first event.
//   - First returns the first event.
//
// Events are consumed as they are yielded by the iterator. An event is
// considered consumed once produced by the iterator, unless Postpone is called.
//
// Example:
//
//	for event := range events.All() {
//		...
//		// event is now consumed unless Postpone is called here
//		if postpone {
//			events.Postpone()
//			continue
//		}
//		...
//	}
//
// Calling Postpone during iteration marks the current event as not consumed, so
// it will be available in subsequent SendEvents or PreviewSendEvents calls.
//
// Only one iteration (using All or SameUser) or call to First may be active on
// an Events value. After an iteration completes or First is called, the Events
// value must not be used again.
type Events interface {

	// All returns an iterator to read all events. Type.Values of the events in the
	// sequence may be modified unless the event is subsequently postponed.
	All() iter.Seq[*Event]

	// Discard discards the current event in the iteration with the provided error.
	// Discard may only be called during iterations from All or SameUser.
	// It panics if err is nil, or if the record has already been postponed or
	// discarded.
	Discard(err error)

	// First returns the first event. The event's Type.Values may be modified.
	// After First is called, no further method calls on Events are allowed.
	First() *Event

	// Peek retrieves the next event without advancing the iterator. It returns the
	// event and true if an event is available, or false if there are no further
	// events. The returned event must not be modified.
	Peek() (*Event, bool)

	// Postpone postpones the current event in the iteration and marks it as unread.
	// Postpone may only be called during iterations from All or SameUser, and only
	// if the event's Type.Values have not been modified.
	Postpone()

	// SameUser returns an iterator over the events of the same user. Type.Values of
	// the events in the sequence may be modified unless the event is subsequently
	// postponed.
	SameUser() iter.Seq[*Event]
}
```

### Sending one event at a time

If the API can process only one event per request, use the `Events.First` method to retrieve just the first event.

Below is an example implementation:

```go
func (my *MyAPI) SendEvents(ctx context.Context, events meergo.Events) error {

    // Read only the first event.
    event := events.First()

    // Prepare the body.
    var body json.Buffer
    body.WriteString(`{"values":`)
    body.Encode(event.Type.Values)
    body.WriteString(`}`)

    // Create the HTTP request.
    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        "https://api.myapi.com/v1/event", bytes.NewReader(body.Bytes()))
    if err != nil {
        return err
    }

    // Add the headers.
    auth := my.settings.auth
    if redacted {
        auth = "[REDACTED]"
    }
    req.Header.Set("Authorization", "Bearer "+auth)
    req.Header.Set("Content-Type", "application/json")

    // Mark the request as idempotent.
    req.Header["Idempotency-Key"] = nil
    req.GetBody = func() (io.ReadCloser, error) {
        return io.NopCloser(bytes.NewReader(body.Bytes())), nil
    }

    // Send the events.
    res, err := my.httpClient.Do(req)
    if err != nil {
        return err
    }

    // Handle the response.
    // ...

    return nil
}
```

#### Key concepts:

* **Read the first event**\
  Use `events.First()` to read the first event in the collection.

This method ensures that only one event is processed per request, in accordance with the API's limitations. Meergo will automatically re-invoke the method to handle any remaining events.

### Batch of events from the same user

If the API supports processing multiple events in a batch but requires them to belong to the same user (events with the same anonymous ID), you can iterate over `events.SameUser()` to retrieve only the events associated with the first event's user.

Below is an example implementation:

```go
func (my *MyAPI) SendEvents(ctx context.Context, events meergo.Events) error {

    // Prepare the body.
    var body json.Buffer
    body.WriteString(`{"events":[`)

    n := 0
    for event := range events.SameUser() {
        if n > 0 {
            body.WriteByte(',')
        }
        body.WriteString(`{"values":`)
        body.Encode(event.Type.Values)
        body.WriteString(`}`)
        n++
        if n == bodyMaxEvents {
            break
        }
    }
    body.WriteString(`]}`)

    // Create the HTTP request.
    req, err := http.NewRequestWithContext(ctx, http.MethodPost,  "https://api.myapi.com/v1/events", &body)
    if err != nil {
        return err
    }
	
    // Add the headers.
    auth := my.settings.auth
    if redacted {
        auth = "[REDACTED]"
    }
    req.Header.Set("Authorization", "Bearer "+auth)
    req.Header.Set("Content-Type", "application/json")

    // Mark the request as idempotent.
    req.Header["Idempotency-Key"] = nil
    req.GetBody = func() (io.ReadCloser, error) {
        return io.NopCloser(bytes.NewReader(body.Bytes())), nil
    }

    // Send the events.
    res, err := my.env.HTTPClient.Do(req)
    if err != nil {
        return err
    }

    // Handle the response.
    // ...

    return nil
}
```

#### Key concepts:

* **Iterate over events**\
  Use `events.SameUser()` to read only the events belonging to the same user (events with the same anonymous ID) as the first event. This ensures that all events in the batch come from the same user.

* **Batch size limitation**\
  The example demonstrates breaking the loop once the maximum number of events (`bodyMaxEvents`) is reached. This ensures the request complies with the API's limits.
  
### Batch of events from mixed users

If the API supports sending multiple events from different users in a single HTTP request, you can iterate over all events using `events.All()`.

Here is an example implementation:

```go
func (my *MyAPI) SendEvents(ctx context.Context, events meergo.Events) error {

    // Prepare the body.
    var body json.Buffer
    body.WriteString(`{"events":[`)

    n := 0	
    for event := range events.All() {
        if n > 0 {
            body.WriteByte(',')
        }
        body.WriteString(`{"values":`)
        body.Encode(event.Type.Values)
        body.WriteString(`}`)
        n++
        if n == bodyMaxEvents {
            break
        }
    }
    body.WriteString(`]}`)

    // Create the HTTP request.
    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        "https://api.myapi.com/v1/events", bytes.NewReader(body.Bytes()))	
    if err != nil {
        return err
    }

    // Add the headers.
    auth := my.settings.auth
    if redacted {
        auth = "[REDACTED]"
    }
    req.Header.Set("Authorization", "Bearer "+auth)
    req.Header.Set("Content-Type", "application/json")

    // Mark the request as idempotent.
    req.Header["Idempotency-Key"] = nil
    req.GetBody = func() (io.ReadCloser, error) {
        return io.NopCloser(bytes.NewReader(body.Bytes())), nil
    }

    // Send the events.
    res, err := my.env.HTTPClient.Do(req)
    if err != nil {
        return err
    }
	
    // Handle the response.
    // ...

    return nil
}
```

#### Key concepts:

* **Iterating over all events**\
  The `events.All()` method iterates over all events, regardless of the user, allowing mixed batches to be processed in a single request.

* **Limit on events**\
  The loop stops once the maximum number of events (`bodyMaxEvents`) is reached, ensuring that the request body size stays within the API's limits.

This approach enables efficient processing of mixed events from different users in a single batch request, reducing the number of API requests needed.

### Handling body size limits

In the previous examples, the loop stops when the number of events reaches the API's maximum limit. However, if the API imposes a body size limit rather than an event count limit, you can use the `Postpone` method to postpone an event after it has been read. This ensures that the event remains unprocessed and can be included in a subsequent call to the `SendEvents` method.

Below is an example implementation:

```go
body.WriteString(`{"events":[`)

first := true
for event := range events.All() {

    // Track size before adding the event.
    size := body.Len()

    if !first {
        body.WriteString(`,`)
    }
    first = false

    // Build the event JSON object.
    body.WriteString(`{"values":`)
    body.Encode(event.Type.Values)
    body.WriteString(`}`)

    // Stop if body exceeds the API's size limit.
    if body.Len() + len(`]}`) > bodySizeLimit {
        body.Truncate(size)
        events.Postpone()
        break
    }

}

body.WriteString(`]}`)
```

#### Key concepts:

* **Tracking body size**\
  Before adding an event to the request body, the current length of the body is tracked using `body.Len()`. This allows for easy truncation if the body size limit is exceeded.

* **Truncating the body**\
  To ensure the request is valid, the `body.Truncate(n)` method removes the last added event from the body. This prevents the body from exceeding the size limit while maintaining a valid JSON structure.

* **Using `Postpone` to reprocess events**\
  When the body size exceeds the limit:
    - The `Postpone` method is called to notify Meergo that the last processed event has been postponed.
    - The processed events remain unchanged, meaning they can potentially be postponed later.
    - This postponed event will remain unprocessed and will be included in the next call to the `SendEvents` method.

## Error handling

If, during the iteration over the event sequence, an event cannot be processed—for example, because it fails validation—you should call the `Discard` method on the iterator:

```go
n := 0
for event := range events.All() {
    if !valid(event) {
        events.Discard(errors.new("event is invalid"))
    }
    // ...
    n++
}
// Return early if all events have been discarded. 
if n == 0 {
    return nil
}
```

Unlike postponed events, **discarded** events will not be retried in future calls to `SendEvents` or `PreviewSendEvents`.

If a validation error occurs _after_ sending the request to the API, you should return an `EventsError`. This type of error lets you indicate which events failed and why:

```go
// EventsError can be returned by the SendEvents and PreviewSendEvents methods
// of an API connector when one or more events are rejected by the API due to
// validation issues—such as schema mismatches, missing require fields, or
// invalid values. It maps the index of each failed event (starting from 0) to
// the corresponding error.
//
// This error type only reports validation-related failures. Other kinds of
// errors (e.g., network issues or internal failures) may be returned
// separately.
//
// For example, if the third event is rejected due to a validation error while
// all other events are accepted, the returned error would be:
//
// EventsError{2: errors.New("event is invalid")}
type EventsError map[int]error
```

If the error affects all events—such as when the entire request fails—you should return a generic error. In that case, all processed events will be marked with the same error.

### Key concepts:

* **Discarding events during iteration**\
  If an event fails validation before sending, you can discard it during iteration using `events.Discard(err)`. Discarded events are removed from processing entirely and will not be retried in future calls to `SendEvents` or `PreviewSendEvents`.  

* **Handling individual event errors**\
  When certain events fail due to validation issues (e.g., returned by the API), you can return an `EventsError` that maps each failed event to its specific error, instead of returning a single error for the whole batch.

* **Error index mapping**\
  Each key in the `EventsError` type represents the index of a failed event (in the order they were consumed and likely sent in the HTTP request), and each value holds the corresponding error.

### When to validate events

When it comes to event validation, there are a few possible scenarios depending on the target API:

* **The API never returns validation errors** (e.g., Google Analytics's API). In this case, your connector should validate events as much as possible before sending them. This allows users to quickly understand why certain events aren't accepted by the API.

* **The API validates events but only returns a single error in the response** (e.g., Klaviyo's API), typically the first error encountered. In this case, it's important that your connector performs validation ahead of time — otherwise, a validation error on a single event would cause all events in the same request to be marked as invalid. If the API still returns a validation error, you should return a generic error, which will mark all events as invalid.

* **The API validates events and returns a separate error for each invalid event** (e.g., Mixpanel's API). In this case, you can return an `EventsError` that maps each failed event to its corresponding error. However, note that these validation errors won't be visible in the preview (`PreviewSendEvents`), since no actual request is sent during that step.

If your connector relies entirely on the API's validation and doesn't perform any local checks, validation errors **won't** appear in the preview—because the events aren't actually sent. But if the API provides a dedicated validation endpoint, you can call it from `PreviewSendEvents` to simulate the validation step. If such an endpoint is not available, your connector should implement its own validation logic.

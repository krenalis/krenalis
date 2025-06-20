{% extends "/layouts/doc.html" %}
{% macro Title string %}Send Events to Apps{% end %}
{% Article %}

# Send events to apps

Meergo makes it easy to send events to apps that can receive them.

Here's how to get started with setting up your connector to send events:

```go
meergo.RegisterApp(meergo.AppInfo{
    ...
    AsDestination: &meergo.AsAppDestination{
        ...
        Targets: meergo.TargetEvent | meergo.TargetUser,
        ...
    },
    ...
}, New)
```

This piece of code registers your connector, telling Meergo that it's ready to manage events (as well as users) when used as destination. Next, you'll need to implement the `EventSender` interface:

```go
// EventSender is implemented by app connectors that support event sending.
type EventSender interface {

	// EventTypeSchema returns the schema of the specified event type.
	//
	// The returned schema describes properties required by the connector to
	// send an event of this type. Actions based on the specified event type
	// will have a transformation that, given the received event, provides the
	// properties required by the connector. These properties, along with the
	// raw event, are passed to the connector's PreviewSendEvents and SendEvents
	// methods.
	//
	// If no extra information is needed for the event type, the returned schema
	// is the invalid schema. If the event type does not exist, it returns the
	// ErrEventTypeNotExist error.
	EventTypeSchema(ctx context.Context, eventType string) (types.Type, error)

	// EventTypes returns the event types of the connector's instance.
	EventTypes(ctx context.Context) ([]*EventType, error)

	// PreviewSendEvents builds and returns the HTTP request that would be used to
	// send the given events to the app, without actually sending it.
	//
	// If any event type does not exist, it returns the ErrEventTypeNotExist error.
	//
	// Authentication data in the returned request is redacted (i.e., replaced with
	// "[REDACTED]").
	//
	// This method is safe for concurrent use, on the same instance, by multiple
	// goroutines.
	PreviewSendEvents(ctx context.Context, events Events) (*http.Request, error)

	// SendEvents sends a non-empty sequence of events to an app.
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

Sometimes, the event might lack necessary information required for sending to the app. In such cases, the schema of the event type specifies the extra information needed.

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

When sending the event, the `PreviewSendEvents` and `SendEvents` methods receives, as an argument, a sequence of events that should be sent. Each event of the sequence has the `Properties` field with the values of the three parameters "method," "content_type," and "item_id" conforming to the event type schema.

If a field in the schema is mandatory, set the `Required` field in the `types.Property` struct to `true`. Additionally, you can specify a placeholder using the `Placeholder` field for easier mapping compilation.

> When selecting a placeholder, consider that certain property names and traits hold specific meanings and can thus serve as suitable placeholders. Refer to the prefilled properties and traits sections within the events for further details:
>
>    - [page](/events/page#prefilled-properties)
>    - [screen](/events/screen#prefilled-properties)
>    - [track](/events/track#prefilled-properties)
>    - [identify](/events/identify#prefilled-traits)
>    - [group](/events/group#prefilled-traits)

Now, let's move on to sending events to the app using the `EventsRequest` method.

## Send events

Finally, to actually send events to the app, the `SendEvents` method sends the events to the app:

```go
SendEvents(ctx context.Context, events Events) error
```

The parameters are:

- `ctx`: The context.
- `events`: An iterator over the events.

The `events` parameter is a collection of events to send. **You don’t need to process all the events in the collection at once.** Instead, handle only as many as can be sent in a single HTTP request to the application. Even if the application supports processing only one event per request, that's fine. Meergo will automatically call the method again for any events that remain unprocessed.

#### Key concept: processed events

Meergo considers a event processed as soon as it has been read from the `Events` collection. To better understand how this works, let’s first explore the methods provided by the `Events` interface. Afterward, we’ll review how to use these methods effectively in various scenarios.

```go
// Events represents a collection of events to be sent to an app. The collection
// is guaranteed to contain at least one event.
//
// After calling First or once the iterator returned by All or SameUser stops,
// no further method calls on Events are allowed.
type Events interface {

    // All returns an iterator to read all events. Properties of the events in the
    // sequence may be modified unless the event is subsequently skipped.
    All() iter.Seq2[int, *Event]

    // First returns the first event. The event's properties may be modified.
    // After First is called, no further method calls on Events are allowed.
    First() *Event

    // Peek retrieves the next event without advancing the iterator. It returns the
    // event and true if an event is available, or false if there are no further
    // events. The returned event must not be modified.
    Peek() (*Event, bool)

    // SameUser returns an iterator over the events of the same user. Properties of
    // the events in the sequence may be modified unless the event is subsequently
    // skipped.
    SameUser() iter.Seq2[int, *Event]

    // Skip skips the current event in the iteration and marks it as unread. The
    // subsequent iteration will resume at the next event while preserving the same
    // index. Skip may only be called during iterations from All or SameUser, and only
    // if the event's properties have not been modified.
    Skip()
}
```

### Sending one event at a time

If the application can process only one event per request, use the `Events.First` method to retrieve just the first event.

Below is an example implementation:

```go
func (my *MyApp) SendEvents(ctx context.Context, events meergo.Events) error {

    // Read only the first event.
    event := events.First()

    // Prepare request.
    req := &meergo.EventsRequest{
        Method: "POST",
        URL:    "https://api.myapp.com/v1/event",
        Header: http.Header{},
    }

    // Add the headers.
    auth := my.settings.auth
    if redacted {
        auth = "[REDACTED]"
    }
    req.Header.Set("Authorization", "Bearer "+auth)
    req.Header.Set("Content-Type", "application/json")

    // Add the body.
    var body json.Buffer
    body.WriteString(`{"properties":`)
    body.Encode(event.Properties)
    body.WriteString(`}`)

    req.Body = body.Bytes()

	// send the events.
	
    return req, nil
}
```

#### Key concepts:

* **Read the first event**\
  Use `events.First()` to read the first event in the collection.

This method ensures that only one event is processed per request, in accordance with the API's limitations. Meergo will automatically re-invoke the method to handle any remaining events.

### Batch of events from the same user

If the application supports processing multiple events in a batch but requires them to belong to the same user, you can iterate over `events.SameUser()` to retrieve only the events associated with the first event's user.

Below is an example implementation:

```go
func (my *MyApp) SendEvents(ctx context.Context, events meergo.Events) error {

    // Prepare request.
    req := &meergo.EventsRequest{
        Method: "POST",
        URL:    "https://api.myapp.com/v1/events",
        Header: http.Header{},
    }

    // Add the headers.
    auth := my.settings.auth
    if redacted {
        auth = "[REDACTED]"
    }
    req.Header.Set("Authorization", "Bearer "+auth)
    req.Header.Set("Content-Type", "application/json")

    // Add the body.
    var body json.Buffer
    body.WriteString(`{"events":[`)
    for i, event := range events.SameUser() {
        if i > 0 {
            body.WriteByte(',')
        }
        body.WriteString(`{"properties":`)
        body.Encode(event.Properties)
        body.WriteString(`}`)
        if i+1 == bodyMaxEvents {
            break
        }
    }
    body.WriteString(`]}`)

    req.Body = body.Bytes()

    return req, nil
}
```

#### Key concepts:

* **Iterate over events**\
  Use `events.SameUser()` to read only the events belonging to the same user as the first event. This ensures that all events in the batch come from the same user.

* **Batch size limitation**\
  The example demonstrates breaking the loop once the maximum number of events (`bodyMaxEvents`) is reached. This ensures the request complies with the application's API limits.

### Batch of events from mixed users

If the application supports sending multiple events from different users in a single HTTP request, you can iterate over all events using `events.All()`.

Here is an example implementation:

```go
func (my *MyApp) SendEvents(ctx context.Context, events meergo.Events) error {

    // Prepare request.
    req := &meergo.EventsRequest{
        Method: "POST",
        URL:    "https://api.myapp.com/v1/events",
        Header: http.Header{},
    }

    // Add the headers.
    auth := my.settings.auth
    if redacted {
        auth = "[REDACTED]"
    }
    req.Header.Set("Authorization", "Bearer "+auth)
    req.Header.Set("Content-Type", "application/json")

    // Add the body.
    var body json.Buffer
    body.WriteString(`{"events":[`)
    for i, event := range events.All() {
        if i > 0 {
            body.WriteByte(',')
        }
        body.WriteString(`{"properties":`)
        body.Encode(event.Properties)
        body.WriteString(`}`)
        if i+1 == bodyMaxEvents {
            break
        }
    }
    body.WriteString(`]}`)

    req.Body = body.Bytes()

    return req, nil
}
```

#### Key concepts:

* **Iterating over all events**\
  The `events.All()` method iterates over all events, regardless of the user, allowing mixed batches to be processed in a single request.

* **Limit on events**\
  The loop stops once the maximum number of events (`bodyMaxEvents`) is reached, ensuring that the request body size stays within the application's limits.

This approach enables efficient processing of mixed events from different users in a single batch request, reducing the number of API calls needed.

### Handling body size limits

In the previous examples, the loop stops when the number of events reaches the API's maximum limit. However, if the API imposes a body size limit rather than an event count limit, you can use the `Skip` method to skip an event after it has been read. This ensures that the event remains unprocessed and can be included in a subsequent call to the `EventsRequest` method.

Below is an example implementation:

```go
    for i, event := range events.All() {

        // Track length before adding the event.
        n := body.Len()

        if i > 0 {
            body.WriteString(`,`)
        }

        // Build the event JSON object.
        body.WriteString(`{"properties":`)
        body.Encode(event.Properties)
        body.WriteString(`}`)

        // Stop if body exceeds app size limit.
        if body.Len() > bodySizeLimit {
            body.Truncate(n)
            events.Skip()
            break
        }

    }
```

#### Key concepts:

* **Tracking body size**\
  Before adding an event to the request body, the current length of the body is tracked using `body.Len()`. This allows for easy truncation if the body size limit is exceeded.

* **Truncating the body**\
  To ensure the request is valid, the `body.Truncate(n)` method removes the last added event from the body. This prevents the body from exceeding the size limit while maintaining a valid JSON structure.

* **Using `Skip` to reprocess events**\
  When the body size exceeds the limit:
    - The `Skip` method is called to notify Meergo that the last processed event has been skipped.
    - The processed events remain unchanged, meaning they can potentially be skipped later.
    - This skipped event will remain unprocessed and will be included in the next call to the `EventsRequest` method.

### Differentiating errors by event

When the `EventsRequest` method returns an error, the error applies to all consumed events. These events will not be sent to the app and will no longer be available in future `EventsRequest` calls.

However, if you need to report an error for specific events while still returning a request with the other events, you can return both a request and an `EventsError`. The `EventsError` value contains only the events that could not be added to the returned request, along with their corresponding error messages.

The `EventsError` type is a map where:

* The key is the index of the event as it was read from the iterator.
* The value is a string that explains why the event could not be sent to the app.

```go
// EventsError is returned by the EventsRequest method of an app connector when
// some events have been consumed, but they cannot be included in the current or
// any future requests. It maps the event indices to the errors related to those
// events.
//
// The EventsRequest method may return both a request with the events that were
// included in the request and an EventsError error with the events that could
// not be included in any request.
type EventsError map[int]error

func (err EventsError) Error() string {
    var msg string
    for i, e := range err {
        msg += fmt.Sprintf("event %d: %v\n", i, e)
    }
    return msg
}
```

### Key concepts:

* **Error Handling for individual events**\
  Instead of returning a single error for the entire batch, you can return a specific error for each event that fails to be sent. This allows you to pinpoint exactly which event(s) encountered an issue.

* **Mapping errors by event index**\
  In the `EventsError` type, the key represents the index of the event within the iteration, and the value is the error associated with that specific event.

This approach enables you to handle errors individually for each event, rather than dealing with a single error for the entire batch.

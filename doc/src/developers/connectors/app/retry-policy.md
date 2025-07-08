{% extends "/layouts/doc.html" %}
{% macro Title string %}Backoff{% end %}
{% Article %}

# Retry policy

App connectors can use retry policies to manage retries of idempotent HTTP requests based on the app's response status code.

You only need to set up the retry policy when you register the connector. After that, use the provided HTTP client to make calls, and Meergo will handle retries using the policy you set.

Here's an example of how to register a connector with two retry strategies: one that uses the "Retry-After" header for status code 429 (Too Many Requests) and another that uses an exponential backoff strategy for status codes 500 (Internal Server Error) and 503 (Service Unavailable).

```go
meergo.RegisterApp(meergo.AppInfo{
    RetryPolicy: meergo.RetryPolicy{
        "429":     meergo.RetryAfterStrategy(),
        "500 503": meergo.ExponentialStrategy(meergo.NetFailure, 200 * time.Millisecond),
    },
    // other fields are omitted.
}, New)
```

The `RetryPolicy` field specifies the retry policy for the connector. It maps one or more HTTP status codes to the corresponding retry strategy. If an idempotent HTTP request fails, Meergo will look up the status code in the connector policy and retry the request using the corresponding strategy.

You can use the strategies provided by Meergo, so you do not have to implement it, or create your own. If the app documentation does not specify how to handle errors, do not set a retry policy. Meergo will use a default policy in that case.

## Retriable requests

Only idempotent requests can be retried. The `Do` method of the HTTP client passed to a connector decides whether a request is retriable using the same rules as Go's standard library (see [http.Transport](https://pkg.go.dev/net/http#Transport)).

A request is considered retriable if:

* it is **idempotent**, and
* it either **has no body** or provides a `Request.GetBody` function to recreate the body if needed.

A request is idempotent if:

* its HTTP method is `GET`, `HEAD`, `OPTIONS`, or `TRACE`, or
* it includes the header `Idempotency-Key` or `X-Idempotency-Key`.

If the idempotency header is present but is empty (nil or empty slice), the request is still treated as idempotent — but the header won't be sent over the network.

### Example

If your app supports idempotency and requires an idempotency key in the header called `Idempotency-Key`, you can do this:

```go
// Mark the request as idempotent.
req.Header.Set("Idempotency-Key", key)
req.GetBody = func() (io.ReadCloser, error) {
    return io.NopCloser(bytes.NewReader(body)), nil
}
```

The `Request.GetBody` function is called if the request needs to be retried and must return the original request body.

If you want a random UUID to use as the idempotency key, you can use:  

```go
// Mark the request as idempotent.
req.Header.Set("Idempotency-Key", meergo.UUID())
req.GetBody = func() (io.ReadCloser, error) {
    return io.NopCloser(bytes.NewReader(body)), nil
}
```

### Idempotent events

Events are usually idempotent, meaning apps don't require an idempotency key to receive them. Instead, an app generally uses each event's unique ID for deduplication.

However, even if an idempotency key is not needed, the request must still be marked as idempotent. You can mark the request as idempotent without sending the header like this:

```go
// Mark the request as idempotent.
req.Header["Idempotency-Key"] = nil
req.GetBody = func() (io.ReadCloser, error) {
    return io.NopCloser(bytes.NewReader(body)), nil
}
```

Setting the header value to `nil` tells the HTTP client that the request is idempotent and should be retried on failure, but the header itself will not be sent to the app.

## Retry strategies

Meergo offers Constant, Exponential, RetryAfter, and Header strategies for managing retries. Jitter is automatically added to the wait time calculated by strategies to introduce variability.

### Constant strategy

This strategy waits a fixed amount of time before retrying. For example, to wait 1 second:

```go
meergo.ConstantStrategy(meergo.NetFailure, time.Second)
```

### Exponential strategy

This strategy increases the wait time exponentially, starting from a base value. For example, to start with 100ms:

```go
meergo.ExponentialStrategy(meergo.Slowdown, 100 * time.Millisecond)
```

### RetryAfter strategy

This strategy uses the "Retry-After" header from the response. The header can specify either the number of seconds to wait or a specific date and time (as specified in [RFC 9110](https://httpwg.org/specs/rfc9110.html#field.retry-after)). The strategy returned by the `RetryAfter` function handles both:

```go
meergo.RetryAfterStrategy()
```

The strategy returned by `RetryAfterStrategy` also supports fractional seconds.

### Header strategy

This strategy reads the wait time from a specific header. The `HeaderStrategy` function takes a header name and a function to parse the header value and returns the strategy. For example, to use the "Wait-Seconds" header and parse it as a duration:

```go
meergo.HeaderStrategy(meergo.Slowdown, "Wait-Seconds", time.ParseDuration)
```

The `parse` function has the following type:

```go
func(s string) (time.Duration, error)
```

If the `parse` function returns an error, the request will not be retried. If `parse` is `nil`, the strategy defaults to the behavior of the RetryAfter strategy.


### Custom strategy

To create a custom strategy, implement a function with the `meergo.BackoffStrategy` type:

```go
type RetryStrategy func(res *http.Response, retries int) (reason meergo.FailureReason, waitTime time.Duration)
```

This function takes the failed response and the number of retries so far, and returns a failure reason and time to wait before retrying. Parameters include: 

- `res`: The HTTP response from the app.
- `retries`: The number of times the request has been retried, starting from 0.
- `reason`: The reason of the failure.
- `waitTime`: The amount of time to wait before retrying.

Do not add jitter to the wait time; it is added automatically.

### Failure reasons

The failure reason determines whether a request is eligible for retry and how it affects the rate control system: 

* `meergo.PermanentFailure`: the request cannot be retried, and the error contributes to the app’s error rate.
* `meergo.NetFailure`: the request can be retried, and the error still counts toward the error rate.
* `meergo.Slowdown`: the request can be retried. Upon receiving the first `Slowdown`, request rate is significantly reduced; further slowdowns also contribute to the error rate.
* `meergo.RateLimited`: the request can be retried, but future requests are suspended until the returned wait time has passed.

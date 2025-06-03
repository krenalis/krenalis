{% extends "/layouts/doc.html" %}
{% macro Title string %}Backoff{% end %}
{% Article %}

# Backoff

App connectors can use backoff policies to manage retries of idempotent HTTP requests based on the app's response status code.

You only need to set up the backoff policy when you register the connector. After that, use the provided HTTP client to make calls, and Meergo will handle retries using the policy you set.

Here's an example of how to register a connector with two backoff strategies: one that uses the "Retry-After" header for status code 429 (Too Many Requests) and another that uses an exponential strategy for status codes 500 (Internal Server Error) and 503 (Service Unavailable).

```go
meergo.RegisterApp(meergo.AppInfo{
    BackoffPolicy: meergo.BackoffPolicy{
        "429":     meergo.RetryAfterStrategy(),
        "500 503": meergo.ExponentialStrategy(200 * time.Millisecond),
    },
    // other fields are omitted.
}, New)
```

The `BackoffPolicy` field specifies the backoff policy for the connector. It maps one or more HTTP status codes to the corresponding retry strategy. If an idempotent HTTP request fails, Meergo will look up the status code in the connector policy and retry the request using the corresponding strategy.

You can use the strategies provided by Meergo, so you do not have to implement it, or create your own. If the app documentation does not specify how to handle errors, do not set a backoff policy. Meergo will use a default policy in that case.

## Idempotency

As mentioned earlier, only idempotent requests can be retried. The `Do` method of the provided HTTP client automatically treats GET, PUT, DELETE, and HEAD requests as idempotent. If you need to explicitly specify idempotency, use the `DoIdempotent` method. It remains your responsibility to ensure that the request is idempotent. When you need to generate an idempotency key for an HTTP request, you can use the `meergo.UUID` function, which returns a random version 4 UUID.

## Strategies

Meergo offers Constant, Exponential, RetryAfter, and Header strategies for managing retries. Jitter is automatically added to the wait time calculated by strategies to introduce variability.

### Constant strategy

This strategy waits a fixed amount of time before retrying. For example, to wait 1 second:

```go
meergo.ConstantStrategy(time.Second)
```

### Exponential strategy

This strategy increases the wait time exponentially, starting from a base value. For example, to start with 100ms:

```go
meergo.ExponentialStrategy(100 * time.Millisecond)
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
meergo.HeaderStrategy("Wait-Seconds", time.ParseDuration)
```

The `parse` function has the following type:

```go
func(s string) (time.Duration, error)
```

If the `parse` function returns an error, the request will not be retried. If `parse` is `nil`, the strategy defaults to the behavior of the RetryAfter strategy.


### Custom strategy

To create a custom strategy, implement a function with the `meergo.BackoffStrategy` type:

```go
type BackoffStrategy func(res *http.Response, retries int) (waitTime time.Duration, err error)
```

This function takes the failed response and the number of retries so far, and returns the time to wait before retrying. Parameters include: 

- `res`: The HTTP response from the app.
- `retries`: The number of times the request has been retried, starting from 0.
- `waitTime`: The amount of time to wait before retrying.
- `err`: If the request should not be retried, it is `meergo.NoRetry`, otherwise `nil`.

Do not add jitter to the wait time; it is added automatically.

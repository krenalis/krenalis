# Backoff

App connectors can use backoff policies to manage retries of idempotent HTTP requests based on the app's response status code.

You only need to set up the backoff policy when you register the connector. After that, use the HTTP client provided to make calls, and Meergo will handle retries using the policy you set.

Here's an example of how to register a connector with two backoff strategies: one that uses the "Retry-After" header for status code 429 (Too Many Requests) and another that uses an exponential strategy for status codes 500 (Internal Server Error) and 503 (Service Unavailable).

```go
meergo.RegisterApp(meergo.AppInfo{
    Backoff:   map[string]meergo.Backoff{
		"429":     meergo.RetryAfterBackoff(),
        "500 503": meergo.ExponentialBackoff(200 * time.Millisecond),
    },
    // other fields are omitted.
}, New)
```

The `AppInfo.Backoff` field is a `map[string]meergo.Backoff`. If an idempotent HTTP request fails, Meergo will look up the status code in the connector policy and retry the request using the corresponding backoff strategy.

You can use the strategies provided by Meergo, so you do not have to implement it, or create your own. If the app documentation does not specify how to handle errors, do not set a backoff policy. Meergo will use a default policy in that case.

## Idempotency

As mentioned earlier, only idempotent requests can be retried. The `Do` method of `meergo.HTTPClient` automatically treats GET, PUT, DELETE, and HEAD requests as idempotent. Use the `DoIdempotent` method if you need to specify idempotency explicitly.

If your application supports idempotency, you can use the `UUID` method of `meergo.HTTPClient` to generate a unique version 4 UUID to use as an idempotency key.

## Strategies

Meergo provides several built-in backoff strategies that you can use to manage retries. These strategies include a random jitter added to the calculated duration.

### Constant Strategy

This strategy waits a fixed amount of time before retrying. For example, to wait 1 second:

```go
meergo.ConstantBackoff(1 * time.Seconds)
```

### Exponential Strategy

This strategy increases the wait time exponentially, starting from a base value. For example, to start with 100ms:

```go
meergo.ExponentialBackoff(100 * time.Milliseconds)
```

### RetryAfter Strategy

This strategy uses the "Retry-After" header from the response. The header can specify either the number of seconds to wait or a specific date and time (as specified in [RFC 9110](https://httpwg.org/specs/rfc9110.html#field.retry-after)). The strategy returned by the `RetryAfterBackoff` function handles both:  

```go
meergo.RetryAfterBackoff()
```

The strategy returned by the `RetryAfterBackoff` function also supports fractional seconds.

### Header Strategy

This strategy reads the wait time from a specific header. The `HeaderBackoff` function takes a header name and a function to parse the header value and returns the strategy. For example, to use the "Wait-Seconds" header and parse it as a duration:

```go
meergo.HeaderAfter("Wait-Seconds", time.ParseDuration)
```

The `parse` function has the following type:

```go
func (s string) (time.Duration, error)
```

If the `parse` function returns an error, the request will not be retried. If no `parse` function is provided, the strategy defaults to the behavior of the RetryAfter strategy.


### Custom Strategy

To create a custom backoff strategy, implement a function with the `meergo.Backoff` type:

```go
type Backoff func(res *http.Response, retries int) (waitTime time.Duration, err error)
```

This function takes the failed response and the number of retries so far, and returns the time to wait before retrying. Parameters include: 

- `res`: The HTTP response from the app.
- `retries`: The number of times the request has been retried, starting from 0.
- `waitTime`: The amount of time to wait before retrying.
- `err`: If the request should not be retried, it is `meergo.NoRetry`, otherwise `nil`.

For custom strategies, jitter is not automatically added, so you need to include it in your function. If `d` is the calculated wait time, you can add jitter with the following code before returning the value:

```go
d += time.Duration(float64(d) * rand.Float64() * 0.5)
```

This code multiplies the calculated duration by a random factor (between 0 and 0.5) to introduce variability.

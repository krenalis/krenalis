{% extends "/layouts/doc.html" %}
{% macro Title string %}OAuth{% end %}
{% Article %}

# OAuth

For applications that need OAuth authentication to access an account's information, Meergo provides an OAuth implementation for connectors. The connector only needs to provide some information during registration and implement a method to retrieve the app's account that was connected through authentication.

After that, you just need to use the HTTP client provided to the constructor to make HTTP calls to the app. Meergo will automatically handle adding authentication information and keep it updated.

The following example shows how the HubSpot connector provides some OAuth-specific information about HubSpot during registration:

```go
meergo.RegisterApp(meergo.AppInfo{
    OAuth:   meergo.OAuth{
        AuthURL:           "https://app-eu1.hubspot.com/oauth/authorize",
        TokenURL:          "https://api.hubapi.com/oauth/v1/token",
        SourceScopes:      []string{"crm.objects.contacts.read", "crm.schemas.contacts.read"},
        DestinationScopes: []string{"crm.objects.contacts.read", "crm.objects.contacts.write", "crm.schemas.contacts.read"},
    },
    // other fields are omitted.
}, New)
```

The `meergo.OAuth` type contains this information:

- `AuthURL`: authorization endpoint, the URL where users are redirected to grant consent.
- `TokenURL`: token endpoint, the URL used to retrieve the access token, refresh token, and token lifetime.
- `SourceScopes`: required scopes when used as a source. Leave empty if there are no source scopes.
- `DestinationScopes`: required scopes when used as a destination. Leave empty if there are no destination scopes.
- `ExpiresIn`: lifetime of the access token in seconds. If the value is zero or negative, the lifetime is provided by the `TokenURL` endpoint.

If `AuthURL` and `TokenURL` contain query string arguments, they will be preserved.

## OAuthAccount method

```go
OAuthAccount(ctx context.Context) (string, error)
```

The `OAuthAccount` method, part of the `AppOAuth` interface, is called by Meergo after a successful OAuth authentication to obtain the app's account associated with the given authorization. The connector uses the HTTP client provided to the constructor to call the app's API to obtain the account. The account must be a non-empty UTF-8 encoded string.

## ClientSecret and AccessToken

If you need the client secret or the access token, they can be obtained from the `ClientSecret` and `AccessToken` methods of the HTTP client.
{% extends "/layouts/doc.html" %}
{% macro Title string %}SDK Connectors{% end %}
{% Article %}

# SDKs

SDK connectors are paired with an SDK that can be used within an application to send events to Meergo. For example, the JavaScript connector is paired with the [JavaScript SDK](/sources/javascript-sdk), which can be integrated into a website or web application to collect and send events to Meergo.

An SDK connector does not need to implement any specific methods. All event handling and transmission logic resides in the SDK paired with the connector. Therefore, if you develop an SDK connector, you must also implement the corresponding SDK to be used in the applications it targets.

Like other types of connectors, SDK connectors are written in Go. A connector is a Go module that implements specific functions and methods.

## Quick start

In the creation of a new Go module, for your SDK connector, you can utilize the following template by pasting it into a Go file. Not all methods in the file need to be implemented; see below for descriptions of individual methods. Customize the template with your desired package name, type name, and pertinent connector information:

```go
// Package javascript implements a connector for JavaScript.
package javascript

import (
    "github.com/meergo/meergo"
)

func init() {
    meergo.RegisterSDK(meergo.SDKSpec{
        Code:                "javascript",
        Label:               "JavaScript",
        Categories:          meergo.CategorySDKAndAPI,
        Strategies:          true,
        FallbackToRequestIP: true,
        Documentation: meergo.ConnectorDocumentation{
            Source: meergo.ConnectorRoleDocumentation{
                Summary:  "Import events and users using JavaScript",
            },
       },
    }, New)
}

type JavaScript struct {
    // Your connector fields.
}

// New returns a new connector instance for JavaScript.
func New(env *meergo.SDKEnv) (*JavaScript, error) {
    // ...
}
```

## Implementation

Let's explore how to implement a SDK connector, for example for JavaScript.

First create a Go module:

```sh
$ mkdir javascript
$ cd javascript
$ go mod init javascript
```

Then add a Go file to the new directory. For example copy the previous template file.

Later on, you can [build an executable with your connector](/installation/from-source#building-using-the-go-tools).

### About the connector

The `SDKSpec` type describes the specification of the SDK connector:

- `Code`: unique identifier in kebab-case (`a-z0-9-`), e.g. "javascript", "android", "dotnet".
- `Label`: display label shown in the Admin console, typically the name of the language or platform (e.g. "JavaScript", "Android", ".NET").
- `Categories`: the categories that the connector falls into. There must be at least one category.
- `Strategies`: whether this connector supports user strategies.
- `FallbackToRequestIP`: whether to use the request IP as the event IP if `context.ip` was not provided.

This information is passed to the `RegisterSDK` function that, executed during package initialization, registers the SDK connector:

```go
func init() {
    meergo.RegisterSDK(meergo.SDKSpec{
        Code:                "javascript",
        Label:               "JavaScript",
        Categories:          meergo.CategorySDKAndAPI,
        Strategies:          true,
        FallbackToRequestIP: true,
        Documentation: meergo.ConnectorDocumentation{
            Source: meergo.ConnectorRoleDocumentation{
                Summary:  "Import events and users using JavaScript",
            },
       },
    }, New)
}
```

### Constructor

The second argument supplied to the `RegisterSDK` function is the function utilized for creating an SDK instance:

```go
func New(env *meergo.SDKEnv) (*JavaScript, error)
```

This function accepts an SDK environment and yields a value representing your custom type.

The structure of `SDKEnv` is defined as follows:

```go
// SDKEnv is the environment for an application SDK connector.
type SDKEnv struct {

    // Settings holds the raw settings data.
    Settings []byte

    // SetSettings is the function used to update the settings.
    SetSettings SetSettingsFunc
}
```

- `Settings`: Contains the instance settings in JSON format. Further details on how the connector defines its settings will be discussed later.
- `SetSettings`: A function that enables the connector to update its settings as necessary.

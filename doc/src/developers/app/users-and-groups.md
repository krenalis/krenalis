# Users and Groups

An app connector, if the related app allows it, can read, create, and update users and groups within the app, enabling Chichi to import and export users and groups. An app connector may support users only, groups only, or both.

Firstly, include the `Users` and `Groups` flags during connector registration, based on what the connector supports, as the target:

```go
chichi.RegisterApp(chichi.AppInfo{
    ...
    Targets: chichi.Events | chichi.Users | chichi.Groups,
    ...
})
```

After that, to read records from an app, the connector must implement the `Records`, `Create`, and `Update` methods. These methods take the target they should operate on as an argument, which can be either `Users` or `Groups`. They should only implement the targets that the connector supports.

Here, we'll use the term "records" to refer to both users and groups interchangeably.

Let's start by looking at how to read records using the `Records` method.

## Read Records

Chichi calls the connector's `Records` method to read records from the app:

```go
Records(ctx context.Context, target chichi.Targets, properties []string, cursor chichi.Cursor) (records []chichi.Record, next string, err error)
```

The parameters for this method are:

- `ctx`: The context, which is always non-nil.

- `target`: Specifies whether user or group records should be returned. It can be either `Users` or `Groups`.

- `properties`: Contains the names of the properties that must be returned for each record. The names correspond to the properties of the schema as returned by the `Schema` method.

- `cursor`: Indicates from which record to start returning records.

> Normally, the `Records` method would return at least one record if there are no errors. However, it is permissible to return no records even in the absence of errors, enhancing flexibility in handling different situations.

First, let's examine the structure of a single returned record. Then, we'll explore how the `Records` method can return records incrementally, rather than all at once, by utilizing the `cursor` input parameter and the `next` output parameter.

The `Record` type is defined as follows:

```go
type Record struct {
    ID           string
    Properties   map[string]any
    UpdatedAt    time.Time
	Associations []string
	Err          error
}
```

- `ID`: The record's identifier in the app. It must be a valid, non-empty UTF-8 string.

- `Properties`: The record's properties and their values. All requested properties must be present; additional properties are not considered. The values of the requested properties should conform to their respective schema, as returned by the connector's `Schema` method.

- `UpdatedAt`:  The date and time the record was last updated. It can have any time zone. If the date is unknown, return `time.Zero`.

- `Associations`: Identifiers of the groups the user belongs to, if the record refers to a user, or identifiers of the users that belong to the group. If none exist, or if the app only supports users or groups, indicate `nil` or an empty slice.

- `Err`: Any error that occurred while reading the record. It must be `io.EOF` if there are no more records to read beyond those returned. If `Err` is different from `nil` and is not `io.EOF`, then only the `ID` field, along with `Err`, is significant.

If a record encounters an error, meaning `Record.Err` is not `nil`, the import process is not halted but continues with subsequent records.

### How to Use the Cursor

The `Records` method does not necessarily have to return all records, instead, it should return only the records that it can return with a single call to the app.

During an import, Chichi calls the method multiple times until all records have been returned, i.e., until `io.EOF` is returned. To allow `Records` to return all records over multiple calls, Chichi passes a cursor as an argument, defined by the `Cursor` type:

```go
type Cursor struct {
    ID        string
    UpdatedAt time.Time
    Next      string
}
```

For the first call of a **complete** import (e.g., the first import after creating the import action), the cursor is the zero value of `Cursor`:

```go
Cursor{
	ID: "",
	UpdatedTime: time.Zero,
	Next: "",
}
```

For subsequent calls in the same import process, `ID` and `UpdatedAt` are the `ID` and `UpdatedAt` fields of the last record returned, while `Next` is the value of `next` returned by the last successful call.

As a special case, the first call of an **incremental** import, unlike a complete import, receives a cursor with `ID` and `UpdatedAt` being those of the last record returned from the previous import. This way, the import can resume from where the previous one ended.

> For apps that do not return a "next" value in the response to use for reading the next records, the connector can still rely on the cursor's `ID` and `UpdatedAt` fields.

### Making HTTP Calls to the App

When a connector instance is created, an HTTP client is passed to the constructor through the `AppConfig.HTTPClient` field. This client should be used by the connector's methods to make HTTP calls to the app. It takes care of:

- Retrying calls in case of an error, if the request allows for it.
- Proper resource management.
- Adding the "Authorization" HTTP header for connectors that use OAuth.
- Refreshing the access token for connectors that use OAuth.

The client implements the following interface:

```go
type HTTPClient interface {

	// Do sends an HTTP request with an Authorization header if required.
	Do(req *http.Request) (res *http.Response, err error)

	// ClientSecret returns the OAuth client secret of the HTTP client.
	ClientSecret() (string, error)

	// AccessToken returns an OAuth access token.
	AccessToken(ctx context.Context) (string, error)
}
```

If you need to make direct HTTP calls without using the provided client, the `ClientSecret` and `AccessToken` methods can be used by OAuth connectors to obtain the client secret and an access token for authentication with the app.

If a method from `HTTPClient` returns an error, connector methods should return that exact error, without any modification or wrapping.

## Create Records

To create a record, Chichi invokes the connector's `Create` method:

```go
Create(ctx context.Context, target chichi.Targets, properties map[string]any) error
```

This is called during export when a new user or group should be created in the app. `target` can either be `Users` or `Groups`, limited to the supported targets by the connector.

The `properties` parameter specifies the properties for the new record to be created, adhering to the schema provided by the connector's `Schema` method. Note that `properties` is always populated; it is never empty.

The `Create` method can use the HTTP client passed to the constructor for making HTTP calls to the app.

## Update Records

To update a record, Chichi invokes the connector's `Update` method:

```go
Update(ctx context.Context, target chichi.Targets, id string, properties map[string]any) error
```

This is called during export when an app's user or group needs to be updated. `target` can either be `Users` or `Groups`, limited to the supported targets by the connector.

The `properties` parameter represents the properties to update, with unchanged properties not being present. The properties' values follow the schema returned by the connector's `Schema` method. Note that `properties` is always populated; it is never empty.

The `Update` method can use the HTTP client passed to the constructor to do HTTP calls to the app.

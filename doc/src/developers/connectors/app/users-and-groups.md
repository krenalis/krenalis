# Users and Groups

An app connector, if the related app allows it, can read, create, and update users and groups within the app, enabling Meergo to import and export users and groups. An app connector may support users only, groups only, or both.

Firstly, include the `Users` and `Groups` flags during connector registration, based on what the connector supports, as the target:

```go
meergo.RegisterApp(meergo.AppInfo{
    ...
    Targets: meergo.Events | meergo.Users | meergo.Groups,
    ...
})
```

After that, to read, update, and create app records, the connector must implement the `Records` and `Upserts` methods. These methods take the target they should operate on as an argument, which can be either `Users` or `Groups`. They should only implement the targets that the connector supports.

Here, we'll use the term "records" to refer to both users and groups interchangeably.

Let's start by looking at how to read records using the `Records` method.

## Read Records

Meergo calls the connector's `Records` method to read records from the app:

```go
Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string) ([]meergo.Record, string, error)
```

The parameters for this method are:

- `ctx`: The context, which is always non-nil.
- `target`: Specifies whether user or group records should be returned. It can be either `Users` or `Groups`.
- `ids`: Identifiers of the records to return. If `nil`, `Records` returns all records.
- `lastChangeTime`: If not the zero time, return only the records that were created or modified at or after. The precision of `lastChangeTime` is limited to microseconds.
- `properties`: Contains the names of the properties that must be returned for each record. The names correspond to the properties of the schema as returned by the `Schema` method.
- `cursor`: Indicates the starting position for reading records. This is the cursor value from a previous call in a paginated query. For the first call, it is empty.

> Normally, the `Records` method would return at least one record if there are no errors. However, it is permissible to return no records even in the absence of errors, enhancing flexibility in handling different situations.

First, let's examine the structure of a single returned record. Then, we'll explore how the `Records` method can return records incrementally, rather than all at once, by utilizing the `cursor` input parameter and the `next` output parameter.

The `Record` type is defined as follows:

```go
type Record struct {
    ID             string
    Properties     map[string]any
    LastChangeTime time.Time
	Associations   []string
	Err            error
}
```

- `ID`: The record's identifier in the app. It must be a valid, non-empty UTF-8 string.
- `Properties`: The record's properties and their values. Additional properties not requested are not considered. The connector may omit a property for a user if that user does not have that property. This is distinct from a property with a `null` value. The values of requested properties should conform to their respective schemas, as returned by the connector's `Schema` method. If a requested property is always returned, its `Required` field in the schema must be `true`; if it may not be returned for some users, it must be `false`.
- `LastChangeTime`:  The date and time the record was last changed. It can have any time zone. The precision of this time is limited to microseconds; any precision beyond microseconds will be truncated. If the date is unknown, return the zero time `time.Time{}`.
- `Associations`: Identifiers of the groups the user belongs to, if the record refers to a user, or identifiers of the users that belong to the group. If none exist, or if the app only supports users or groups, indicate `nil` or an empty slice.
- `Err`: Any error that occurred while reading the record. It must be `io.EOF` if there are no more records to read beyond those returned. If `Err` is different from `nil` and is not `io.EOF`, then only the `ID` field, along with `Err`, is significant.

If a record encounters an error, meaning `Record.Err` is not `nil`, the import process is not halted but continues with subsequent records.

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

## Updating and Creating Records

To update or create a record, Meergo uses the connector's `Upsert` method:

```go
Upsert(ctx context.Context, target meergo.Targets, records []meergo.UpsertRecord) ([]int, error)
```

This method is called during an export when users or groups need to be updated or when new users or groups need to be created in the application. The `target` parameter can be either `Users` or `Groups`, depending on what the connector supports.

The `records` parameter contains the records to be updated or created, with the following structure:

```go
type UpsertRecord struct {
	ID         string
	Properties map[string]any
}
```

- **ID**: This is the identifier of the user or group in the app to be updated. It is left empty when creating a new record.
- **Properties**: These are the properties of the record to be created or updated, following the schema provided by the connector's Schema method. Note that Properties always contain at least one property.

The method does not need to process all records in a single call. It must process at least the first record and can process additional records based on the app’s API capabilities. For example, if the app’s API only supports updating one record at a time, the `Upsert` method will handle only the first record in each call. If updates and creations need to be handled separately, and the first record is for updating, the method may process all update records together.

The `Upsert` method returns a slice of integers representing the indexes of the processed records, along with an error related to these records. Notably, index `0`, which corresponds to the first record, can be omitted from the returned slice. If the only record processed is the first one, the method can return `nil` instead of `[]int{0}`.

If not all records are processed, Meergo will call `Upsert` again with the remaining records and any additional records as necessary.

The `Upsert` method can use the HTTP client provided during construction to make HTTP requests to the application.
{% extends "/layouts/doc.html" %}
{% macro Title string %}Users and Groups{% end %}
{% Article %}

# Users and groups

An app connector, if the related app allows it, can read, create, and update users and groups within the app, enabling Meergo to import and export users and groups. An app connector may support users only, groups only, or both.

Firstly, include the `Users` and `Groups` flags during connector registration, based on what the connector supports, as the targets for source and destination:

```go
meergo.RegisterApp(meergo.AppInfo{
	...
	AsSource: &meergo.AsAppSource{
		...
		Targets:  meergo.Users,
		...
	},
	AsDestination: &meergo.AsAppDestination{
		...
		Targets:  meergo.Users,
		...
	},
	...
}, New)
```

After that, to read, update, and create app records, the connector must implement the `Records` and `Upserts` methods. These methods take the target they should operate on as an argument, which can be either `Users` or `Groups`. They should only implement the targets that the connector supports.

Here, we'll use the term "records" to refer to both users and groups interchangeably.

Let's start by looking at how to read records using the `Records` method.

## Read records

Meergo calls the connector's `Records` method to read records from the app:

```go
Records(ctx context.Context, target meergo.Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]meergo.Record, string, error)
```

The parameters for this method are:

- `ctx`: The context.
- `target`: Specifies whether user or group records should be returned. It can be either `Users` or `Groups`.
- `lastChangeTime`: If not the zero time, return only the records that were created or modified at or after. The precision of `lastChangeTime` is limited to microseconds.
- `ids`: Identifiers of the records to return. If `nil`, `Records` should return all records.
- `properties`: Contains the names of the properties that must be returned for each record.
- `cursor`: Indicates the starting position for reading records. This is the cursor value from a previous call in a paginated query. For the first call, it is empty.
- `schema`: It is a recent user or group source schema, as returned by the `Schema` method.


> Typically, the `Records` method returns at least one record if there are no errors. However, it is valid for it to return no records even when there are no errors. Additionally, the `Records` method may return duplicate records (i.e., records with the same ID), but only the first record in such cases will be processed.

First, let's examine the structure of a single returned record. Then, we'll explore how the `Records` method can return records incrementally, rather than all at once, by utilizing the `cursor` input parameter and the `next` output parameter.

The `Record` type is defined as follows:

```go
type Record struct {
	ID             string
	Properties     map[string]any
	LastChangeTime time.Time
	Associations   []string
	Err            error // Not used by the Records method.
}
```

- `ID`: The record's identifier in the app. It must be a valid, non-empty UTF-8 string.
- `Properties`: The record's properties and their values. Additional properties not requested are not considered. The connector may omit a property for a user if that user does not have that property. This is distinct from a property with a `null` value.
- `LastChangeTime`:  The date and time the record was last changed. It can have any time zone. The precision of this time is limited to microseconds; any precision beyond microseconds will be truncated. If the date is unknown, return the zero time `time.Time{}`.
- `Associations`: Identifiers of the groups the user belongs to, if the record refers to a user, or identifiers of the users that belong to the group. If none exist, or if the app only supports users or groups, indicate `nil` or an empty slice.

If an error occurs, `Records` must return a non-nil error, and it should not be an EOF error. 

### Making HTTP calls to the app

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

## Updating and creating records

To update or create a record, Meergo uses the connector's `Upsert` method:

```go
Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error
```

This method is used during an export process to update existing users or groups, or to create new users or groups in the application. The `target` parameter specifies whether the operation applies to **Users** or **Groups**, depending on what the connector supports.

The `records` parameter contains a collection of items to update or create. **You don’t need to process all the records in the collection at once.** Instead, only handle as many as you can send in a single HTTP request to the application. Even if the application supports processing only one record per request, that's fine. Meergo will automatically call the method again for any records that remain unprocessed.

#### Key concept: processed records

Meergo considers a record processed as soon as it has been read from the `Records` collection. To better understand how this works, let’s first explore the methods provided by the `Records` interface. Afterward, we’ll review how to use these methods effectively in various scenarios.

```go
// Records represents a collection of records to be created or updated. A record
// to be created has an empty ID. The collection is guaranteed to contain at
// least one record.
//
// After calling First or once the iterator returned by All or Same stops, no
// further method calls on Records are allowed.
type Records interface {

	// All returns an iterator to read all records. Properties of the records in the
	// sequence may be modified unless the record is subsequently skipped.
	All() iter.Seq2[int, Record]

	// First returns the first record. The record's properties may be modified.
	// After First is called, no further method calls on Records are allowed.
	First() Record

	// Peek retrieves the next record without advancing the iterator. It returns the
	// record and true if a record is available, or false if there are no further
	// records. The returned record must not be modified.
	Peek() (Record, bool)

	// Same returns an iterator for records: either all records to update
	// (if the first record is for update) or all records to create
	// (if the first record is for creation). Properties of the records in the
	// sequence may be modified unless the record is subsequently skipped.
	Same() iter.Seq2[int, Record]

	// Skip skips the current record in the iteration and marks it as unread. The
	// subsequent iteration will resume at the next record while preserving the same
	// index. Skip may only be called during iterations from All or Same, and only
	// if the record's properties have not been modified.
	Skip()
}

```

#### Sending one record at a time

The most common scenario involves an application whose API can handle only one record (user or group) per request. In this case, you should use the `First` method of `Records` to read only the first record.

Below is an example implementation:

```go
func (my *MyApp) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	// Read the first record.
	record := records.First()

	// Prepare request body.
	var body bytes.Buffer
	body.WriteString(`{"properties":`)
	json.Encode(&body, record.Properties)
	body.WriteString(`}`)

	// Prepare the method for update (PUT) or create (POST).
	method := http.MethodPut
	if record.ID == "" {
		method = http.MethodPost
	}

	// Prepare the path.
	path := "/v1/customers"
	if method == http.MethodPost {
		path += "/" + url.PathEscape(record.ID)
	}

	// Create the HTTP request.
	req, _ := http.NewRequestWithContext(ctx, method, "https://api.myapp.com"+path, &body)

	// Send the HTTP request.
	res, err := my.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check the response status code.
	if res.StatusCode != 200 {
		return fmt.Errorf("app server response: %s", res.Status)
	}

	return nil
}
```

#### Key concepts:

* **Read the First Record**\
   Use `records.First()` to read the first record that needs to be processed.

* **Determine the Type of Operation**\
   If `record.ID` is empty, the record should be created; otherwise, it should be updated.

This method ensures that only one record is processed per request, aligning with the API's limitations. Meergo will automatically re-invoke the method for unread records.

### Batch of records of the same type (update or create)

If the application allows processing multiple records in a batch but requires them to be of the same type (either all updates or all creates), you can use the `Peek` method to peek the first record without consuming it. This helps you determine whether the batch should execute an update or a create operation. After that, you can iterate over `records.Same()` to read only records of the same type as the first record.

Below is an example implementation:

```go
func (m *MyApp) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	// Peek at the first record to determine the type of request.
	record, _ := records.Peek()
	method := http.MethodPut
	if record.ID == "" {
		method = http.MethodPost
	}

	// Prepare request body.
	var body bytes.Buffer
	body.WriteString(`{"customers":[`)
	for i, record := range records.Same() {
		if i > 0 {
			body.WriteString(`,`)
		}
		body.WriteString(`{`)
		if record.ID != "" {
			body.WriteString(`"id":`)
			json.Encode(&body, record.ID)
			body.WriteString(`,`)
		}
		body.WriteString(`"properties":`)
		json.Encode(&body, record.Properties)
		body.WriteString(`}`)
		if i+1 == bodyMaxRecords {
			break
		}
	}
	body.WriteString(`]}`)

	// Create the HTTP request.
	req, _ := http.NewRequestWithContext(ctx, method, "https://api.myapp.com/v1/customers/batch", &body)

	// Send the HTTP request.
	res, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check the response status code.
	if res.StatusCode != 200 {
		return fmt.Errorf("server responded with status %s", res.Status)
	}

	return nil
}
```

#### Key concepts:

* **Determine the Type of Operation**\
   Use `records.Peek()` to examine the first record without consuming it.
   - If the record has an `ID`, it's an update.
   - If `ID` is empty, it's a creation.

   Do not use the `records.First()` in this scenario, as it consumes the record and prevents any other methods from being called.

* **Iterate Over Records**\
   Use `records.Same()` to read only records of the same type as the first one. This ensures all records in the batch are valid for the chosen operation.

* **Batch Size Limitation**\
   The example demonstrates breaking the loop once the maximum number of records (`bodyMaxRecords`) is reached. This ensures the request complies with the application's API limits.

### Batch of records of mixed types (create and update)

When the application allows sending multiple records of different types (both records to create and records to update) in a single HTTP request, you can iterate over all records using `records.All()`.

Here is an example implementation:

```go
func (m *MyApp) Upsert(ctx context.Context, target meergo.Targets, records meergo.Records) error {

	// Prepare request body.
	var body bytes.Buffer
	body.WriteString(`{"customers":[`)
	for i, record := range records.All() {
		if i > 0 {
			body.WriteString(`,`)
		}
		body.WriteString(`{`)
		if record.ID != "" {
			body.WriteString(`"id":`)
			json.Encode(&body, record.ID)
			body.WriteString(`,`)
		}
		body.WriteString(`"properties":`)
		json.Encode(&body, record.Properties)
		body.WriteString(`}`)
		if i+1 == bodyMaxRecords {
			break
		}
	}
	body.WriteString(`}]`)

	// Create the HTTP request.
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.myapp.com/v1/customers/batch", &body)

	// Send the HTTP request.
	res, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check the response status code.
	if res.StatusCode != 200 {
		return fmt.Errorf("server responded with status %s", res.Status)
	}

	return nil
}
```

#### Key concepts:

* **Iterating Over All Records**\
   The method `records.All()` is used to iterate over both types of records—those to be created and those to be updated. This makes it possible to process mixed batches in a single request.

* **Determine the Type of Operation**\
   If the record has an `ID`, it's an update, if `ID` is empty, it's a creation.

* **Limit on Records**\
   The loop stops once the maximum number of records (`bodyMaxRecords`) is reached, ensuring that the body size does not exceed the application’s limit.

This approach allows you to efficiently handle mixed record types (create and update) in a single batch request, reducing the number of API calls required.

### Handling body size limits

In the previous examples, the loop stops when the number of records reaches the API's maximum limit. However, if the API imposes a body size limit rather than a record count limit, you can use the `Skip` method to skip a record after it has been read. This ensures that the record remains unprocessed and can be included in a subsequent call to the `Upsert` method.

Below is an example implementation:

```go
	for i, record := range records.All() {

		// Track length before adding the record.
		n = body.Len()

		if i > 0 {
			body.WriteString(`,`)
		}

		// Build the record JSON object.
		body.WriteString(`{`)
		if record.ID != "" {
			body.WriteString(`"id":`)
			json.Encode(&body, record.ID)
			body.WriteString(`,`)
		}
		body.WriteString(`"properties":`)
		json.Encode(&body, record.Properties)
		body.WriteString(`}`)

		// Stop if body exceeds app size limit.
		if body.Len() > bodySizeLimit {
			body.Truncate(n)
			records.Skip()
			break
		}

	}
```

#### Key concepts:

* **Tracking Body Size**\
  Before adding a record to the request body, the current length of the body is tracked using `body.Len()`. This allows for easy truncation if the body size limit is exceeded.

* **Truncating the Body**\
  To ensure the request is valid, the `body.Truncate(n)` method removes the last added record from the body. This prevents the body from exceeding the size limit while maintaining a valid JSON structure.

* **Using `Skip` to Reprocess Records**\
  When the body size exceeds the limit:
    - The Skip method is called to notify Meergo that the last processed record has been skipped.
    - The processed records remain unchanged, meaning they can potentially be skipped later.
    - This skipped record will remain unprocessed and will be included in the next call to the `Upsert` method.

### Differentiating errors by record

When records are sent in a batch, some APIs respond with specific error messages for each individual record in the batch. In this case, instead of returning a common error for all records, you can return a specific error for each record that encountered an issue using the `RecordsError` type:

```go
// RecordsError is returned by the Upsert method of an app connector when only
// some records have failed or when the method can distinguish errors based on
// individual records. It maps record indices to their respective errors.
type RecordsError map[int]error

func (err RecordsError) Error() string {
    var msg string
    for i, e := range err {
        msg += fmt.Sprintf("record %d: %v\n", i, e)
    }
    return msg
}
```

### Key concepts:

* **Error Handling for Individual Records**\
   Instead of returning a single error for the entire batch, you can return an error specific to each record that failed. This helps identify exactly which record(s) caused the issue.

* **Mapping Errors by Record Index**\
   The key of the `RecordsError` type is the index of the record in the iteration, which corresponds to the position of the record in the request body (assuming the records were written in the same order). The value is the error associated with that specific record.

This approach lets you identify and handle errors for each record separately, instead of having a single error for the whole batch.
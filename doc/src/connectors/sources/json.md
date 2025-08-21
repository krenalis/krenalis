{% extends "/layouts/doc.html" %}
{% macro Title string %}JSON data source{% end %}
{% Article %}

# JSON data source

The JSON data source allows you to import user data from a JSON file, which you can then unify into users within Meergo.

> Before adding a JSON data source, ensure that you have configured a storage data source such as S3, SFTP, or HTTP Files. If you haven’t set up a storage source yet, please do so before proceeding with the JSON file import.

### On this page

- [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)
- [Supported JSON formats](#supported-json-formats)

### Import users into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the storage data source from which you want to import the JSON file.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. From the **Format** menu, select **JSON**.
5. In the **Path** field, enter the path of the JSON file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct.
6. (Optional) In the **Compression** field, if the JSON file is compressed, select the compression format; Meergo automatically decompresses the file upon reading.
7. In the **Properties** section, enter the properties names to read from the file and indicate if they are required or optional.
8. Click **Preview** to view a preview of the file.
9. Click **Confirm** to confirm your selections. You can change them at any time later if needed.

### Supported JSON formats

JSON files containing an Array of Objects are supported, where each Object maps properties to values, as in this example:

```json
[
    {
        "name": "John",
        "email": "john@example.com",
        "score": 328.2
    },
    {
        "name": "Paul",
        "email": "paul@example.com",
        "score": 240.2
    }
]
```

Or JSON files containing an Object with a single key — regardless of what that key is — whose associated value is an Array of Objects, with the same structure as the previous example, are supported:

```json
{
    "some_key_name": [
        {
            "name": "John",
            "email": "john@example.com",
            "score": 328.2
        },
        {
            "name": "Paul",
            "email": "paul@example.com",
            "score": 240.2
        }
    ]
}
```

Any other JSON format is not supported at this time.

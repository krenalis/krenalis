{% extends "/layouts/doc.html" %}
{% macro Title string %}JSON (Destination){% end %}
{% Article %}

# JSON (Destination)

The destination connector for JSON allows you to export unified users (i.e., users consolidated through identity resolution) into a JSON file and save it to a storage location, such as S3 or SFTP.

> Before adding a destination connection for JSON, ensure that you have configured a destination connection for a storage such as S3, SFTP, or HTTP POST. If you haven't set up a storage destination yet, please do so before proceeding with the JSON file export.

### On this page

- [Add destination connection for JSON](#add-destination-connection-for-json)
- [Exported JSON format](#exported-json-format)

### Add destination connection for JSON

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the storage data destination where you want to export the JSON file.
3. If there are no actions, click **Add**; otherwise, click **Add new action**.
4. From the **Format** menu, select **JSON**.
5. In the **Path** field, enter the path of the JSON file, relative to the storage root path. The absolute path will be displayed so you can verify its accuracy.
6. (Optional) Proceed with the other fields:
    * **Compression**: Compression format. Select a format if you want the JSON file to be compressed.
    * **Order users by**: Sorting of users. Select a property if you want the users to be written in ascending order based on this property.
    * **Indent the generated output**: Indicates if the file should contain only ASCII characters. If selected, non-ASCII characters in JSON strings are escaped; for example `"José"` is written as `"Jos\u00e9"`.
    * **Allow non-standard NaN, Infinity, and -Infinity values**: Indicates how to write NaN and ±Infinity values in JSON. Select this option if you want them to be written as non-standard values `NaN`, `Infinity`, and `-Infinity`.
7. Click **Add** to add the action.

### Exported JSON format

The exported JSON file is a JSON Object containing a single key, `records`, whose value is an Array containing the various Objects representing the exported users. For example:

```json
{
    "records": [
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

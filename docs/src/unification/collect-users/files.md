{% extends "/layouts/doc.html" %}
{% import "/imports/image.html" %}
{% macro Title string %}Collect users from files{% end %}
{% Article %}

# Collect users from files
## Learn how to collect users from files.

Meergo makes it easy to collect user data directly from files, seamlessly map its schema to the Customer Model schema, and load it into your data warehouse for a unified and consistent customer view.

Meergo currently supports **CSV**, **Excel**, **JSON**, and **Parquet** file formats. Files can be read from **S3**, **SFTP**, and **HTTP** sources. In a testing environment, it is also possible to read files directly from the local file system.

## Steps

### 1. Connect a storage

To get started, connect to the file storage where your file is located (unless you've already connected it previously, for example when importing another file).

1. Go to the **Sources** page of your Meergo workspace.
2. Click on **Add a new source ⊕** and click on the card corresponding to your file storage type (**S3**, **SFTP**, or **HTTP GET**).
3. Click on **Add source...**.

**Enter the connection details for the storage.** You don't need to specify the file name at this step — you'll do that after adding the storage.

<!-- tabs settings -->

### S3

| Field             | Description                                                     |
|-------------------|-----------------------------------------------------------------|
| Access Key ID     | Your AWS access key ID.                                         |
| Secret Access Key | Your AWS secret access key.                                     |
| Region            | AWS region where the S3 bucket is located.                      |
| Bucket name       | Name of the S3 bucket that contains the files you wish to read. |

### SFTP

| Field    | Description                                                 |
|----------|-------------------------------------------------------------|
| Host     | Hostname or IP address of the SFTP server.                  |
| Port     | Port number used for the SFTP connection (default is `22`). |
| Username | Username for authentication.                                |
| Password | Password associated with the username.                      |

### HTTP GET

| Field   | Description                                               |
|---------|-----------------------------------------------------------|
| Host    | Hostname or IP address of the HTTP server.                |
| Port    | Port used to connect (default is `443` for `https`).      |
| Headers | Key/value pairs of HTTP headers to send with the request. |

<!-- end tabs -->

Click **Add** to confirm the configuration. The connection you just created is a source connection. You can access it later by clicking **Sources** section in the sidebar.

### 2. Add an action to import users

On the connection page, click on **Add action...**.

{{ Screenshot("Add action", "/docs/unification/collect-users/add-action.png", "", 264) }}

### 3. Choose a format

Choose the format of the file you want to import. You can change it later if needed.

{{ Screenshot("Select file format", "/docs/unification/collect-users/select-file-format.png", "", 2275) }}

### 4. Enter file settings

Fill in the following fields:

<!-- tabs settings -->

#### CSV

{{ Screenshot("CSV format settings", "/docs/unification/collect-users/csv-format-settings.png", "", 2172) }}

| Field                                   | Description                                                                                                                                                                                                              |
|-----------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Path                                    | Path of the Excel file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct. |
| Compression                             | Compression format. If the CSV file is compressed, select the compression format; Meergo automatically decompresses the file upon reading.                                                                               |
| Separator                               | Character used to separate fields. By default, this is a comma. Specify another character if different.                                                                                                                  |
| Number of columns                       | Expected number of columns. If **Number of columns** is set to **0**, the number of expected columns is taken from the first record.                                                                                     |
| Trim leading space in fields            | Indicates whether leading whitespace in a field should be ignored.                                                                                                                                                       |
| The first row contains the column names | Indicates if the first row of the CSV file contains the column names. If not selected, the column names will default to A, B, C, etc., similar to Excel files.                                                           |

#### Excel

| Field                                   | Description                                                                                                                                                                                                                  |
|-----------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Path                                    | Path of the Excel file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct.     |
| Sheet                                   | Sheet name from which you want to read the users.                                                                                                                                                                            |
| Compression                             | Compression format. Note that the XLSX format is already compressed by design, so select a compression format only if the file has been additionally compressed. Meergo automatically decompresses the file when reading it. |
| The first row contains the column names | Indicates if the first row of the Excel file contains the column names. If not selected, the column names will default to A, B, C, etc.                                                                                      |

#### JSON

| Field       | Description                                                                                                                                                                                                             |
|-------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Path        | Path of the JSON file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct. |
| Compression | Compression format. If the JSON file is compressed, select the compression format; Meergo automatically decompresses the file upon reading.                                                                             |
| Properties  | Names of the properties to read from the file, and whether each is required or optional. Click **⊕** to add more properties.                                                                                            |

#### Parquet

| Field       | Description                                                                                                                                                                                                                |
|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Path        | Path of the Parquet file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct. |
| Compression | Compression format. If the Parquet file is compressed, select the compression format; Meergo automatically decompresses the file upon reading.                                                                             |

> For technical details on how a Parquet file is imported, see [How Parquet columns are imported](/sources/parquet#how-parquet-columns-are-imported). 

<!-- end tabs -->

Click **Preview** to show a preview of the file with the first rows.

Click **Confirm** to apply the settings. You can still modify them later if needed.

### 5. Filter rows

If you don't want to import all rows from the file, use a filter to select which users to import. Only users that match the filter conditions will be imported. If no filter is set, all users in the file will be imported. For more information on how to use filters, see the [Filter documentation](/filters).

{{ Screenshot("Filter", "/docs/unification/collect-users/filter.png", "", 2172) }}

### 6. Identity columns

Select the columns that uniquely identify each user in the file, and if available, the column that contains the user’s last change time. For this column, you can use the ISO 8601 format, a custom date format, or, for Excel files only, the native Excel date format.

{{ Screenshot("Identity columns", "/docs/unification/collect-users/file-identity-columns.png", "", 2172) }}

Select **Run incremental import** if you want subsequent imports to include only the rows updated after the last import.

### 7. Transformation

The **Transformation** section allows you to harmonize the file schema with your Customer Model schema. You can choose between visual mapping or advanced transformations using JavaScript or Python.

Its purpose is to assign values from the file to the properties of the Customer Model. You have full control over which properties to map, assigning only those that matter to your business context while leaving others unassigned when no corresponding values exist.

{{ Screenshot("Visual mapping", "/docs/unification/collect-users/file-visual-mapping.png", "", 2168) }}

For complete details on how transformations work for harmonization, see how to [harmonize data](/unification/harmonization).

### 8. Save your changes

When you're done, click **Add** (or **Save** if you're editing an existing action).

For a single storage connection, you can also create multiple actions to import different files from that storage, each with its own set of users.

## Continue reading

### Process collected users

Clean and standardize user data using visual mapping, JavaScript, or Python transformations.

{{ render "../_includes/manage-users-cards.html" }}

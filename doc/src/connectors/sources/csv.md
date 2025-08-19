{% extends "/layouts/doc.html" %}
{% macro Title string %}CSV data source{% end %}
{% Article %}

# CSV data source

The CSV data source allows you to import user data from a CSV (Comma-Separated Values) file, which you can then unify as users within Meergo.

> Before adding a CSV data source, ensure that you have configured a storage data source such as S3, SFTP, or HTTP Files. If you haven’t set up a storage source yet, please do so before proceeding with the CSV file import.

### On this page

* [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Import users into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the storage data source from which you want to import the CSV file.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. From the **Format** menu, select **CSV**.
5. In the **Path** field, enter the path of the CSV file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct.
6. (Optional) Proceed with the other fields:
   * **Compression**: Format of compression. If the CSV file is compressed, select the compression format; Meergo automatically decompresses the file upon reading. 
   * **Separator**: Character used to separate fields. By default, this is a comma. Specify another character if different.
   * **Number of columns**: Expected number of columns. If **Number of columns** is set to **0**, the number of expected columns is taken from the first record.
   * **Trim leading space**: Indicates whether leading whitespace in a field should be ignored.
   * **The first row contains the column names**: Indicates if the first row of the CSV file contains the column names. If not selected, the column names will default to A, B, C, etc., similar to Excel files. 
7. Click **Preview** to view a preview of the file.
8. Click **Confirm** to confirm your selections and continue. You can change them at any time later if needed.

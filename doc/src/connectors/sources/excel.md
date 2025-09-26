{% extends "/layouts/doc.html" %}
{% macro Title string %}Excel (Source){% end %}
{% Article %}

# Excel (Source)

The source connector for Excel allows you to import user data from an Excel file, which you can then unify into users within Meergo.

> Before adding a source connection for Excel, ensure that you have configured a source connection for a storage such as S3, SFTP, or HTTP GET. If you haven't set up a storage source yet, please do so before proceeding with the Excel file import.

### On this page

- [Supported file formats](#supported-file-formats)
- [Import users into the workspace's data warehouse](#import-users-into-the-workspaces-data-warehouse)

### Supported file formats

The Excel data source only supports importing XLSX files (Microsoft Excel Spreadsheets). Other formats (e.g., ODS) are not supported at this time.

### Import users into the workspace's data warehouse

1. From the Meergo Admin console, go to **Connections > Sources**.
2. Click on the storage data source from which you want to import the Excel file.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. From the **Format** menu, select **Excel**.
5. In the **Path** field, enter the path of the Excel file, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct.
6. In the **Sheet** field, select the file sheet from which you want to read the users.
7. (Optional) Proceed with the other fields:
   * **Compression**: Format of compression. If the Excel file has been compressed further, select the compression format; Meergo automatically decompresses the file upon reading.
   * **The first row contains the column names**: Indicates if the first row of the Excel file contains the column names. If not selected, the column names will default to A, B, C, etc.
8. Click **Preview** to view a preview of the file.
9. Click **Confirm** to confirm your selections. You can change them at any time later if needed.

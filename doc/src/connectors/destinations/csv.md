{% extends "/layouts/doc.html" %}
{% macro Title string %}CSV data destination{% end %}
{% Article %}

# CSV data destination

The CSV data destination allows you to export unified users (i.e., users consolidated through identity resolution) into a CSV (Comma-Separated Values) file and save it into a storage, such as S3 and SFTP.

> Before adding a CSV data destination, ensure that you have configured a storage data destination such as S3, SFTP, or HTTP Files. If you haven’t set up a storage destination yet, please do so before proceeding with the CSV file export.

### On this page

* [Add a CSV data destination](#add-a-csv-data-destination)

### Add a CSV data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. Click on the storage data source from which you want to export the CSV file.
3. If there are no actions, click  **Add**, otherwise click  **Add new action**.
4. From the **Format** menu, select **CSV**.
5. In the **Path** field, enter the path of the CSV file to create, relative to the storage root path. Note that when you enter the relative path, the absolute path of the file will be displayed, so you can check that the path that you have entered is correct.
6. Optionally proceed with the other fields:
    * **Compression**: Select a compression format if you want the file to be compressed.
    * **Order users by**: Choose the property you want to use to sort the users in the file.
    * **Comma**: Character used to separate fields. By default, this is a comma. Specify another character if you want a different delimiter.
    * **Use CRLF**: Terminate lines with `\r\n` instead of `\n`.
7. Click **Add** to add the action.

{% extends "/layouts/doc.html" %}
{% macro Title string %}Excel data destination{% end %}
{% Article %}

# Excel data destination

The Excel data destination allows you to export unified users (i.e., users consolidated through identity resolution) into an Excel file and save it to a storage location, such as S3 or SFTP.

> Before adding an Excel data destination, ensure that you have configured a storage data destination such as S3, SFTP, or HTTP POST. If you haven’t set up a storage destination yet, please do so before proceeding with the Excel file export.

### On this page

- [Supported file formats](#supported-file-formats)
- [Add an Excel data destination](#add-an-excel-data-destination)

### Supported file formats

The Excel data destination only supports exporting XLSX files (Microsoft Excel Spreadsheets). Other formats (e.g., ODS) are not supported at this time.

### Add an Excel data destination

1. From the Meergo Admin console, go to **Connections > Destinations**.
2. Click on the storage data destination where you want to export the Excel file.
3. If there are no actions, click **Add**; otherwise, click **Add new action**.
4. From the **Format** menu, select **Excel**.
5. In the **Path** field, enter the path of the Excel file, relative to the storage root path. The absolute path will be displayed so you can verify its accuracy.
6. In the **Sheet** field, enter the name you want for the sheet in the Excel file.
7. (Optional) Proceed with the other fields:
   * **Compression**: Select a compression format if you want the Excel file to be compressed. Note, however, that Excel files are already compressed by their nature.
   * **Order users by**: Choose the property you want to use to sort the users in the file.
8. Click **Add** to add the action.

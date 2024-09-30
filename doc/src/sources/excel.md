# Excel data source

The Excel data source allows you to read user data from an Excel file, which you can then unify into users within Meergo.

> Before adding an Excel data source, ensure that you have configured a storage data source such as HTTP, S3, or SFTP. If you haven’t set up a storage source yet, please do so before proceeding with the Excel file import.

### On this page

* [Add an Excel data source](#add-an-excel-data-source)

### Add an Excel data source

1. From the Meergo admin, go to **Connections > Sources**.
2. Click on the storage data source from which you want to read the Excel file.
3. If there is no actions, clic on **Add**, otherwise clic on **Add new action**.
4. From the **Type** menu, select **Excel**.
5. In the **Path** field, enter the path of the Excel file, relative to the storage root path. Note that when you enter the relative path, the complete path of the file will be displayed, so you can check that the path that you have entered is correct.
6. In the **Sheet** field, select the file sheet from which you want to read the users.
7. Optionally proceed with the other fields:
   * **Compression**: Format of compression. If the Excel file has been compressed further, select the compression format; Meergo automatically decompresses the file upon reading.
   * **The first row contains the column names**: Indicates if the first row of the Excel file contains the column names. If not selected, the column names will default to A, B, C, etc.
8. Click **Preview** to view a preview of the file.
9. Click **Confirm** if the file has been read as expected.

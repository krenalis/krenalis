{% extends "/layouts/doc.html" %}
{% macro Title string %}Parquet data destination{% end %}
{% Article %}

# Parquet data destination

The Parquet data destination allows you to export unified users (i.e., users consolidated through identity resolution) into a Parquet file and save it to a storage location, such as S3 or SFTP.

> Before adding a Parquet data destination, ensure that you have configured a storage data destination such as S3, SFTP, or HTTP Files. If you haven’t set up a storage destination yet, please do so before proceeding with the Parquet file export.

### On this page

* [Add a Parquet data destination](#add-a-parquet-data-destination)

### Add a Parquet data destination

1. From the Meergo admin, go to **Connections > Destinations**.
2. Click on the storage data destination where you want to export the Parquet file.
3. If there are no actions, click **Add**; otherwise, click **Add new action**.
4. From the **Format** menu, select **Parquet**.
5. In the **Path** field, enter the path of the Parquet file, relative to the storage root path. The absolute path will be displayed so you can verify its accuracy.
6. Optionally, proceed with the other fields:
    * **Compression**: Compression format. Select a format if you want the Parquet file to be compressed.
    * **Order users by**: Sorting of users. Select a property if you want the users to be written in ascending order based on this property.
7. Click **Add** to add the action.

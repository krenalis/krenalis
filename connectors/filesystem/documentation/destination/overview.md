
File System is a connector for testing the export of files to the local file system.

## What can you do with this?

Using this connector you can export files to the file system of the installation that Krenalis is running on.

Its sole purpose is to test file exports and explore the various file formats supported by Krenalis, and it is strongly discouraged to use this connector in production, as it can freely access the file system.

## What does it require?

Requirements for the File System connector:

* A local file system directory that will be accessed by the File System connector.
* The running Krenalis instance must have the `KRENALIS_CONNECTOR_FILESYSTEM_ROOT` environment variable, which points to the file system directory that will be accessed by the connector.
* Optionally, the `KRENALIS_CONNECTOR_FILESYSTEM_DISPLAYED_ROOT` environment variable controls the root displayed in the admin.

The File System connector reads `KRENALIS_CONNECTOR_FILESYSTEM_*` directly from the process environment. These settings are not loaded from AWS Parameter Store.

💡 **Note:** When running Krenalis with Docker Compose, the File System connector is automatically configured by default and you can skip this section.

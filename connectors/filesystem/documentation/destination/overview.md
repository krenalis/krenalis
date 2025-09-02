
Filesystem is a connector for testing the export of files to the local filesystem.

## What can you do with this?

Using this connector you can export files to the filesystem of the installation that Meergo is running on.

Its sole purpose is to test file exports and explore the various file formats supported by Meergo, and it is strongly discouraged to use this connector in production, as it can freely access the filesystem.

## What does it require?

> 💡 Note: When running Meergo with Docker Compose, the Filesystem connector is automatically configured by default and you can skip this section.

Requirements for the Filesystem connector:

* A local filesystem directory that will be accessed by the Filesystem connector.
* The running Meergo instance must have the `MEERGO_CONNECTOR_FILESYSTEM_ROOT` environment variable, which points to the filesystem directory that will be accessed by the connector.
* Optionally, the `MEERGO_CONNECTOR_FILESYSTEM_DISPLAYED_ROOT` environment variable controls the root displayed in the admin.

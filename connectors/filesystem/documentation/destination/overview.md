
Filesystem is a connector for testing the export of files to the local filesystem.

## What can you do with this?

Using this connector you can export files to the filesystem of the installation that Meergo is running on.

Its sole purpose is to test file exports and explore the various file formats supported by Meergo, and it is strongly discouraged to use this connector in production, as it can freely access the filesystem.

## What does it require?

A directory in the local filesystem to use as the Root Path, which Meergo can access.

**When running Meergo under Docker**, you can add a Filesystem connection whose Root Path is:

```plain
/bin/meergo-files/sample-filesystem
```

which is mapped to the directory:

```plain
./docker-compose/sample-filesystem
```

within your local Meergo repository.

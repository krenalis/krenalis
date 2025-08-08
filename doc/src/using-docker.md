{% extends "/layouts/doc.html" %}
{% macro Title string %}Using Docker{% end %}
{% Article %}

# Using Docker

This is the easiest and fastest way to start using Meergo on your PC. It's ideal for local development, testing, and exploring its features.

> 🚧 Using Meergo with Docker will be easier when the Meergo image will be published to Docker Hub.

## Before you begin

To use Meergo using Docker, you will need:

* [Git](https://git-scm.com/downloads). To download the Meergo repository.
* [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/). To build the Meergo image and run it within a pre-configured environment.

## Steps

Clone the Meergo's repository and enter into the repository's directory: 

```bash
git clone https://github.com/meergo/meergo
cd meergo
```

Build the Docker image of Meergo:

```sh
docker build -t meergo:dev . --progress=plain
```

Launch the built image with Docker Compose:

```
docker compose up
```

> 🧹 If you have previously started Meergo using Docker Compose and want to reset it, perhaps for a clean installation or because you are running a new version of Meergo, you just need to clear the Meergo Docker data by running `docker compose down -v` before starting Meergo with `docker compose up`.

Now you can start using Meergo by visiting the admin at [http://localhost:9090/admin/](http://localhost:9090/admin/).

Keep reading the documentation to see how [create your first workspace](./create-workspace).

## Import and export local files with Docker

When running Meergo under Docker, for importing and exporting files locally, you can add a Filesystem connection whose Root Path is:

```plain
/bin/meergo-files/sample-filesystem
```

which is mapped to the directory:

```plain
<local Meergo repository>/docker-compose/sample-filesystem
```

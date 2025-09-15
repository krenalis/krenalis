{% extends "/layouts/doc.html" %}
{% macro Title string %}Using Docker{% end %}
{% Article %}

# Using Docker

This is the easiest and fastest way to start using Meergo on your PC.
## Before you begin

To use Meergo using Docker, you will need:

* [Git](https://git-scm.com/downloads). To download the Meergo repository.
* [Docker](https://docs.docker.com/engine/install/) and [Docker Compose](https://docs.docker.com/compose/). To build the Meergo image and run it within a pre-configured environment.

## One-line step

Try Meergo locally by simply running:

```bash
git clone https://github.com/meergo/meergo.git && cd meergo && docker compose up
```

Or, alternatively, follow the detailed steps below.

## Detailed steps

Clone the Meergo's repository and enter into the repository's directory: 

```bash
git clone https://github.com/meergo/meergo
cd meergo
```

Launch Meergo with Docker Compose:

```
docker compose up
```

> 🧹 If you have previously started Meergo using Docker Compose and want to reset it, perhaps for a clean installation or because you are running a new version of Meergo, you just need to clear the Meergo Docker data by running `docker compose down -v` before starting Meergo with `docker compose up`.

Now you can start using Meergo by visiting the Admin console at [http://localhost:2022/admin/](http://localhost:2022/admin/).

Keep reading the documentation to see how [create your first workspace](./create-workspace).

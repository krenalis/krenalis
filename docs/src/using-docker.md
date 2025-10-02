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

### 1. Clone the repository

Clone the Meergo's repository and enter into the repository's directory: 

```bash
git clone https://github.com/meergo/meergo
cd meergo
```

### 2. Start Meergo

Launch Meergo with Docker Compose:

```
docker compose up
```

If you have previously started Meergo using Docker Compose and want to reset it, perhaps for a clean installation or because you are running a new version of Meergo, you must clear the Meergo Docker data with `docker compose down -v`. This command removes containers **and volumes**, permanently deleting all stored data.

### 3. Open the Admin console

⏳ Wait until the initialization of Meergo's databases is complete. This may take a few seconds.

Once the initialization is finished, as indicated by the console message stating that the Meergo Admin Console is available, you can access it by opening the address [http://127.0.0.1:2022/admin/](http://127.0.0.1:2022/admin/) in a browser.

Keep reading the documentation to see how to [create your first workspace](./create-workspace).

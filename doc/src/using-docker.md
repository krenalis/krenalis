{% extends "/layouts/doc.html" %}
{% macro Title string %}Using Docker{% end %}
{% Article %}

# Using Docker

This is the easiest and fastest way to start using Meergo on your PC. It's ideal for local development, testing, and exploring its features.

> 🚧 Using Meergo with Docker will be easier when the Meergo image will be published to Docker Hub.

For this installation method you need to have [Git](https://git-scm.com/downloads) and [Docker](https://docs.docker.com/engine/install/).

1. Clone the Meergo's repository and enter into the repository's directory: 

   ```sh
   git clone https://github.com/meergo/meergo
   cd meergo
   ```

2. Build the Docker image of Meergo and launch it:

   ```sh
   docker build -t meergo:dev . --progress=plain
   docker compose up
   ```

> 🧹 If you have previously started Meergo using Docker Compose and want to reset it, perhaps for a clean installation or because you are running a new version of Meergo, you just need to clear the Meergo Docker data by running `docker compose down -v` before starting Meergo with `docker compose up`.

3. Open the Meergo admin at [http://localhost:9090/admin/](http://localhost:9090/admin/)

Initially, login is not required with the Docker installation. To enable login, create a new member with their email and password.

# Installation

At present, you can install Chichi by compiling its source code in Go.

## Requirements

* Git
* Go v1.22
* Node.js 20 or higher
* npm
* PostgreSQL 15 or higher
* Linux, Unix, OSX or Windows
* A data warehouse (currently only PostgreSQL)

## How to install

1. Clone the repository

    ```sh
   git clone https://github.com/open2b/chichi
   ```

2. Build the executable

    ```sh
    cd ui
    npm install 
    cd ../chichi
    go build -tags osusergo,netgo -trimpath
    ```

3. Create the configuration file

    ```sh
    cp config.example.yaml config.yaml
    ```

    The configuration file `config.yaml` contains settings for starting Chichi. This is written in the [YAML markup language](https://yaml.org/spec/1.2.2/).

4. Add the `cert.pem` file with the certificate for the domain and the `key.pem` file with its key to the main directory of the repository.
 
5. Set up the database by executing the `database/PostgreSQL.sql` script. This script is designed to configure the PostgreSQL database.

## Start

```sh
./chichi
```

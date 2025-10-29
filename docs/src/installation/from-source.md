{% extends "/layouts/doc.html" %}
{% macro Title string %}Install from source{% end %}
{% Article %}

# Install from source

You can also compile Meergo directly from source. This is recommended for advanced users who want full control over the build and customization process. There are two options:

<ul class="cards" data-columns="2">
  <li>
    <a href="#build-with-go-tools">
      <div>Build with Go tools</div>
      <p>Use Go's command-line tools to compile Meergo. Does not require Git. Recommended if you only want to choose which connectors to include in the executable.</p>
    </a>
  </li>
  <li>
    <a href="#build-from-the-repository">
      <div>Build from the repository</div>
      <p>Clone and build the Meergo repository. Offers maximum flexibility and customization.</p>
    </a>
  </li>
</ul>

## Build with Go tools

This method uses Go’s command-line tools and does not require Git. It is the recommended approach if you only want to customize which connectors are included in the executable.

### Before you begin

Make sure you have installed:

* [Go](https://go.dev/doc/install) 1.24 or later
* [PostgreSQL](https://www.postgresql.org/download/) 13 or later
* [curl](https://curl.se/) or any tool capable of downloading files over HTTP

### 1. Create the *meergo* directory

```sh
$ mkdir meergo
$ cd meergo
```

### 2. Create *main.go*

Download the default `main.go` file from the Meergo repository:

```sh
$ curl -o main.go 'https://raw.githubusercontent.com/meergo/meergo/refs/heads/main/cmd/meergo/main.go?token=GHSAT0AAAAAACG2OK4HH2M7QWHVRAVO322W2F6X3YA'
```

### 3. Initialize the module

```sh
$ go mod init meergo
$ go mod tidy
```

### 4. Generate assets

Generate the Admin console assets:

```sh
$ go generate
```

After this step, your directory should look like:

```
meergo
├── go.mod
├── go.sum
├── main.go
└── meergo-assets
    ├── index.css.br
    ├── index.css.map.br
    ├── index.html.br
    ├── index.js.br
    ├── index.js.map.br
    ├── monaco
    └── shoelace
```

### 5. Customize what is included

To define which Meergo connectors and data warehouse drivers are included in the executable, edit the _main.go_ file. You can remove built-in connectors or drivers, or add your own.

### 6. Build

Build the executable:

```sh
$ go build
```

Or to remove absolute paths from stack traces, use:

```sh
$ go build -trimpath
```

Verify the build (replace `meergo` with `meergo.exe` on Windows):

```sh
$ ./meergo --help
```

### Next step

Proceed with the [database setup](./database-setup).

## Build from the repository

### Before you begin

Install the following:

* [Git](https://git-scm.com/downloads)
* [Go](https://go.dev/doc/install) 1.24 or later
* [PostgreSQL](https://www.postgresql.org/download/) 13 or later

### 1. Clone the repository

```sh
$ git clone https://github.com/meergo/meergo
$ cd meergo/cmd/meergo
```

### 2. Generate assets

```sh
$ go generate
```

After this step, the `meergo/cmd/meergo` directory should look like:

```
meergo
├── main.go
├── meergo-assets
│   ├── index.css.br
│   ├── index.css.map.br
│   ├── index.html.br
│   ├── index.js.br
│   ├── index.js.map.br
│   ├── monaco
│   └── shoelace
└── meergo.example.env
```

### 3. Build

```sh
$ go build
```

Or to remove absolute paths from stack traces, use:

```sh
$ go build -trimpath
```

Verify the build (replace `meergo` with `meergo.exe` on Windows):

```sh
$ ./meergo --help
```

### Next step

Proceed with the [database setup](./database-setup).

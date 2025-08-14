{% extends "/layouts/doc.html" %}
{% macro Title string %}From source{% end %}
{% Article %}

# From source

You can also compile Meergo directly from source. This is recommended for more advanced users who want more control over builds and customization.

There are two alternatives:

* [**Building using the Go tools**](#building-using-the-go-tools). It uses Go's command-line tools to compile Meergo and doesn't require Git.
* [**Building from the repository**](#building-from-the-repository). Downloading and building the Meergo repository. This is a more advanced method, allowing for maximum control and customization of the build.

## Building using the Go tools

### Before you begin

To build Meergo using the Go tools and use it, you need:

* [Go](https://go.dev/doc/install) 1.23
* [PostgreSQL](https://www.postgresql.org/download/) version 13 or later.
* `curl`, or any other tool to download a file via HTTP from the web.

### Steps

Create a new directory called `meergo` and cd into it:

```bash
mkdir meergo
cd meergo 
```

Create a `main.go` file copying its content from the Meergo repository:

```bash
curl -o main.go 'https://raw.githubusercontent.com/meergo/meergo/refs/heads/main/cmd/meergo/main.go?token=GHSAT0AAAAAACG2OK4GJGLTBXJ5J337DZMM2EUZV6A'
```

Initialize a Go module:

```bash
go mod init meergo
go mod tidy
```

Generate the Admin console assets:

```bash
go generate
```

Now the directory should look like:

```plain
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

You are ready to build the `meergo` executable:

```bash
go build
```

> Note: You can provide the `-trimpath` option to the `go build` command to remove absolute paths from any error stack traces in Meergo. This way, if telemetry is enabled, the absolute paths will not be sent.

Verify it has been built correctly (replace `meergo` with `meergo.exe` on Windows):

```
./meergo --help
```

You can now proceed with the [database setup](./database-setup).

## Building from the repository

### Before you begin

To build Meergo from the repository the Go tools and use it, you need:

* [Git](https://git-scm.com/downloads)
* [Go](https://go.dev/doc/install) 1.23
* [PostgreSQL](https://www.postgresql.org/download/) version 13 or later.

### Steps

Clone the Meergo repository from GitHub and cd into the `meergo/cmd/meergo` directory:

```bash
git clone https://github.com/meergo/meergo
cd meergo/cmd/meergo
```

Generate the Admin console assets:

```bash
go generate
```

Now the `meergo/cmd/meergo` directory should look like:

```plain
meergo
├── main.go
├── meergo-assets
│   ├── index.css.br
│   ├── index.css.map.br
│   ├── index.html.br
│   ├── index.js.br
│   ├── index.js.map.br
│   ├── monaco
│   └── shoelace
└── meergo.example.env
```

You are ready to build the `meergo` executable:

```bash
go build
```

> Note: You can provide the `-trimpath` option to the `go build` command to remove absolute paths from any error stack traces in Meergo. This way, if telemetry is enabled, the absolute paths will not be sent.

Verify it has been built correctly (replace `meergo` with `meergo.exe` on Windows):

```
./meergo --help
```

You can now proceed with the [database setup](./database-setup).

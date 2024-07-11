# meergo-cli

`meergo-cli` is a tool that lets you interact with Meergo using a command line
interface.

- [Configuration](#configuration)
- [How to compile](#how-to-compile)
- [Alternative: install `meergo-cli` (tested on Linux only)](#alternative-install-meergo-cli-tested-on-linux-only)
- [Usage](#usage)

## Configuration

Put inside the directory `meergo-cli` of this repository a YAML file named
`meergo-cli.yaml` like this:

```yaml
apis:
  URL: https://localhost:9090

workspace: 1
```

## How to compile

From your terminal:

```
cd meergo-cli
go build
```

## Alternative: install `meergo-cli` (tested on Linux only)

You can install `meergo-cli` and invoke it from the command line.
To do so:

1. Add the configuration above to the directory `$HOME/.config/meergo-cli`, in a file called `meergo-cli.yaml`
2. Enter the directory `meergo-cli` of this repository
3. Run `go install`

## Usage

Run:

```
meergo-cli help
```

to get an overview of available subcommands. You can then run:

```
meergo-cli help <subcommand>
```

to get information about a specific subcommand, for example:

```
meergo-cli help connections
```

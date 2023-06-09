# chichi-cli

`chichi-cli` is a tool that lets you interact with Chichi using a command line
interface.

- [Configuration](#configuration)
- [How to compile](#how-to-compile)
- [Alternative: install `chichi-cli` (tested on Linux only)](#alternative-install-chichi-cli-tested-on-linux-only)
- [Usage](#usage)

## Configuration

Put inside the directory `chichi-cli` of this repository a YAML file named
`chichi-cli.yaml` like this:

```yaml
apis:
  URL: https://localhost:9090

workspace: 1
```

## How to compile

From your terminal:

```
cd chichi-cli
go build
```

## Alternative: install `chichi-cli` (tested on Linux only)

You can install `chichi-cli` and invoke it from the command line.
To do so:

1. Add the configuration above to the directory `$HOME/.config/chichi-cli`, in a file called `chichi-cli.yaml`
2. Enter the directory `chichi-cli` of this repository
3. Run `go install`

## Usage

Run:

```
chichi-cli help
```

to get an overview of available subcommands. You can then run:

```
chichi-cli help <subcommand>
```

to get information about a specific subcommand, for example:

```
chichi-cli help connections
```

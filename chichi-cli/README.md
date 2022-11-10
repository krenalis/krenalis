# chichi-cli

- [Configuration](#configuration)
- [How to compile](#how-to-compile)
- [Alternative: install `chichi-cli` (tested on Linux only)](#alternative-install-chichi-cli-tested-on-linux-only)
- [Usage](#usage)
  - [Not implemented subcommands](#not-implemented-subcommands)

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

**NOTE**: some of the commands above may be still not implemented. Please refer to
the section below.

```
chichi-cli connections
chichi-cli connections import <connection ID>
chichi-cli connections reimport <connection ID>
chichi-cli connections properties <connection ID>

chichi-cli transformations list <connection ID>
chichi-cli transformations get-source <transformation ID>
chichi-cli transformations update-source <transformation ID> { <filename> | - }
chichi-cli transformations get-connections <transformation ID>
chichi-cli transformations update-connections <transformation ID>

chichi-cli schemas show { user | group | event }
chichi-cli schemas update { user | group | event } { <filename> | - }
chichi-cli schemas properties { user | group | event }

chichi-cli users [list]
```

### Not implemented subcommands

Note that these subcommands are still not implemented:

- `chichi-cli connectors`
- `chichi-cli schemas` and related subcommands
- `chichi-cli transformations get-source`
- `chichi-cli transformations update-source`
- `chichi-cli transformations get-connections`
- `chichi-cli transformations update-connections`
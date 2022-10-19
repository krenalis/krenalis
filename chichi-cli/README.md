# chichi-cli

- [Configuration](#configuration)
- [How to compile](#how-to-compile)
- [Alternative: install `chichi-cli` (tested on Linux only)](#alternative-install-chichi-cli-tested-on-linux-only)
- [Usage](#usage)
  - [Implemented subcommands](#implemented-subcommands)

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
chichi-cli connectors

chichi-cli data-sources
chichi-cli data-sources import <connector ID>
chichi-cli data-sources reimport <connector ID>
chichi-cli data-sources properties <connector ID>

chichi-cli transformations show <connector ID>
chichi-cli transformations update <connector ID> { <filename> | - }

chichi-cli schemas show { user | group | event }
chichi-cli schemas update { user | group | event } { <filename> | - }
chichi-cli schemas properties { user | group | event }

chichi-cli users [list]
```

### Implemented subcommands

- [x] `chichi-cli data-sources`
- [x] `chichi-cli data-sources import <connector ID>`
- [x] `chichi-cli data-sources reimport <connector ID>`
- [x] `chichi-cli data-sources properties <connector ID>`
- [x] `chichi-cli transformations show <connector ID>`
- [x] `chichi-cli transformations update <connector ID> { <filename> | - }`
- [x] `chichi-cli users [list]`

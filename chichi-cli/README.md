# chichi-cli

- [Configuration](#configuration)
- [How to compile](#how-to-compile)
- [Usage](#usage)
  - [Implemented subcommands](#implemented-subcommands)

## Configuration

Put inside the directory `chichi-cli` of this repository a YAML file named
`chichi-cli.yaml` like this:

```yaml
apis:
  URL: https://localhost:9090

account: 1
```

## How to compile

From your terminal:

```
cd chichi-cli
go build
```

## Usage

**NOTE**: some of the commands above may be still not implemented. Please refer to
the section below.

```
chichi-cli connectors [list-enabled]
chichi-cli connectors list-available
chichi-cli connectors import <connector ID>
chichi-cli connectors reimport <connector ID>
chichi-cli connectors properties <connector ID>

chichi-cli transformations show <connector ID>
chichi-cli transformations update <connector ID> { <filename> | - }

chichi-cli schemas show { user | group | event }
chichi-cli schemas update { user | group | event } { <filename> | - }
chichi-cli schemas properties { user | group | event }

chichi-cli users [list]
```

### Implemented subcommands

- [x] `chichi-cli connectors [list-enabled]`
- [x] `chichi-cli connectors import <connector ID>`
- [x] `chichi-cli connectors reimport <connector ID>`
- [x] `chichi-cli connectors properties <connector ID>`
- [x] `chichi-cli transformations show <connector ID>`
- [x] `chichi-cli transformations update <connector ID> { <filename> | - }`
- [x] `chichi-cli users [list]`
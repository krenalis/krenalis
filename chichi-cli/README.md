# chichi-cli

- [Configuration](#configuration)
- [How to run](#how-to-run)
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

## How to run

From your terminal:

```
cd chichi-cli
go run main.go
```

## Usage

**NOTE**: some of the commands above may be still not implemented. Please refer to
the section below.

```
chichi connectors [list-enabled]
chichi connectors list-available
chichi connectors import <connector ID>
chichi connectors reimport <connector ID>
chichi connectors properties <connector ID>

chichi transformations show <connector ID>
chichi transformations update <connector ID> { <filename> | - }

chichi schemas show { user | group | event }
chichi schemas update { user | group | event } { <filename> | - }
chichi schemas properties { user | group | event }

chichi users [list]
```

### Implemented subcommands

- [x] `chichi connectors [list-enabled]`
- [x] `chichi connectors import <connector ID>`
- [x] `chichi connectors reimport <connector ID>`
- [x] `chichi connectors properties <connector ID>`
- [x] `chichi transformations show <connector ID>`
- [x] `chichi transformations update <connector ID> { <filename> | - }`
- [x] `chichi users [list]`
# NDID Migration Tools

## Prerequisites

- Go version >= 1.18.0
- (Optional) (for reading from LevelDB using C lib) LevelDB version >= 1.7 and snappy

## Getting Started

```sh
go run main.go
```

To run with C lib support for LevelDB:

```sh
CGO_ENABLED=1 CGO_LDFLAGS="-lsnappy" go run -tags "cleveldb" main.go
```

**Environment variable options**

- `INITIAL_STATE_DATA_DIR` : Directory path of initial state data
- `INITIAL_STATE_DATA_FILENAME` : File name of ABCI initial state data file [Default: `data`]
- `BACKUP_VALIDATORS_FILENAME` : File name of validators backup data
- `CHAIN_HISTORY_FILENAME` : File name of chain history data [Default: `chain_history`]

*Specific to `create-initial-state-data` command*

- `TM_HOME`: Source Tendermint home directory path
- `ABCI_DB_DIR_PATH` : Source ABCI state DB directory path

*Specific to `restore` command*

- `NDID_NODE_ID` : NDID node ID [Default: `NDID`]
- `KEY_DIR`: NDID node key directory path [Default: `./dev_keys/`]
- `TENDERMINT_RPC_HOST` : Tendermint RPC host [Default: `localhost`]
- `TENDERMINT_RPC_PORT` : Tendermint RPC port [Default: `45000`]

## Migrate Data to a New Chain

### Option 1

1. Run backup with command `create-initial-state-data [fromVersion] [toVersion]`

Example:

```sh
go run main.go create-initial-state-data 4 5
```

2. Run restore with command `restore [toVersion]`

Example:

```sh
go run main.go restore 5
```

### Option 2

1. Run create initial state data with command `create-initial-state-data [fromVersion] [toVersion]`

Example:

```sh
go run main.go create-initial-state-data 6 7
```

2. Use created initial state data with Tendermint/ABCI for `InitChain`. Refer to https://github.com/ndidplatform/smart-contract for usage.

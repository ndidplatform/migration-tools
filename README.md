# NDID Migration Tools

## Prerequisites

- Go version >= 1.13.0
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

- `BACKUP_DATA_DIR` : Directory path of backup data
- `BACKUP_DATA_FILENAME` : File name of ABCI stateDB backup data
- `BACKUP_VALIDATORS_FILENAME` : File name of validators backup data
- `CHAIN_HISTORY_FILENAME` : File name of chain history data

*Specific to `convert-and-backup` command*

- `TM_HOME`: Source Tendermint home directory path
- `ABCI_DB_DIR_PATH` : Source ABCI state DB directory path

*Specific to `restore` command*

- `NDID_NODE_ID` : NDID node ID [Default: `NDID`]
- `TENDERMINT_RPC_HOST` : Tendermint RPC host [Default: `localhost`]
- `TENDERMINT_RPC_PORT` : Tendermint RPC port [Default: `45000`]

## Migrate Data to a New Chain

1. Run backup with command `convert-and-backup [fromVersion] [toVersion]`

Example:

```sh
go run main.go convert-and-backup 4 5
```

2. Run restore with command `restore [toVersion]`

Example:

```sh
go run main.go restore 5
```

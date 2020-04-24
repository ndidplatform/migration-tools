# Backup from v3 for v4

Backup data from v3, convert data to v4, and save to file

## Prerequisites

- Go version >= 1.13.0

## Getting Started

```sh
go run main.go
```

**Environment variable options**

- `TM_HOME`: Source Tendermint home directory path
- `ABCI_DB_DIR_PATH` : Source ABCI state DB directory path
- `BACKUP_DATA_DIR` : Directory path for save backup data
- `BACKUP_DATA_FILE_NAME` : File name of ABCI stateDB backup data
- `BACKUP_VALIDATORS_FILE_NAME` : File name of validators backup data
- `CHAIN_HISTORY_FILE_NAME` : File name of chain history data

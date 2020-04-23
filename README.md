# NDID Migration Tools

## Migrate Data to a New Chain

1. Run backup script

   ```sh
   go run backup/main.go
   ```

   **Environment variable options**

   - `TM_HOME`: Source Tendermint home directory path
   - `ABCI_DB_DIR_PATH` : Source ABCI state DB directory path
   - `BACKUP_DATA_DIR` : Directory path for save backup data
   - `BACKUP_DATA_FILE_NAME` : File name of ABCI stateDB backup data
   - `BACKUP_VALIDATORS_FILE_NAME` : File name of validators backup data
   - `CHAIN_HISTORY_FILE_NAME` : File name of chain history data

2. Run restore script

   ```sh
   go run restore/main.go
   ```

   **Environment variable options**

   - `NDID_NODE_ID` : NDID node id
   - `BACKUP_DATA_DIR` : Directory path of backup data
   - `BACKUP_DATA_FILENAME` : File name of ABCI stateDB backup data
   - `CHAIN_HISTORY_FILENAME` : File name of chain history data
   - `TENDERMINT_ADDRESS` : Tendermint address

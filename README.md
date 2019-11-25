# NDID stateDB migarion tools

## Migrate Chain

1. Run backup script

    ```sh
    go run backup/main.go
    ```

    **Environment variable options**

    - `BLOCK_NUMBER` : Backup block number
    - `DB_NAME` : Source directory path for copy ABCI stateDB
    - `BACKUP_DATA_FILE_NAME` : File name of ABCI stateDB backup data
    - `BACKUP_VALIDATORS_FILE_NAME` : File name of validators backup data
    - `CHAIN_HISTORY_FILE_NAME` : File name of chain history data
    - `TENDERMINT_ADDRESS` : Tendermint address
    - `BACKUP_DATA_DIR` : Directory path for save backup data

2. Run restore script

    ```sh
    go run restore/main.go
    ```

    **Environment variable options**

    - `NDID_NODE_ID` : NDID node id
    - `BACKUP_DATA_FILE_NAME` : File name of ABCI stateDB backup data
    - `CHAIN_HISTORY_FILE_NAME` : File name of chain history data
    - `TENDERMINT_ADDRESS` : Tendermint address
    - `BACKUP_DATA_DIR` : Directory path of backup data
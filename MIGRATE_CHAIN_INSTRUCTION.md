# Migrate chain instructions

## Prerequisites

- Go version >= 1.13.0

  - [Install Go](https://golang.org/dl/) by following [installation instructions.](https://golang.org/doc/install)
  - Set GOPATH environment variable (https://github.com/golang/go/wiki/SettingGOPATH)

## Clone migation-tools

1. Clone the project

   ```sh
   git clone https://github.com/ndidplatform/migration-tools.git
   ```

## Disable chain

1. SetLastBlock through NDID API path POST `/set_last_block` or `/setLastBlock` for older versions

   body

   ```
   {
      block_height: number
   }
   ```

   Example:

   ```sh
   curl -vkX POST https://IP:PORT/ndid/set_last_block \
      -H "Content-Type: application/json" \
      -d "{\"block_height\":0}"
   ```

- `block_height` คือเลข Block สุดท้ายที่จะให้สามารถทำ Transaction ลง Blockchain ได้ **(block_height = 0 คือ setLastBlock เท่ากับ Block ปัจจุบัน และ -1 คือ ยกเลิกการ SetLastBlock)**

## Create initial ABCI state data

1. Save/Write down NDID node ID(s), master public key, and public key. These values can be queried using GET `/utility/nodes/<NDID_NODE_ID>`

   Example:

   ```sh
   curl -vkX GET https://localhost:8443/utility/nodes/ndid1 \
      -H "Content-Type: application/json"
   ```

2. Stop Tendermint and ABCI (`did-tendermint`)

3. Backup data directories and files of the current chain (both Tendermint and ABCI data)

4. Run migration command in `migration-tools` to create initial ABCI state data for another version from current version state DB / ABCI to files

   Example:

   ```sh
   cd migration-tools

   TM_HOME=/home/support/ndid/ndid/tendermint/ \
   ABCI_DB_DIR_PATH=/home/support/ndid/ndid/data/ndid/abci/ \
   go run main.go create-initial-state-data 8 9
   ```

   or run with C lib support for LevelDB (in case DB to backup uses cleveldb):

   ```sh
   TM_HOME=/home/support/ndid/ndid/tendermint/ \
   ABCI_DB_DIR_PATH=/home/support/ndid/ndid/data/ndid/abci/ \
   CGO_ENABLED=1 CGO_LDFLAGS="-lsnappy" go run -tags "cleveldb" main.go create-initial-state-data 8 9
   ```

   - `TM_HOME` คือ Home directory ของ Tendermint
   - `ABCI_DB_DIR_PATH` คือ Directory state DB ของ ABCI

5. (Optional) Remove all service containers (Tendermint-ABCI  `did-tendermint`, API, and redis)

   Example:

   ```sh
   docker-compose down
   ```

## Remove previous chain data

1. Reset blockchain data

   Example (using docker compose):

   ```sh
   docker-compose run --rm tm-abci unsafe_reset_all
   ```

2. Remove ABCI data / stateDB

   Example:

   ```sh
   rm -rf /path/to/abci/data/directory/abci/didDB.db
   ```

3. (Optional) Remove/clear API service's redis cache

## Restore data

1. Pull new docker image version

2. เอา `config.toml` และ `genesis.json` อันใหม่ไปวางใน directory `config` ที่ Tendermint home directory

3. แก้ `TM_P2P_PORT` ของ tendermint ใน `.env` file เพื่อไม่ให้ node อื่นต่อเข้ามาได้ระหว่าง restore

4. Start Tendermint/ABCI (`did-tendermint`) (docker container) with environment variable `ABCI_INITIAL_STATE_DIR_PATH` points to directory generated in step 4 of ["Create initial ABCI state data"](#create-initial-abci-state-data) to load initial state on `InitChain`. Then, wait for Tendermint to finish chain initialization and block 1 is created.

5. Copy `master private key` ของ NDID ไปวางไว้ที่ `./dev_keys/` (หรือ directory path อื่นตาม environment variable `KEY_DIR` ที่กำหนด) ตั้งชื่อไฟล์ว่า `ndid_master` และ Copy `private key` ของ NDID ไปวางไว้ที่ `./dev_keys/` (หรือ directory path อื่นตาม environment variable `KEY_DIR` ที่กำหนด) ตั้งชื่อไฟล์ว่า `ndid` (ถ้าใช้ external key service เช่น HSM ให้ใช้ key ใดๆก่อนก็ได้ แล้วสั่งเปลี่ยน public key หลัง restore สำเร็จ)

6. Run `InitNDID` and `EndInit`.

   ```sh
   NDID_NODE_ID=<NDID_NODE_ID> \
   TENDERMINT_RPC_HOST=localhost \
   TENDERMINT_RPC_PORT=26000 \
   INITIAL_STATE_DATA_DIR=<PATH_TO_INITIAL_STATE_DATA_DIRECTORY> \
   go run main.go init-ndid 7
   ```

   ```sh
   NDID_NODE_ID=<NDID_NODE_ID> \
   TENDERMINT_RPC_HOST=localhost \
   TENDERMINT_RPC_PORT=26000 \
   go run main.go end-init 7
   ```

7. หลังจาก restore เสร็จเรียบร้อยแล้ว stop docker container ของ ABCI (`did-tendermint`)

8. แก้ `TM_P2P_PORT` ของ tendermint ใน `.env` file คืนค่าเดิม เพื่อให้ node อื่น ๆ สามารถต่อเข้ามาได้

9. Start docker containers (Tendermint-ABCI `did-tendermint`, API, and redis)

   Example:

   ```sh
   docker-compose up
   ```

10. (Optional) Set NDID node master public key and public key

   Example:

   ```sh
   cd migration-tools

   NDID_NODE_ID=<NDID_NODE_ID> \
   TENDERMINT_RPC_HOST=localhost \
   TENDERMINT_RPC_PORT=26000 \
   NODE_NEW_MASTER_PUBLIC_KEY_FILEPATH=<PATH_TO_NODE_NEW_MASTER_PUBLIC_KEY_FILE> \
   NODE_NEW_PUBLIC_KEY_FILEPATH=<PATH_TO_NODE_NEW_PUBLIC_KEY_FILE> \
   go run main.go update-node 7
   ```

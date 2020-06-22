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

1. SetLastBlock ผ่าน NDID API path POST `/set_last_block` หรือ `/setLastBlock` สำหรับ version เก่า

```sh
body

{
    block_height: number
}
```

หรือ

```sh
curl -skX POST https://IP:PORT/ndid/set_last_block \
    -H "Content-Type: application/json" \
    -d "{\"block_height\":0}" \
    -w '%{http_code}' \
    -o /dev/null
```

- `block_height` คือเลข Block สุดท้ายที่จะให้สามารถทำ Transaction ลง Blockchain ได้ **(block_height = 0 คือ setLastBlock เท่ากับ Block ปัจจุบัน และ -1 คือ ยกเลิกการ SetLastBlock)**

## Backup data

1. Save/Write down NDID node ID(s), master public key, and public key. These values can be queried using GET `/utility/nodes/<NDID_NODE_ID>`

2. Stop container ของ ABCI

   ```sh
   docker-compose stop
   ```

3. Backup data directories and files of the current chain

4. Run script จาก `migration-tools` เพื่อ convert และ backup ข้อมูลจาก state DB ของ ABCI ลงไฟล์

   Example:

   ```sh
   cd migration-tools

   TM_HOME=/home/support/ndid/ndid/tendermint/ \
   ABCI_DB_DIR_PATH=/home/support/ndid/ndid/data/ndid/abci/ \
   go run main.go convert-and-backup 4 5
   ```

   or run with C lib support for LevelDB (ในกรณีที่ DB ที่ต้องการ backup ใช้ cleveldb):

   ```sh
   TM_HOME=/home/support/ndid/ndid/tendermint/ \
   ABCI_DB_DIR_PATH=/home/support/ndid/ndid/data/ndid/abci/ \
   CGO_ENABLED=1 CGO_LDFLAGS="-lsnappy" go run -tags "cleveldb" main.go convert-and-backup 4 5
   ```

   - `TM_HOME` คือ Home directory ของ Tendermint
   - `ABCI_DB_DIR_PATH` คือ Directory state DB ของ ABCI

5. หลังจาก Run script backup เรียบร้อยแล้วสั่ง

   ```sh
   docker-compose down
   ```

6. เพื่อความปลอดภัย Backup directory ที่ mount ออกมาจาก container ABCI

## Remove previous chain data

1. Reset blockchain data

   ```sh
   docker-compose run --rm tm-abci unsafe_reset_all
   ```

2. ลบ stateDB ของ ABCI

   Example:

   ```sh
   rm -rf /home/support/ndid/ndid/data/ndid/abci/didDB.db
   ```

3. (Optional) ลบ/ล้าง redis cache ของ API

## Restore data

1. Pull docker image version ใหม่

2. เอา `config.toml` และ `genesis.json` อันใหม่ไปวางใน directory `config` ที่ Tendermint hone directory

3. แก้ `TM_P2P_PORT` ของ tendermint ใน `.env` file เพื่อไม่ให้ node อื่นต่อเข้ามาได้ระหว่าง restore

4. เปิด docker container เฉพาะของ ABCI ขึ้นมาเพื่อจะทำการ restore

5. Copy `master private key` ของ NDID ไปวางไว้ที่ `$GOPATH/src/github.com/ndidplatform/migration-tools/key/` ตั้งชื่อไฟล์ว่า `ndid_master` และ Copy `private key` ของ NDID ไปวางไว้ที่ `$GOPATH/src/github.com/ndidplatform/migration-tools/key/` ตั้งชื่อไฟล์ว่า `ndid` (ถ้าใช้ external key service เช่น HSM ให้ใช้ key ใดๆก่อนก็ได้ แล้วสั่งเปลี่ยน public key หลัง restore สำเร็จ)

6. Run script จาก `migration-tools` เพื่อ restore ข้อมูล

   Example:

   ```sh
   cd migration-tools

   NDID_NODE_ID=<NDID_NODE_ID> \
   TENDERMINT_RPC_HOST=localhost \
   TENDERMINT_RPC_PORT=26000 \
   BACKUP_DATA_DIR=<PATH_TO_BACKUP_DIRECTORY> \
   go run main.go restore 5
   ```

   - `NDID_NODE_ID` คือ ชื่อ node_id ของ NDID ที่จะใช้ initialize/register
   - `TENDERMINT_RPC_HOST` คือ Tendermint RPC host
   - `TENDERMINT_RPC_PORT` คือ Tendermint RPC port
   - `BACKUP_DATA_DIR` คือ directory ที่มีข้อมูลที่ convert แล้วเป็น JSON อยู่ในรูปแบบ text file

7. หลังจาก restore เสร็จเรียบร้อยแล้ว stop docker container ของ ABCI

8. แก้ `TM_P2P_PORT` ของ tendermint ใน `.env` file คืนค่าเดิม เพื่อให้ node อื่น ๆ สามารถต่อเข้ามาได้

9. เปิด docker container (ABCI, API และ Redis)

```sh
docker-compose up
```

10. (Optional) Set NDID node master public key and public key

# Prerequisites

- Go version >= 1.13.0

  - [Install Go](https://golang.org/dl/) by following [installation instructions.](https://golang.org/doc/install)
  - Set GOPATH environment variable (https://github.com/golang/go/wiki/SettingGOPATH)
- Dependency management for go
  - By following [installation instructions.](https://golang.github.io/dep/docs/installation.html)

## Clone migation-tools

1. Clone the project

    ```sh
    git clone https://github.com/ndidplatform/migration-tools.git
    ```

2. Checkout to correctly version

    ```sh
    git checkout v3.0.0-to-v4.0.0
    ```

## Disable chain

1. SetLastBlock ผ่าน NDID API path POST `/setLastBlock`

```sh
body

{
    block_height: number
}
```

หรือ

```sh
curl -skX POST https://IP:PORT/ndid/setLastBlock \
    -H "Content-Type: application/json" \
    -d "{\"block_height\":0}" \
    -w '%{http_code}' \
    -o /dev/null
```

- `block_height` คือเลข Block สุดท้ายที่จะให้สามารถทำ Transaction ลง Blockchain ได้ **(block_height = 0 คือ setLastBlock เท่ากับ Block ปัจจุบัน และ -1 คือ ยกเลิกการ SetLastBlock)**

## Backup data

1. Stop container ของ ABCI

    ```sh
    docker-compose stop
    ```

2. Run script จาก `migration-tools` เพื่อ backup ข้อมูลจาก stateDB ของ ABCI

    ```sh
    cd migration-tools

    TM_HOME=/home/support/ndid/ndid/tendermint/ \
    ABCI_DB_DIR_PATH=/home/support/ndid/ndid/data/ndid/abci/ \
    go run backup/main.go
    ```

    - TM_HOME คือ Home directory ของ Tendermint
    - ABCI_DB_DIR_PATH คือ Directory state DB ของ ABCI

3. หลังจาก Run script backup เรียบร้อยแล้วสั่ง

    ```sh
    docker-compose down
    ```

4. เพื่อความปลอดภัย Backup directory ที่ mount ออกมาจาก container ABCI

## Remove previous chain data

1. Reset blockchain data

    ```sh
    docker-compose run --rm tm-abci unsafe_reset_all
    ```

2. ลบ stateDB ของ ABCI

    ```sh
    rm -rf /home/support/ndid/ndid/data/ndid/abci/didDB.db
    ```

## Restore data

1. Pull docker image version ใหม่
2. เอา `config.toml` และ `genesis` อันใหม่ไปวางใน directory `config`
3. แก้ `TM_P2P_PORT` ของ tendermint ใน `.env` file เพื่อไม่ให้ node อื่นต่อเข้ามาได้ระหว่าง restore
4. เปิด docker container เฉพาะของ ABCI ขึ้นมาเพื่อจะทำการ restore
5. Copy `master private key` ของ NDID ไปวางไว้ที่ `$GOPATH/src/github.com/ndidplatform/migration-tools/key/` ตั้งชื่อไฟล์ว่า `ndid_master` และ Copy `private key` ของ NDID ไปวางไว้ที่ `$GOPATH/src/github.com/ndidplatform/migration-tools/key/` ตั้งชื่อไฟล์ว่า `ndid`
6. Run script จาก `migration-tools` เพื่อ restore ข้อมูล

    ```sh
    cd migration-tools

    NDID_NODE_ID=ndid1 TENDERMINT_ADDRESS=http://localhost:26000 go run restore/main.go
    ```

    - NDID_NODE_ID คือ ชื่อ node_id ของ NDID ที่จะใช้ init
    - TENDERMINT_ADDRESS คือ RPC Tendermint address

7. หลักจาก restore เสร็จเรียบร้อยแล้ว stop docker container ของ ABCI
8. แก้ `TM_P2P_PORT` ของ tendermint ใน `.env` file เพื่อให้ node อื่น ๆ สามารถต่อเข้ามาได้
9. เปิด docker container (ABCI, API และ Redis)

```sh
docker-compose up
```

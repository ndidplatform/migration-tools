package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/ndidplatform/migration-tools/utils"
	did "github.com/ndidplatform/smart-contract/abci/did/v1"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	kvPairPrefixKey = []byte("kvPairKey:")
)

func main() {
	curChainData := getLastestTendermintData()
	waitAnswer("Y")
	readStateDBAndWriteToFile(curChainData)
}

func getLastestTendermintData() (chainData ChainHistoryDetail) {
	// Get lastest chain info
	resStatus := utils.GetTendermintStatus()
	latestBlockHeightStr := resStatus.Result.SyncInfo.LatestBlockHeight
	latestBlockHeight, err := strconv.ParseInt(latestBlockHeightStr, 10, 64)
	if err != nil {
		panic(err)
	}
	blockStatus := utils.GetBlockStatus(latestBlockHeight)
	chainID := blockStatus.Result.Block.Header.ChainID
	latestBlockHash := blockStatus.Result.BlockMeta.BlockID.Hash
	latestAppHash := blockStatus.Result.Block.Header.AppHash
	fmt.Println("Chain ID: " + chainID)
	fmt.Println("Latest Block Height: " + latestBlockHeightStr)
	fmt.Println("Latest Block Hash: " + latestBlockHash)
	fmt.Println("Latest App Hash: " + latestAppHash)
	chainData.ChainID = chainID
	chainData.LatestBlockHeight = latestBlockHeightStr
	chainData.LatestBlockHash = latestBlockHash
	chainData.LatestAppHash = latestAppHash
	return chainData
}

func waitAnswer(expected string) {
	buf := bufio.NewReader(os.Stdin)
	fmt.Print("Please input Y when you stop an ABCI process: ")
	input, err := buf.ReadBytes('\n')
	if err != nil {
		fmt.Println(err)
	}
	inputStr := strings.Replace(string(input), "\n", "", -1)
	if inputStr == expected {
		return
	}
	waitAnswer(expected)
}

func readStateDBAndWriteToFile(curChain ChainHistoryDetail) {
	// Variable
	goPath := getEnv("GOPATH", "")
	dbDir := getEnv("DB_NAME", goPath+"/src/github.com/ndidplatform/smart-contract/DB1")
	dbName := "didDB"
	backupDataFileName := getEnv("BACKUP_DATA_FILE", "data")
	backupValidatorFileName := getEnv("BACKUP_VALIDATORS_FILE", "validators")
	chainHistoryFileName := getEnv("CHAIN_HISTORY_FILE", "chain_history")
	backupBlockNumberStr := getEnv("BLOCK_NUMBER", "")
	backupDataDir := getEnv("BACKUP_DATA_DIR", "backup_Data/")

	if backupBlockNumberStr == "" {
		backupBlockNumberStr = curChain.LatestBlockHeight
	}
	backupBlockNumber, err := strconv.ParseInt(backupBlockNumberStr, 10, 64)
	if err != nil {
		panic(err)
	}

	// Delete backup file
	deleteFile(backupDataDir + backupDataFileName + ".txt")
	deleteFile(backupDataDir + backupValidatorFileName + ".txt")
	deleteFile(backupDataDir + chainHistoryFileName + ".txt")

	// Read state db
	db, err := dbm.NewGoLevelDBWithOpts(dbName, dbDir, &opt.Options{ReadOnly: true})
	if err != nil {
		panic(err)
	}

	oldTree := iavl.NewMutableTree(db, 0)
	oldTree.LoadVersion(backupBlockNumber)
	tree, _ := oldTree.GetImmutable(backupBlockNumber)
	_, ndidNodeID := tree.Get(prefixKey([]byte("MasterNDID")))
	totalKV := 0
	tree.Iterate(func(key []byte, value []byte) (stop bool) {
		// Validator
		if strings.Contains(string(key), "val:") {
			var kv did.KeyValue
			kv.Key = key
			kv.Value = value
			jsonStr, err := json.Marshal(kv)
			if err != nil {
				panic(err)
			}
			fWriteLn(backupValidatorFileName, jsonStr, backupDataDir)
			return false
		}
		// Chain history info
		if strings.Contains(string(key), "ChainHistoryInfo") {
			var chainHistory ChainHistory
			if string(value) != "" {
				err := json.Unmarshal([]byte(value), &chainHistory)
				if err != nil {
					panic(err)
				}
			}
			chainHistory.Chains = append(chainHistory.Chains, curChain)
			chainHistoryStr, err := json.Marshal(chainHistory)
			if err != nil {
				panic(err)
			}
			fWriteLn(chainHistoryFileName, chainHistoryStr, backupDataDir)
			return false
		}
		if strings.Contains(string(key), string(ndidNodeID)) {
			return false
		}
		if strings.Contains(string(key), "MasterNDID") {
			return false
		}
		if strings.Contains(string(key), "InitState") {
			return false
		}
		// If key is last block key, not save to backup file
		if strings.Contains(string(key), "lastBlock") {
			return false
		}
		var kv did.KeyValue
		kv.Key = key
		kv.Value = value
		jsonStr, err := json.Marshal(kv)
		if err != nil {
			panic(err)
		}
		fWriteLn(backupDataFileName, jsonStr, backupDataDir)
		totalKV++
		if math.Mod(float64(totalKV), 100) == 0.0 {
			fmt.Printf("Total number of saved kv: %d\n", totalKV)
		}
		return false
	})
	// If key do not have "ChainHistoryInfo" key, create file
	if !tree.Has(prefixKey([]byte("ChainHistoryInfo"))) {
		var chainHistory ChainHistory
		chainHistory.Chains = append(chainHistory.Chains, curChain)
		chainHistoryStr, err := json.Marshal(chainHistory)
		if err != nil {
			panic(err)
		}
		fWriteLn(chainHistoryFileName, chainHistoryStr, backupDataDir)
		totalKV++
	}
	fmt.Printf("Total number of saved kv: %d\n", totalKV)
	fmt.Printf("Total number of kv: %d\n", totalKV)
}

func prefixKey(key []byte) []byte {
	return append(kvPairPrefixKey, key...)
}

func fWriteLn(filename string, data []byte, backupDataDir string) {
	createDirIfNotExist(backupDataDir)
	f, err := os.OpenFile(backupDataDir+filename+".txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString("\r\n")
	if err != nil {
		panic(err)
	}
}

func createDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func deleteFile(dir string) {
	_, err := os.Stat(dir)
	if err != nil {
		return
	}
	err = os.Remove(dir)
	if err != nil {
		panic(err)
	}
}

type ChainHistoryDetail struct {
	ChainID           string `json:"chain_id"`
	LatestBlockHash   string `json:"latest_block_hash"`
	LatestAppHash     string `json:"latest_app_hash"`
	LatestBlockHeight string `json:"latest_block_height"`
}

type ChainHistory struct {
	Chains []ChainHistoryDetail `json:"chains"`
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = defaultValue
	}
	return value
}

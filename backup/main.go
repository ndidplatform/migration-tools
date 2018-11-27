package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ndidplatform/migration-tools/utils"
	did "github.com/ndidplatform/smart-contract/abci/did/v1"
	"github.com/tendermint/iavl"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	kvPairPrefixKey = []byte("kvPairKey:")
)

func main() {
	// Variable
	goPath := getEnv("GOPATH", "")
	dbDir := getEnv("DB_NAME", goPath+"/src/github.com/ndidplatform/smart-contract/DB1")
	dbName := "didDB"
	backupDbDir := getEnv("BACKUP_DB_FILE", "backup_DB")
	backupDataFileName := getEnv("BACKUP_DATA_FILE", "data")
	backupValidatorFileName := getEnv("BACKUP_VALIDATORS_FILE", "validators")
	chainHistoryFileName := getEnv("CHAIN_HISTORY_FILE", "chain_history")
	backupBlockNumberStr := getEnv("BLOCK_NUMBER", "")
	backupDataDir := getEnv("BACKUP_DATA_DIR", "backup_Data/")

	// Delete backup file
	deleteFile(backupDataDir + backupDataFileName + ".txt")
	deleteFile(backupDataDir + backupValidatorFileName + ".txt")
	os.Remove(backupDbDir)
	deleteFile(backupDataDir + chainHistoryFileName + ".txt")

	// Save previous chain info
	resStatus := utils.GetTendermintStatus()
	if backupBlockNumberStr == "" {
		backupBlockNumberStr = resStatus.Result.SyncInfo.LatestBlockHeight
	}
	backupBlockNumber, err := strconv.ParseInt(backupBlockNumberStr, 10, 64)
	if err != nil {
		panic(err)
	}

	blockStatus := utils.GetBlockStatus(backupBlockNumber)
	chainID := blockStatus.Result.Block.Header.ChainID
	latestBlockHeight := blockStatus.Result.Block.Header.Height
	latestBlockHash := blockStatus.Result.BlockMeta.BlockID.Hash
	latestAppHash := blockStatus.Result.Block.Header.AppHash
	fmt.Printf("--- Chain info at block: %s ---\n", backupBlockNumberStr)
	fmt.Println("Chain ID: " + chainID)
	fmt.Println("Latest Block Height: " + latestBlockHeight)
	fmt.Println("Latest Block Hash: " + latestBlockHash)
	fmt.Println("Latest App Hash: " + latestAppHash)

	// Copy stateDB dir
	copyDir(dbDir, backupDbDir)

	// Save kv from backup DB
	db := dbm.NewDB(dbName, "leveldb", backupDbDir)
	oldTree := iavl.NewMutableTree(db, 0)
	oldTree.LoadVersion(backupBlockNumber)
	fmt.Println(oldTree.Version())
	tree, _ := oldTree.GetImmutable(backupBlockNumber)
	_, ndidNodeID := tree.Get(prefixKey([]byte("MasterNDID")))
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
			var prevChain ChainHistoryDetail
			prevChain.ChainID = chainID
			prevChain.LatestBlockHeight = latestBlockHeight
			prevChain.LatestBlockHash = latestBlockHash
			prevChain.LatestAppHash = latestAppHash
			chainHistory.Chains = append(chainHistory.Chains, prevChain)
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
		return false
	})
	// If key do not have "ChainHistoryInfo" key, create file
	if !tree.Has(prefixKey([]byte("ChainHistoryInfo"))) {
		var chainHistory ChainHistory
		var prevChain ChainHistoryDetail
		prevChain.ChainID = chainID
		prevChain.LatestBlockHeight = latestBlockHeight
		prevChain.LatestBlockHash = latestBlockHash
		prevChain.LatestAppHash = latestAppHash
		chainHistory.Chains = append(chainHistory.Chains, prevChain)
		chainHistoryStr, err := json.Marshal(chainHistory)
		if err != nil {
			panic(err)
		}
		fWriteLn(chainHistoryFileName, chainHistoryStr, backupDataDir)
	}
}

func copyDir(source string, dest string) (err error) {
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}
	directory, _ := os.Open(source)
	objects, err := directory.Readdir(-1)
	for _, obj := range objects {
		sourcefilepointer := source + "/" + obj.Name()
		destinationfilepointer := dest + "/" + obj.Name()
		if obj.IsDir() {
			err = copyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err = copyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	return
}

func copyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourcefile.Close()
	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destfile.Close()
	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}
	return
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

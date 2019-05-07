package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gogo/protobuf/proto"
	"github.com/ndidplatform/migration-tools/utils"
	did "github.com/ndidplatform/smart-contract/abci/did/v1"
	"github.com/ndidplatform/smart-contract/protos/data"
	"github.com/tendermint/iavl"
	bcTm "github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	stateTm "github.com/tendermint/tendermint/state"
)

var (
	kvPairPrefixKey = []byte("kvPairKey:")
)

func main() {
	curChainData := getLastestTendermintData()
	readStateDBAndWriteToFile(curChainData)
}

func getLastestTendermintData() (chainData chainHistoryDetail) {
	home := utils.GetEnv("HOME", "")
	tmHome := utils.GetEnv("TM_HOME", path.Join(home, "go/src/github.com/ndidplatform/smart-contract/config/tendermint/IdP"))
	configFile := path.Join(tmHome, "config/config.toml")
	var config tomlConfig
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		fmt.Println(err)
		return
	}
	dbDir := path.Join(tmHome, config.DBPath)
	dbType := dbm.DBBackendType(config.DBBackend)
	stateDB := dbm.NewDB("state", dbType, dbDir)
	state := stateTm.LoadState(stateDB)
	blockDB := dbm.NewDB("blockstore", dbType, dbDir)
	blockStore := bcTm.NewBlockStore(blockDB)
	block := blockStore.LoadBlockMeta(state.LastBlockHeight)
	latestBlockHeight := strconv.FormatInt(state.LastBlockHeight, 10)
	chainID := block.Header.ChainID
	latestBlockHash := block.BlockID.Hash.String()
	latestAppHash := block.Header.AppHash.String()
	fmt.Println("Chain ID: " + chainID)
	fmt.Println("Latest Block Height: " + latestBlockHeight)
	fmt.Println("Latest Block Hash: " + latestBlockHash)
	fmt.Println("Latest App Hash: " + latestAppHash)
	chainData.ChainID = chainID
	chainData.LatestBlockHeight = latestBlockHeight
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

func readStateDBAndWriteToFile(curChain chainHistoryDetail) {
	// Variable
	goPath := utils.GetEnv("GOPATH", "")
	dbType := utils.GetEnv("ABCI_DB_TYPE", "goleveldb")
	dbDir := utils.GetEnv("ABCI_DB_DIR_PATH", path.Join(goPath, "src/github.com/ndidplatform/smart-contract/DB1"))
	dbName := "didDB"
	backupDataFileName := utils.GetEnv("BACKUP_DATA_FILE_NAME", "data")
	backupValidatorFileName := utils.GetEnv("BACKUP_VALIDATORS_FILE_NAME", "validators")
	chainHistoryFileName := utils.GetEnv("CHAIN_HISTORY_FILE_NAME", "chain_history")
	backupBlockNumberStr := utils.GetEnv("BLOCK_NUMBER", "")
	backupDataDir := utils.GetEnv("BACKUP_DATA_DIR", "backup_Data/")

	if backupBlockNumberStr == "" {
		backupBlockNumberStr = curChain.LatestBlockHeight
	}
	backupBlockNumber, err := strconv.ParseInt(backupBlockNumberStr, 10, 64)
	if err != nil {
		panic(err)
	}
	// Delete backup file
	utils.DeleteFile(backupDataDir + backupDataFileName + ".txt")
	utils.DeleteFile(backupDataDir + backupValidatorFileName + ".txt")
	utils.DeleteFile(backupDataDir + chainHistoryFileName + ".txt")
	// Read state db
	db := dbm.NewDB(dbName, dbm.DBBackendType(dbType), dbDir)
	oldTree := iavl.NewMutableTree(db, 0)
	oldTree.LoadVersion(backupBlockNumber)
	tree, _ := oldTree.GetImmutable(backupBlockNumber)
	_, ndidNodeID := tree.Get(utils.PrefixKey([]byte("MasterNDID")))
	totalKV := 0
	tree.Iterate(func(key []byte, value []byte) (stop bool) {
		// Delete prefix
		if bytes.Contains(key, kvPairPrefixKey) {
			key = bytes.TrimPrefix(key, kvPairPrefixKey)
		}
		switch {
		case strings.Contains(string(key), "val:"):
			// Validator
			var kv did.KeyValue
			kv.Key = key
			kv.Value = value
			jsonStr, err := json.Marshal(kv)
			if err != nil {
				panic(err)
			}
			utils.FWriteLn(backupValidatorFileName, jsonStr, backupDataDir)
		case strings.Contains(string(key), "ChainHistoryInfo"):
			var chainHistory chainHistory
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
			utils.FWriteLn(chainHistoryFileName, chainHistoryStr, backupDataDir)
		case strings.Contains(string(key), string(ndidNodeID)):
		case strings.Contains(string(key), "MasterNDID"):
		case strings.Contains(string(key), "InitState"):
		case strings.Contains(string(key), "lastBlock"):
		case strings.Contains(string(key), "Proxy|"):
		case strings.Contains(string(key), "NodeID|"):
			// If key is node detail, check node is behind proxy.
			// If node is behind proxy, set proxy ID and proxy config
			splitedKey := strings.Split(string(key), "|")
			proxyKey := "Proxy" + "|" + splitedKey[1]
			_, proxyValue := tree.Get(utils.PrefixKey([]byte(proxyKey)))
			if proxyValue != nil {
				var proxy data.Proxy
				err := proto.Unmarshal([]byte(proxyValue), &proxy)
				if err != nil {
					panic(err)
				}
				splitedKey := strings.Split(string(key), "|")
				nodeDetailKey := "NodeID" + "|" + splitedKey[1]
				_, nodeDetailValue := tree.Get(utils.PrefixKey([]byte(nodeDetailKey)))
				var nodeDetail data.NodeDetail
				err = proto.Unmarshal([]byte(nodeDetailValue), &nodeDetail)
				if err != nil {
					panic(err)
				}
				nodeDetail.ProxyNodeId = proxy.ProxyNodeId
				nodeDetail.ProxyConfig = proxy.Config
				value, err = utils.ProtoDeterministicMarshal(&nodeDetail)
				if err != nil {
					panic(err)
				}
			}
		case strings.Contains(string(key), "Request") && !strings.Contains(string(key), "TokenPriceFunc"):
			// If key is about request, Save version of value and update key
			versionsKeyStr := string(key) + "|versions"
			versionsKey := []byte(versionsKeyStr)
			var versions []int64
			versions = append(versions, 1)
			var keyVersions data.KeyVersions
			keyVersions.Versions = versions
			value, err := utils.ProtoDeterministicMarshal(&keyVersions)
			if err != nil {
				panic(err)
			}
			var kv did.KeyValue
			kv.Key = versionsKey
			kv.Value = value
			jsonStr, err := json.Marshal(kv)
			if err != nil {
				panic(err)
			}
			utils.FWriteLn(backupDataFileName, jsonStr, backupDataDir)
			totalKV++

			key = []byte(string(key) + "|" + "1")
		default:
			var kv did.KeyValue
			kv.Key = key
			kv.Value = value
			jsonStr, err := json.Marshal(kv)
			if err != nil {
				panic(err)
			}
			utils.FWriteLn(backupDataFileName, jsonStr, backupDataDir)
			totalKV++
			if math.Mod(float64(totalKV), 100) == 0.0 {
				fmt.Printf("Total number of saved kv: %d\n", totalKV)
			}
		}
		return
	})
	// If key do not have "ChainHistoryInfo" key, create file
	if !tree.Has(utils.PrefixKey([]byte("ChainHistoryInfo"))) {
		var chainHistory chainHistory
		chainHistory.Chains = append(chainHistory.Chains, curChain)
		chainHistoryStr, err := json.Marshal(chainHistory)
		if err != nil {
			panic(err)
		}
		utils.FWriteLn(chainHistoryFileName, chainHistoryStr, backupDataDir)
		totalKV++
	}
	fmt.Printf("Total number of saved kv: %d\n", totalKV)
	fmt.Printf("Total number of kv: %d\n", totalKV)
}

type chainHistoryDetail struct {
	ChainID           string `json:"chain_id"`
	LatestBlockHash   string `json:"latest_block_hash"`
	LatestAppHash     string `json:"latest_app_hash"`
	LatestBlockHeight string `json:"latest_block_height"`
}

type chainHistory struct {
	Chains []chainHistoryDetail `json:"chains"`
}

type tomlConfig struct {
	DBBackend string `toml:"db_backend"`
	DBPath    string `toml:"db_dir"`
}


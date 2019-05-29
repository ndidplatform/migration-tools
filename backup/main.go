package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/ndidplatform/migration-tools/utils"

	"github.com/BurntSushi/toml"
	"github.com/gogo/protobuf/proto"
	didProtoV2 "github.com/ndidplatform/migration-tools/protos/dataV2"
	didProtoV3 "github.com/ndidplatform/migration-tools/protos/dataV3"
	bcTm "github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	stateTm "github.com/tendermint/tendermint/state"
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
	// Delete backup file
	utils.DeleteFile(backupDataDir + backupDataFileName + ".txt")
	utils.DeleteFile(backupDataDir + backupValidatorFileName + ".txt")
	utils.DeleteFile(backupDataDir + chainHistoryFileName + ".txt")
	// Read state db
	db := dbm.NewDB(dbName, dbm.DBBackendType(dbType), dbDir)
	ndidNodeID := db.Get([]byte("MasterNDID"))
	totalKV := 0
	itr := db.Iterator(nil, nil)
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()
		// Delete prefix
		if bytes.Contains(key, utils.KvPairPrefixKey) {
			key = bytes.TrimPrefix(key, utils.KvPairPrefixKey)
		}
		switch {
		case strings.Contains(string(key), "lastBlock"):
			// Last block
			// Do not save
		case strings.Contains(string(key), string(ndidNodeID)) && !strings.Contains(string(key), "MasterNDID"):
			// NDID node detail
			// Do not save
		case strings.Contains(string(key), "MasterNDID"):
			// NDID
			// Do not save
		case strings.Contains(string(key), "InitState"):
			// Init state
			// Do not save
		case strings.Contains(string(key), "ProvideService") || strings.Contains(string(key), "ServiceDestination"):
			// AS need to RegisterServiceDestination after migrate chain completed
			// Do not save
		case strings.Contains(string(key), "Accessor"):
			// All key that have associate with Accessor
			// Do not save
		case strings.Contains(string(key), "val:"):
			// Validator
			writeKeyValue(backupValidatorFileName, backupDataDir, key, value)
			totalKV++
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
			totalKV++
		case strings.Contains(string(key), "NodeID"):
			// Node detail
			// Update to new version of proto
			var nodeDetailV2 didProtoV2.NodeDetail
			var nodeDetailV3 didProtoV3.NodeDetail
			err := proto.Unmarshal(value, &nodeDetailV2)
			if err != nil {
				panic(err)
			}
			nodeDetailV3.PublicKey = nodeDetailV2.PublicKey
			nodeDetailV3.MasterPublicKey = nodeDetailV2.MasterPublicKey
			nodeDetailV3.NodeName = nodeDetailV2.NodeName
			nodeDetailV3.Role = nodeDetailV2.Role
			for _, mq := range nodeDetailV2.Mq {
				var newMq didProtoV3.MQ
				newMq.Ip = mq.Ip
				newMq.Port = mq.Port
				nodeDetailV3.Mq = append(nodeDetailV3.Mq, &newMq)
			}
			nodeDetailV3.Active = nodeDetailV2.Active
			nodeDetailV3.ProxyNodeId = nodeDetailV2.ProxyNodeId
			nodeDetailV3.ProxyConfig = nodeDetailV2.ProxyConfig
			nodeDetailV3.MaxIal = nodeDetailV2.MaxIal
			nodeDetailV3.MaxAal = nodeDetailV2.MaxAal
			nodeDetailV3.SupportedRequestMessageDataUrlTypeList = make([]string, 0)
			newValue, err := utils.ProtoDeterministicMarshal(&nodeDetailV3)
			if err != nil {
				panic(err)
			}
			writeKeyValue(backupDataFileName, backupDataDir, key, newValue)
			totalKV++
		case strings.Contains(string(key), "Request") && strings.Contains(string(key), "versions"):
			// Versions of request
			var keyVersions didProtoV3.KeyVersions
			err := proto.Unmarshal([]byte(value), &keyVersions)
			if err != nil {
				panic(err)
			}
			lastVer := strconv.FormatInt(keyVersions.Versions[len(keyVersions.Versions)-1], 10)
			partOfKey := strings.Split(string(key), "|")
			reqID := partOfKey[1]

			// Get last version of request detail
			reqDetailKey := "Request" + "|" + reqID + "|" + lastVer
			reqDetailValue := db.Get([]byte(reqDetailKey))

			var requestV2 didProtoV2.Request
			var requestV3 didProtoV3.Request
			err = proto.Unmarshal(reqDetailValue, &requestV2)
			if err != nil {
				panic(err)
			}
			requestV3.RequestId = requestV2.RequestId
			requestV3.MinIdp = requestV2.MinIdp
			requestV3.MinAal = requestV2.MinAal
			requestV3.MinIal = requestV2.MinIal
			requestV3.RequestTimeout = requestV2.RequestTimeout
			requestV3.IdpIdList = requestV2.IdpIdList
			for _, dataReq := range requestV2.DataRequestList {
				var newDataReq didProtoV3.DataRequest
				newDataReq.ServiceId = dataReq.ServiceId
				newDataReq.AsIdList = dataReq.AsIdList
				newDataReq.MinAs = dataReq.MinAs
				newDataReq.RequestParamsHash = dataReq.RequestParamsHash
				newDataReq.AnsweredAsIdList = dataReq.AnsweredAsIdList
				newDataReq.ReceivedDataFromList = dataReq.ReceivedDataFromList
				requestV3.DataRequestList = append(requestV3.DataRequestList, &newDataReq)
			}
			requestV3.RequestMessageHash = requestV2.RequestMessageHash
			for _, res := range requestV2.ResponseList {
				var newDataRes didProtoV3.Response
				newDataRes.Ial = res.Ial
				newDataRes.Aal = res.Aal
				newDataRes.Status = res.Status
				newDataRes.Signature = res.Signature
				newDataRes.IdpId = res.IdpId
				newDataRes.ValidIal = res.ValidIal
				newDataRes.ValidSignature = res.ValidSignature
				requestV3.ResponseList = append(requestV3.ResponseList, &newDataRes)
			}
			requestV3.Closed = requestV2.Closed
			requestV3.TimedOut = requestV2.TimedOut
			requestV3.Purpose = requestV2.Purpose
			requestV3.Owner = requestV2.Owner
			requestV3.Mode = int32(requestV2.Mode)
			requestV3.UseCount = requestV2.UseCount
			requestV3.CreationBlockHeight = requestV2.CreationBlockHeight
			requestV3.ChainId = requestV2.ChainId
			newReqDetailValue, err := utils.ProtoDeterministicMarshal(&requestV3)
			if err != nil {
				panic(err)
			}
			// Set to 1 version
			keyVersions.Versions = append(make([]int64, 0), 1)
			newReqVersionsValue, err := utils.ProtoDeterministicMarshal(&keyVersions)
			if err != nil {
				panic(err)
			}
			newReqDetailKey := "Request" + "|" + reqID + "|" + "1"
			// Write request detail and Version of request detail
			writeKeyValue(backupDataFileName, backupDataDir, []byte(newReqDetailKey), newReqDetailValue)
			totalKV++
			writeKeyValue(backupDataFileName, backupDataDir, key, newReqVersionsValue)
			totalKV++ 
		case strings.Contains(string(key), "AllNamespace"):
			// Namespace list
			var namespaceV2 didProtoV2.NamespaceList
			var namespaceV3 didProtoV3.NamespaceList
			err := proto.Unmarshal(value, &namespaceV2)
			if err != nil {
				panic(err)
			}
			for _, namespace := range namespaceV2.Namespaces {
				var newNamesapce didProtoV3.Namespace
				newNamesapce.Namespace = namespace.Namespace
				newNamesapce.Description = namespace.Description
				newNamesapce.Active = namespace.Active
				newNamesapce.AllowedIdentifierCountInReferenceGroup = 1
				newNamesapce.AllowedActiveIdentifierCountInReferenceGroup = 1
				namespaceV3.Namespaces = append(namespaceV3.Namespaces, &newNamesapce)
			}
			newValue, err := utils.ProtoDeterministicMarshal(&namespaceV3)
			if err != nil {
				panic(err)
			}
			writeKeyValue(backupDataFileName, backupDataDir, key, newValue)
			totalKV++
		default:
			writeKeyValue(backupDataFileName, backupDataDir, key, value)
			totalKV++
		}
	}
	fmt.Printf("Total number of saved kv: %d\n", totalKV)
	fmt.Printf("Total number of kv: %d\n", totalKV)
}

func writeKeyValue(filename string, backupDataDir string, key, value []byte) {
	var kv utils.KeyValue
	kv.Key = key
	kv.Value = value
	jsonStr, err := json.Marshal(kv)
	if err != nil {
		panic(err)
	}
	utils.FWriteLn(filename, jsonStr, backupDataDir)
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


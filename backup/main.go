package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ndidplatform/migration-tools/utils"

	"github.com/BurntSushi/toml"
	"github.com/gogo/protobuf/proto"

	didProtoV4 "github.com/ndidplatform/migration-tools/protos/dataV4"
	didProtoV5 "github.com/ndidplatform/migration-tools/protos/dataV5"
	bcTm "github.com/tendermint/tendermint/blockchain"
	dbm "github.com/tendermint/tendermint/libs/db"
	stateTm "github.com/tendermint/tendermint/state"
)

func main() {
	curChainData := getLastestTendermintData()
	readStateDBAndWriteToFile(curChainData)
}

func getLastestTendermintData() (chainData chainHistoryDetail) {
	curDir, _ := os.Getwd()
	tmHome := utils.GetEnv("TM_HOME", path.Join(curDir, "../smart-contract/config/tendermint/IdP"))
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
	curDir, _ := os.Getwd()
	dbType := utils.GetEnv("ABCI_DB_TYPE", "goleveldb")
	dbDir := utils.GetEnv("ABCI_DB_DIR_PATH", path.Join(curDir, "../smart-contract/DB1"))
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
		case strings.Contains(string(key), "IdentityProof"):
			// Identity proof
			// Do not save
		case strings.Contains(string(key), "Accessor"):
			// All key that have associate with Accessor
			// Do not save
		case strings.Contains(string(key), "Request") && !strings.Contains(string(key), "versions"):
			// Request detail
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
		case strings.Contains(string(key), "Request") && strings.Contains(string(key), "versions"):
			// Versions of request
			var keyVersionsV4 didProtoV4.KeyVersions
			err := proto.Unmarshal([]byte(value), &keyVersionsV4)
			if err != nil {
				panic(err)
			}
			latestVersion := strconv.FormatInt(keyVersionsV4.Versions[len(keyVersionsV4.Versions)-1], 10)
			keyParts := strings.Split(string(key), "|")
			requestID := keyParts[1]

			// Get last version of request detail
			requestV4Key := "Request" + "|" + requestID + "|" + latestVersion
			requestV4Value := db.Get([]byte(requestV4Key))

			var requestV4 didProtoV4.Request
			if err := proto.Unmarshal([]byte(requestV4Value), &requestV4); err != nil {
				panic(err)
			}

			// data request, AS responses
			var dataRequestListV5 []*didProtoV5.DataRequest = make([]*didProtoV5.DataRequest, 0)
			for _, dataRequestV4 := range requestV4.DataRequestList {
				responseListV5 := make([]*didProtoV5.ASResponse, 0)
				for _, asID := range dataRequestV4.AnsweredAsIdList {
					receivedData := false
					for _, receivedDataAsID := range dataRequestV4.ReceivedDataFromList {
						if asID == receivedDataAsID {
							receivedData = true
							break
						}
					}
					responseListV5 = append(responseListV5, &didProtoV5.ASResponse{
						AsId:         asID,
						Signed:       true,
						ReceivedData: receivedData,
						ErrorCode:    0,
					})
				}

				dataRequestV5 := &didProtoV5.DataRequest{
					ServiceId:         dataRequestV4.ServiceId,
					AsIdList:          dataRequestV4.AsIdList,
					MinAs:             dataRequestV4.MinAs,
					RequestParamsHash: dataRequestV4.RequestParamsHash,
					ResponseList:      responseListV5,
				}
				dataRequestListV5 = append(dataRequestListV5, dataRequestV5)
			}

			// IdP responses
			var responseListV5 []*didProtoV5.Response = make([]*didProtoV5.Response, 0)
			for _, responseV4 := range requestV4.ResponseList {
				responseV5 := &didProtoV5.Response{
					Ial:            responseV4.Ial,
					Aal:            responseV4.Aal,
					Status:         responseV4.Status,
					Signature:      responseV4.Signature,
					IdpId:          responseV4.IdpId,
					ValidIal:       responseV4.ValidIal,
					ValidSignature: responseV4.ValidSignature,
					ErrorCode:      0,
				}
				responseListV5 = append(responseListV5, responseV5)
			}

			var requestV5 didProtoV5.Request = didProtoV5.Request{
				RequestId:           requestV4.RequestId,
				MinIdp:              requestV4.MinIdp,
				MinAal:              requestV4.MinAal,
				MinIal:              requestV4.MinIal,
				RequestTimeout:      requestV4.RequestTimeout,
				IdpIdList:           requestV4.IdpIdList,
				DataRequestList:     dataRequestListV5,
				RequestMessageHash:  requestV4.RequestMessageHash,
				ResponseList:        responseListV5,
				Closed:              requestV4.Closed,
				TimedOut:            requestV4.TimedOut,
				Purpose:             requestV4.Purpose,
				Owner:               requestV4.Owner,
				Mode:                requestV4.Mode,
				UseCount:            requestV4.UseCount,
				CreationBlockHeight: requestV4.CreationBlockHeight,
				ChainId:             requestV4.ChainId,
			}

			requestV5Bytes, err := utils.ProtoDeterministicMarshal(&requestV5)
			if err != nil {
				panic(err)
			}

			// Set to 1 version
			var keyVersionsV5 didProtoV5.KeyVersions = didProtoV5.KeyVersions{
				Versions: append(make([]int64, 0), 1),
			}
			newReqVersionsValue, err := utils.ProtoDeterministicMarshal(&keyVersionsV5)
			if err != nil {
				panic(err)
			}
			newReqDetailKey := "Request" + "|" + requestID + "|" + "1"
			// Write request detail and Version of request detail
			writeKeyValue(backupDataFileName, backupDataDir, []byte(newReqDetailKey), requestV5Bytes)
			totalKV++
			writeKeyValue(backupDataFileName, backupDataDir, key, newReqVersionsValue)
			totalKV++
		case strings.HasPrefix(string(key), "NodeID"):
			// Add information (IdpAgent, UseWhitelist, Whitelist) to every node
			var nodeDetailV4 didProtoV4.NodeDetail
			if err := proto.Unmarshal([]byte(value), &nodeDetailV4); err != nil {
				panic(err)
			}

			mqV5 := make([]*didProtoV5.MQ, 0, len(nodeDetailV4.Mq))
			for _, mqV4 := range nodeDetailV4.Mq {
				mqV5 = append(mqV5, &didProtoV5.MQ{
					Ip:   mqV4.Ip,
					Port: mqV4.Port,
				})
			}
			nodeDetailV5 := didProtoV5.NodeDetail{
				PublicKey:                              nodeDetailV4.PublicKey,
				MasterPublicKey:                        nodeDetailV4.MasterPublicKey,
				NodeName:                               nodeDetailV4.NodeName,
				Role:                                   nodeDetailV4.Role,
				MaxIal:                                 nodeDetailV4.MaxIal,
				MaxAal:                                 nodeDetailV4.MaxAal,
				Mq:                                     mqV5,
				Active:                                 nodeDetailV4.Active,
				ProxyNodeId:                            nodeDetailV4.ProxyNodeId,
				ProxyConfig:                            nodeDetailV4.ProxyConfig,
				SupportedRequestMessageDataUrlTypeList: nodeDetailV4.SupportedRequestMessageDataUrlTypeList,
				IsIdpAgent:                             false,
				UseWhitelist:                           false,
				Whitelist:                              []string{},
			}

			nodeDetailV5Byte, err := utils.ProtoDeterministicMarshal(&nodeDetailV5)
			if err != nil {
				panic(err)
			}
			writeKeyValue(backupDataFileName, backupDataDir, key, nodeDetailV5Byte)
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

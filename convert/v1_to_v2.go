package convert

// Need to read from IAVL tree

// import (
// 	"bytes"
// 	"encoding/json"
// 	"strings"

// 	"github.com/spf13/viper"

// 	v1 "github.com/ndidplatform/migration-tools/did/v1"
// 	didProtoV1 "github.com/ndidplatform/migration-tools/did/v1/protos/data"
// 	"github.com/ndidplatform/migration-tools/did/v2/protos/data"
// 	didProtoV2 "github.com/ndidplatform/migration-tools/did/v2/protos/data"
// 	"github.com/ndidplatform/migration-tools/proto"
// 	"github.com/ndidplatform/migration-tools/utils"
// )

// func V1ConvertAndBackupStateDBData(
// 	saveNewChainHistory func(chainHistory []byte) (err error),
// 	saveKeyValue func(key []byte, value []byte) (err error),
// ) (err error) {
// 	tmHome := viper.GetString("TM_HOME")
// 	currentChainData, err := v1.GetLastestTendermintData(tmHome)
// 	if err != nil {
// 		return err
// 	}

// 	dbType := viper.GetString("ABCI_DB_TYPE")
// 	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
// 	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

// 	// Delete existing backup files
// 	// utils.DeleteFile(backupDataDir + backupDataFileName + ".txt")
// 	// utils.DeleteFile(backupDataDir + backupValidatorFileName + ".txt")
// 	// utils.DeleteFile(backupDataDir + chainHistoryFileName + ".txt")

// 	v1StateDB := v1.GetStateDB(dbType, dbDir)
// 	ndidNodeID, err := v1StateDB.Get([]byte("MasterNDID"))
// 	if err != nil {
// 		return err
// 	}

// 	dbGet := func(key []byte) (value []byte, err error) {
// 		return v1StateDB.Get(key)
// 	}

// 	itr, err := v1StateDB.Iterator(nil, nil)
// 	if err != nil {
// 		return err
// 	}
// 	for ; itr.Valid(); itr.Next() {
// 		key := itr.Key()
// 		value := itr.Value()

// 		err := V1ConvertStateDBDataToV2(
// 			key,
// 			value,
// 			string(ndidNodeID),
// 			currentChainData,
// 			dbGet,
// 			saveNewChainHistory,
// 			saveKeyValue,
// 		)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// func V1ConvertStateDBDataToV2(
// 	key []byte,
// 	value []byte,
// 	ndidNodeID string,
// 	currentChainData *v1.ChainHistoryDetail,
// 	dbGet func(key []byte) (value []byte, err error),
// 	saveNewChainHistory func(chainHistory []byte) (err error),
// 	saveKeyValue func(key []byte, value []byte) (err error),
// ) (err error) {
// 	// Delete prefix
// 	if bytes.Contains(key, v1.KvPairPrefixKey) {
// 		key = bytes.TrimPrefix(key, v1.KvPairPrefixKey)
// 	}
// 	switch {
// 	case strings.Contains(string(key), "val:"):
// 		// Validator
// 		err := saveKeyValue(key, value)
// 		if err != nil {
// 			return err
// 		}
// 	case strings.Contains(string(key), "ChainHistoryInfo"):
// 		var chainHistory v1.ChainHistory
// 		if string(value) != "" {
// 			err := json.Unmarshal([]byte(value), &chainHistory)
// 			if err != nil {
// 				return err
// 			}
// 		}

// 		// TODO: refactor same chain history structure or separate by version (actually convert from one to another)?

// 		if currentChainData != nil {
// 			chainHistory.Chains = append(chainHistory.Chains, *currentChainData)
// 		}
// 		chainHistoryStr, err := json.Marshal(chainHistory)
// 		if err != nil {
// 			return err
// 		}
// 		err = saveNewChainHistory(chainHistoryStr)
// 		if err != nil {
// 			return err
// 		}
// 	case strings.Contains(string(key), string(ndidNodeID)):
// 	case strings.Contains(string(key), "MasterNDID"):
// 	case strings.Contains(string(key), "InitState"):
// 	case strings.Contains(string(key), "lastBlock"):
// 	case strings.Contains(string(key), "Proxy|"):
// 	case strings.Contains(string(key), "NodeID|"):
// 		// If key is node detail, check node is behind proxy.
// 		// If node is behind proxy, set proxy ID and proxy config
// 		splitedKey := strings.Split(string(key), "|")
// 		proxyKey := "Proxy" + "|" + splitedKey[1]
// 		_, proxyValue := tree.Get(utils.PrefixKey([]byte(proxyKey)))
// 		if proxyValue != nil {
// 			var proxy didProtoV1.Proxy
// 			err := proto.Unmarshal([]byte(proxyValue), &proxy)
// 			if err != nil {
// 				return err
// 			}
// 			splitedKey := strings.Split(string(key), "|")
// 			nodeDetailKey := "NodeID" + "|" + splitedKey[1]
// 			_, nodeDetailValue := tree.Get(utils.PrefixKey([]byte(nodeDetailKey)))
// 			var nodeDetail didProtoV2.NodeDetail
// 			err = proto.Unmarshal([]byte(nodeDetailValue), &nodeDetail)
// 			if err != nil {
// 				return err
// 			}
// 			nodeDetail.ProxyNodeId = proxy.ProxyNodeId
// 			nodeDetail.ProxyConfig = proxy.Config
// 			value, err = proto.DeterministicMarshal(&nodeDetail)
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	case strings.Contains(string(key), "Request") && !strings.Contains(string(key), "TokenPriceFunc"):
// 		// If key is about request, Save version of value and update key
// 		versionsKeyStr := string(key) + "|versions"
// 		versionsKey := []byte(versionsKeyStr)
// 		var versions []int64
// 		versions = append(versions, 1)
// 		var keyVersions data.KeyVersions
// 		keyVersions.Versions = versions
// 		value, err := proto.DeterministicMarshal(&keyVersions)
// 		if err != nil {
// 			return err
// 		}

// 		err = saveKeyValue(versionsKey, value)
// 		if err != nil {
// 			return err
// 		}

// 		key = []byte(string(key) + "|" + "1")
// 	default:
// 		err := saveKeyValue(key, value)
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

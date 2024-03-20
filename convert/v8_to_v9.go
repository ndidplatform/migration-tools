/**
 * Copyright (c) 2018, 2019 National Digital ID COMPANY LIMITED
 *
 * This file is part of NDID software.
 *
 * NDID is the free software: you can redistribute it and/or modify it under
 * the terms of the Affero GNU General Public License as published by the
 * Free Software Foundation, either version 3 of the License, or any later
 * version.
 *
 * NDID is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 * See the Affero GNU General Public License for more details.
 *
 * You should have received a copy of the Affero GNU General Public License
 * along with the NDID source code. If not, see https://www.gnu.org/licenses/agpl.txt.
 *
 * Please contact info@ndid.co.th for any further questions
 *
 */

package convert

import (
	"bytes"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/spf13/viper"

	v8 "github.com/ndidplatform/migration-tools/did/v8"
	didProtoV8 "github.com/ndidplatform/migration-tools/did/v8/protos/data"
	v9 "github.com/ndidplatform/migration-tools/did/v9"
	didProtoV9 "github.com/ndidplatform/migration-tools/did/v9/protos/data"
	v9types "github.com/ndidplatform/migration-tools/did/v9/types"
	"github.com/ndidplatform/migration-tools/proto"
)

var knownKeysV8 []string = []string{
	"MasterNDID",
	"InitState",
	"lastBlock",
	"IdPList",
	"AllNamespace",
	"ServicePriceMinEffectiveDatetimeDelay",

	"ChainHistoryInfo",
	"TimeOutBlockRegisterIdentity",
	"AllowedMinIalForRegisterIdentityAtFirstIdp",
	"rpList",
	"asList",
	"allList",
	"AllService",

	"n", // nonce
	"NodeID",
	"BehindProxyNode",
	"Token",
	"TokenPriceFunc",
	"Service",
	"ServiceDestination",
	"ApproveKey",
	"ProvideService",
	"RefGroupCode",
	"identityToRefCodeKey",
	"accessorToRefCodeKey",
	"AllowedModeList",
	"Request",
	"Message",
	"SignData",
	"ErrorCode",
	"ErrorCodeList",
	"ServicePriceCeiling",
	"ServicePriceMinEffectiveDatetimeDelay",
	"ServicePriceListKey",
	"RequestType",
	"SuppressedIdentityModificationNotificationNode",

	"Validator",
}

var nodeSupportedFeatureOnTheFly = "on_the_fly"

func ConvertInputStateDBDataV8ToV9AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v8.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v8StateDB := v8.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v8StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v8StateDB.Get(key)
	}

	var keyTypeStats map[string]int64 = make(map[string]int64)

	var keysRead int64 = 0

	itr, err := v8StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		keyPrefix, err := ConvertStateDBDataV8ToV9(
			key,
			value,
			string(ndidNodeID),
			currentChainData,
			dbGet,
			saveNewChainHistory,
			saveKeyValue,
		)
		if err != nil {
			return err
		}
		keysRead++
		if keyPrefix != "" {
			keyTypeStats[keyPrefix]++
		}
	}

	log.Println("total key read:", keysRead)
	log.Println("key type stats:", keyTypeStats)

	//
	// data with new keys
	//

	log.Println("adding new state data")

	_, err = AddNewStateDataToV9(
		dbGet,
		saveNewChainHistory,
		saveKeyValue,
	)
	if err != nil {
		return err
	}

	return nil
}

// var v8SavedRequestID map[string]bool = make(map[string]bool)

func ConvertStateDBDataV8ToV9(
	key []byte,
	value []byte,
	ndidNodeID string,
	currentChainData *v8.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (keyType string, err error) {
	// Delete prefix
	if bytes.Contains(key, v8.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v8.KvPairPrefixKey)
	}
	switch {
	case strings.HasPrefix(string(key), "stateKey"):
		// ABCI state metadata
		// Do not save
	case strings.HasPrefix(string(key), "lastBlock"):
		// Last block
		// Do not save
	case ndidNodeID != "" && !strings.HasPrefix(string(key), "MasterNDID") && strings.Contains(string(key), string(ndidNodeID)):
		// NDID node detail
		// Do not save
	case strings.HasPrefix(string(key), "MasterNDID"):
		// NDID
		// Do not save
	case strings.HasPrefix(string(key), "InitState"):
		// Init state
		// Do not save
	case strings.HasPrefix(string(key), "Request") && !strings.HasSuffix(string(key), "versions") &&
		!strings.HasPrefix(string(key), "RequestType"):
		// Request detail
		// Do not save
	case strings.HasPrefix(string(key), "Validator"):
		// Validator
		// Do not save

		// err = saveKeyValue(key, value)
		// if err != nil {
		// 	return err
		// }
	case strings.HasPrefix(string(key), "ChainHistoryInfo"):
		var chainHistory v8.ChainHistory
		if string(value) != "" {
			err := json.Unmarshal([]byte(value), &chainHistory)
			if err != nil {
				panic(err)
			}
		}

		// TODO: refactor same chain history structure or separate by version (actually convert from one to another)?

		if currentChainData != nil {
			chainHistory.Chains = append(chainHistory.Chains, *currentChainData)
		}
		chainHistoryStr, err := json.Marshal(chainHistory)
		if err != nil {
			panic(err)
		}
		err = saveNewChainHistory(chainHistoryStr)
		if err != nil {
			return "", err
		}
	case strings.HasPrefix(string(key), "NodeID"):
		keyType = "NodeID"

		// Changes:
		// - Remove "OnTheFlySupport", Add/change to "SupportedFeatureList"
		// - Remove "PublicKey" and "MasterPublicKey", Change to "SigningPublicKey", "SigningMasterPublicKey", and "EncryptionPublicKey"

		keyParts := strings.Split(string(key), "|")
		nodeID := keyParts[1]

		var nodeDetailV8 didProtoV8.NodeDetail
		if err := proto.Unmarshal([]byte(value), &nodeDetailV8); err != nil {
			panic(err)
		}

		mqV9 := make([]*didProtoV9.MQ, 0, len(nodeDetailV8.Mq))
		for _, mqV8 := range nodeDetailV8.Mq {
			mqV9 = append(mqV9, &didProtoV9.MQ{
				Ip:   mqV8.Ip,
				Port: mqV8.Port,
			})
		}
		supportedFeatureListV9 := make([]string, 0)
		if nodeDetailV8.OnTheFlySupport {
			supportedFeatureListV9 = append(supportedFeatureListV9, nodeSupportedFeatureOnTheFly)
		}
		nodeDetailV9 := didProtoV9.NodeDetail{
			SigningPublicKey: &didProtoV9.NodeKey{
				PublicKey:           nodeDetailV8.PublicKey,
				Algorithm:           string(v9types.SignatureAlgorithmRSAPKCS1V15SHA256),
				Version:             1,
				CreationBlockHeight: 0,
				CreationChainId:     "",
				Active:              true,
			},
			SigningMasterPublicKey: &didProtoV9.NodeKey{
				PublicKey:           nodeDetailV8.MasterPublicKey,
				Algorithm:           string(v9types.SignatureAlgorithmRSAPKCS1V15SHA256),
				Version:             1,
				CreationBlockHeight: 0,
				CreationChainId:     "",
				Active:              true,
			},
			EncryptionPublicKey: &didProtoV9.NodeKey{
				PublicKey:           nodeDetailV8.PublicKey,
				Algorithm:           "RSAES_PKCS1_V1_5",
				Version:             1,
				CreationBlockHeight: 0,
				CreationChainId:     "",
				Active:              true,
			},
			NodeName:                               nodeDetailV8.NodeName,
			Role:                                   nodeDetailV8.Role,
			MaxIal:                                 nodeDetailV8.MaxIal,
			MaxAal:                                 nodeDetailV8.MaxAal,
			Mq:                                     mqV9,
			Active:                                 nodeDetailV8.Active,
			ProxyNodeId:                            nodeDetailV8.ProxyNodeId,
			ProxyConfig:                            nodeDetailV8.ProxyConfig,
			SupportedRequestMessageDataUrlTypeList: nodeDetailV8.SupportedRequestMessageDataUrlTypeList,
			IsIdpAgent:                             false,
			UseWhitelist:                           false,
			Whitelist:                              []string{},
			SupportedFeatureList:                   supportedFeatureListV9,
		}

		nodeDetailV9Byte, err := proto.DeterministicMarshal(&nodeDetailV9)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue(key, nodeDetailV9Byte)
		if err != nil {
			return "", err
		}

		//
		// key history
		//

		nodeKeyKey :=
			v9.NodeKeyKeyPrefix + v9.KeySeparator +
				"signing" + v9.KeySeparator +
				nodeID + v9.KeySeparator +
				strconv.FormatInt(nodeDetailV9.SigningPublicKey.Version, 10)
		nodeKeyV9Byte, err := proto.DeterministicMarshal(nodeDetailV9.SigningPublicKey)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue([]byte(nodeKeyKey), nodeKeyV9Byte)
		if err != nil {
			return "", err
		}

		nodeKeyKey =
			v9.NodeKeyKeyPrefix + v9.KeySeparator +
				"signing_master" + v9.KeySeparator +
				nodeID + v9.KeySeparator +
				strconv.FormatInt(nodeDetailV9.SigningMasterPublicKey.Version, 10)
		nodeKeyV9Byte, err = proto.DeterministicMarshal(nodeDetailV9.SigningMasterPublicKey)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue([]byte(nodeKeyKey), nodeKeyV9Byte)
		if err != nil {
			return "", err
		}

		nodeKeyKey =
			v9.NodeKeyKeyPrefix + v9.KeySeparator +
				"encryption" + v9.KeySeparator +
				nodeID + v9.KeySeparator +
				strconv.FormatInt(nodeDetailV9.EncryptionPublicKey.Version, 10)
		nodeKeyV9Byte, err = proto.DeterministicMarshal(nodeDetailV9.EncryptionPublicKey)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue([]byte(nodeKeyKey), nodeKeyV9Byte)
		if err != nil {
			return "", err
		}
	case strings.HasPrefix(string(key), "Request") && strings.HasSuffix(string(key), "versions"):
		keyType = "Request"
		// Versions of request
		var keyVersionsV8 didProtoV8.KeyVersions
		err := proto.Unmarshal([]byte(value), &keyVersionsV8)
		if err != nil {
			panic(err)
		}
		latestVersion := strconv.FormatInt(keyVersionsV8.Versions[len(keyVersionsV8.Versions)-1], 10)
		keyParts := strings.Split(string(key), "|")
		requestID := keyParts[1]

		// Get last version of request detail
		requestV8Key := "Request" + "|" + requestID + "|" + latestVersion
		requestV8Value, err := dbGet([]byte(requestV8Key))
		if err != nil {
			return "", err
		}

		var requestV8 didProtoV8.Request
		if err := proto.Unmarshal([]byte(requestV8Value), &requestV8); err != nil {
			panic(err)
		}

		// FIXME: make this configurable

		// TODO: also filter by block height from latest?
		// requestV8.CreationBlockHeight
		// if requestV8.Closed || requestV8.TimedOut {
		// 	// Do not save
		// 	keyType = ""
		// 	break
		// }

		// Set to 1 version
		var keyVersionsV9 didProtoV9.KeyVersions = didProtoV9.KeyVersions{
			Versions: append(make([]int64, 0), 1),
		}
		newReqVersionsValue, err := proto.DeterministicMarshal(&keyVersionsV9)
		if err != nil {
			panic(err)
		}
		newReqDetailKey := v9.RequestKeyPrefix + v9.KeySeparator + requestID + v9.KeySeparator + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), requestV8Value)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(key, newReqVersionsValue)
		if err != nil {
			return "", err
		}
		// v8SavedRequestID[requestID] = true
	// case strings.HasPrefix(string(key), "SignData"):
	// 	keySuffix := strings.Split(string(key), keySeparator)
	// 	if v8SavedRequestID[keySuffix[len(keySuffix)-1]] {
	// 		keyType = "SignData"

	// 		err := saveKeyValue(key, value)
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 	}
	// AS response to Request
	// Do not save
	case strings.HasPrefix(string(key), "n") && len(value) == 0:
		// nonce
		// Do not save
	default:
		switch {
		// case strings.HasPrefix(string(key), "NodeID"):
		// 	keyType = "NodeID"
		case strings.HasPrefix(string(key), "RefGroupCode"):
			keyType = "RefGroupCode"
		case strings.HasPrefix(string(key), "SignData"):
			keyType = "SignData"
		case strings.HasPrefix(string(key), "accessorToRefCodeKey"):
			keyType = "accessorToRefCodeKey"
		case strings.HasPrefix(string(key), "identityToRefCodeKey"):
			keyType = "identityToRefCodeKey"
		}

		err := saveKeyValue(key, value)
		if err != nil {
			return "", err
		}
	}

	return keyType, nil
}

func AddNewStateDataToV9(
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (keyType string, err error) {

	//
	log.Println("adding new state data:", "node supported feature:", nodeSupportedFeatureOnTheFly)

	key := v9.NodeSupportedFeatureKeyPrefix + v9.KeySeparator + nodeSupportedFeatureOnTheFly

	var nodeSupportedFeature didProtoV9.NodeSupportedFeature
	value, err := proto.DeterministicMarshal(&nodeSupportedFeature)
	if err != nil {
		return "", err
	}

	err = saveKeyValue([]byte(key), value)
	if err != nil {
		return "", err
	}

	//
	ialList := []float64{1.1, 1.2, 1.3, 2.1, 2.2, 2.3, 3}
	log.Println("adding new state data:", "supported IAL list:", ialList)

	var supportedIALList didProtoV9.SupportedIALList
	supportedIALList.IalList = ialList
	supportedIALListByte, err := proto.DeterministicMarshal(&supportedIALList)
	if err != nil {
		return "", err
	}

	err = saveKeyValue(v9.SupportedIALListKeyBytes, supportedIALListByte)
	if err != nil {
		return "", err
	}

	//
	aalList := []float64{1, 2.1, 2.2, 3}
	log.Println("adding new state data:", "supported AAL list:", aalList)

	var supportedAALList didProtoV9.SupportedAALList
	supportedAALList.AalList = aalList
	supportedAALListByte, err := proto.DeterministicMarshal(&supportedAALList)
	if err != nil {
		return "", err
	}

	err = saveKeyValue(v9.SupportedAALListKeyBytes, supportedAALListByte)
	if err != nil {
		return "", err
	}

	return "", nil
}

func isKnownKeyV8(key string) bool {
	for _, knownKey := range knownKeysV8 {
		if strings.HasPrefix(string(key), knownKey) {
			return true
		}
	}

	return false
}

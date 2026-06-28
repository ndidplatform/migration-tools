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
	"sync"

	"github.com/spf13/viper"

	v10 "github.com/ndidplatform/migration-tools/did/v10"
	didProtoV10 "github.com/ndidplatform/migration-tools/did/v10/protos/data"
	v9 "github.com/ndidplatform/migration-tools/did/v9"
	didProtoV9 "github.com/ndidplatform/migration-tools/did/v9/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

var knownKeysV9 []string = []string{
	"MasterNDID",
	"InitState",
	"lastBlock",
	"IdPList",
	"AllNamespace",
	"ServicePriceMinEffectiveDatetimeDelay",
	"SupportedIALList",
	"SupportedAALList",

	"ChainHistoryInfo",
	"TimeOutBlockRegisterIdentity",
	"AllowedMinIalForRegisterIdentityAtFirstIdp",
	"rpList",
	"asList",
	"allList",
	"AllService",

	"n", // nonce
	"NodeID",
	"NodeKey",
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
	"NodeSupportedFeature",

	"Validator",
}

var (
	byteStateKey             = []byte("stateKey")
	byteLastBlock            = []byte("lastBlock")
	byteMasterNDID           = []byte("MasterNDID")
	byteInitState            = []byte("InitState")
	byteRequest              = []byte("Request")
	byteVersions             = []byte("versions")
	byteRequestType          = []byte("RequestType")
	byteValidator            = []byte("Validator")
	byteChainHistoryInfo     = []byte("ChainHistoryInfo")
	byteAllService           = []byte("AllService")
	byteN                    = []byte("n")
	byteNodeID               = []byte("NodeID")
	byteSignData             = []byte("SignData")
	byteAccessorToRefCodeKey = []byte("accessorToRefCodeKey")
	byteIdentityToRefCodeKey = []byte("identityToRefCodeKey")
	bytePipe                 = []byte(v9.KeySeparator)
	byteServiceSep           = append([]byte("Service"), v9.KeySeparator...)

	byteV10RequestKeyPrefix = []byte(v10.RequestKeyPrefix)
	byteV10KeySeparator     = []byte(v10.KeySeparator)
	byteOne                 = []byte("1")
)

type KV struct {
	Key   []byte
	Value []byte
}

var kvPool = sync.Pool{
	New: func() interface{} {
		return &KV{
			// Pre-allocate a reasonable capacity to avoid small resizes
			Key:   make([]byte, 0, 128),
			Value: make([]byte, 0, 2048),
		}
	},
}

var (
	currentCacheID []byte
	// Maps version string (e.g., "1") to the raw value bytes
	versionCache = make(map[string][]byte)
)

var newReqVersionsValue []byte

func ConvertInputStateDBDataV9ToV10AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v9.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v9StateDB := v9.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v9StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v9StateDB.Get(key)
	}

	// New Request version value (same for all requests)
	var keyVersionsV10 didProtoV10.KeyVersions = didProtoV10.KeyVersions{
		Versions: append(make([]int64, 0), 1),
	}
	newReqVersionsValue, err = proto.DeterministicMarshal(&keyVersionsV10)
	if err != nil {
		return err
	}

	//

	var keyTypeStats map[string]int64 = make(map[string]int64)

	var keysRead int64 = 0

	readBufferSize := viper.GetInt("READ_BUFFER_SIZE")

	log.Printf("read buffer size: %d\n", readBufferSize)

	kvChan := make(chan *KV, readBufferSize)
	errChan := make(chan error, 1)

	go func() {
		defer close(kvChan)

		itr, err := v9StateDB.Iterator(nil, nil)
		if err != nil {
			errChan <- err
			return
		}
		defer itr.Close()
		for ; itr.Valid(); itr.Next() {
			kv := kvPool.Get().(*KV)

			kv.Key = kv.Key[:0]
			kv.Value = kv.Value[:0]

			kv.Key = append(kv.Key, itr.Key()...)
			kv.Value = append(kv.Value, itr.Value()...)

			kvChan <- kv

			// key := make([]byte, len(itr.Key()))
			// copy(key, itr.Key())
			// value := make([]byte, len(itr.Value()))
			// copy(value, itr.Value())

			// kvChan <- KV{Key: key, Value: value}

			keysRead++
		}
	}()

	// saveKeyValueWithCopy := func(key []byte, value []byte) (err error) {
	// 	stableKey := make([]byte, len(key))
	// 	copy(stableKey, key)

	// 	stableValue := make([]byte, len(value))
	// 	copy(stableValue, value)

	// 	return saveKeyValue(stableKey, stableValue)
	// }

	for kv := range kvChan {
		keyPrefix, err := ConvertStateDBDataV9ToV10(
			kv.Key,
			kv.Value,
			ndidNodeID,
			currentChainData,
			dbGet,
			saveNewChainHistory,
			saveKeyValue,
		)
		if err != nil {
			return err
		}

		kvPool.Put(kv)

		if keyPrefix != "" {
			keyTypeStats[keyPrefix]++
		}
	}

	select {
	case err := <-errChan:
		return err
	default:
	}

	// itr, err := v9StateDB.Iterator(nil, nil)
	// if err != nil {
	// 	return err
	// }
	// defer itr.Close()
	// for ; itr.Valid(); itr.Next() {
	// 	key := itr.Key()
	// 	value := itr.Value()

	// 	keyPrefix, err := ConvertStateDBDataV9ToV10(
	// 		key,
	// 		value,
	// 		string(ndidNodeID),
	// 		currentChainData,
	// 		dbGet,
	// 		saveNewChainHistory,
	// 		saveKeyValue,
	// 	)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	keysRead++
	// 	if keyPrefix != "" {
	// 		keyTypeStats[keyPrefix]++
	// 	}
	// }

	log.Println("total key read:", keysRead)
	log.Println("key type stats:", keyTypeStats)

	//
	// data with new keys
	//

	log.Println("adding new state data")

	_, err = AddNewStateDataToV10(
		dbGet,
		saveNewChainHistory,
		saveKeyValue,
	)
	if err != nil {
		return err
	}

	return nil
}

func ConvertStateDBDataV9ToV10(
	key []byte,
	value []byte,
	ndidNodeID []byte,
	currentChainData *v9.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (keyType string, err error) {
	// Delete prefix
	if bytes.Contains(key, v9.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v9.KvPairPrefixKey)
	}
	switch {
	case bytes.HasPrefix(key, byteStateKey):
		// ABCI state metadata
		// Do not save
	case bytes.HasPrefix(key, byteLastBlock):
		// Last block
		// Do not save
	case len(ndidNodeID) > 0 && !bytes.HasPrefix(key, byteMasterNDID) && bytes.Contains(key, ndidNodeID):
		// NDID node detail
		// Do not save
	case bytes.HasPrefix(key, byteMasterNDID):
		// NDID
		// Do not save
	case bytes.HasPrefix(key, byteInitState):
		// Init state
		// Do not save
	case bytes.HasPrefix(key, byteRequest) && !bytes.HasSuffix(key, byteVersions) &&
		!bytes.HasPrefix(key, byteRequestType):
		// Request detail
		// Do not save

		// cache for use on (latest version) Request copy in case below
		keyParts := bytes.Split(key, bytePipe)
		if len(keyParts) >= 3 {
			reqID := keyParts[1]
			versionStr := string(keyParts[2])

			// When moved to a new Request ID, clear the cache to free memory immediately
			if !bytes.Equal(currentCacheID, reqID) {
				// currentCacheID = append(currentCacheID[:0], reqID...) // Reuse underlying memory buffer
				currentCacheID = make([]byte, len(reqID))
				copy(currentCacheID, reqID)
				clear(versionCache) // reset map
			}

			valCopy := make([]byte, len(value))
			copy(valCopy, value)
			versionCache[versionStr] = valCopy
		}
	case bytes.HasPrefix(key, byteValidator):
		// Validator
		// Do not save

		// err = saveKeyValue(bytes.Clone(key), bytes.Clone(value))
		// if err != nil {
		// 	return err
		// }
	case bytes.HasPrefix(key, byteChainHistoryInfo):
		var chainHistory v9.ChainHistory
		if len(value) > 0 {
			err := json.Unmarshal(value, &chainHistory)
			if err != nil {
				return "", err
			}
		}

		// TODO: refactor same chain history structure or separate by version (actually convert from one to another)?

		if currentChainData != nil {
			chainHistory.Chains = append(chainHistory.Chains, *currentChainData)
		}
		chainHistoryStr, err := json.Marshal(chainHistory)
		if err != nil {
			return "", err
		}
		err = saveNewChainHistory(chainHistoryStr)
		if err != nil {
			return "", err
		}
	case bytes.HasPrefix(key, byteServiceSep):
		keyType = "Service"

		// Changes:
		// - Add "Domain"
		// - Add "RequesterNodeWhitelistEnabled"

		var serviceDetailV9 didProtoV9.ServiceDetail
		err := proto.Unmarshal(value, &serviceDetailV9)
		if err != nil {
			return "", err
		}

		// keyParts := strings.Split(string(key), "|")
		// serviceID := keyParts[1]

		serviceDetailV10 := didProtoV10.ServiceDetail{
			ServiceId:                     serviceDetailV9.ServiceId,
			ServiceName:                   serviceDetailV9.ServiceName,
			DataSchema:                    serviceDetailV9.DataSchema,
			DataSchemaVersion:             serviceDetailV9.DataSchemaVersion,
			Active:                        serviceDetailV9.Active,
			Domain:                        "",
			RequesterNodeWhitelistEnabled: false,
		}

		serviceDetailV10Bytes, err := proto.DeterministicMarshal(&serviceDetailV10)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(bytes.Clone(key), serviceDetailV10Bytes)
		if err != nil {
			return "", err
		}
	case bytes.HasPrefix(key, byteAllService):
		keyType = "AllService"

		// Changes:
		// - Add "Domain"

		var serviceDetailListV9 didProtoV9.ServiceDetailList
		err := proto.Unmarshal(value, &serviceDetailListV9)
		if err != nil {
			return "", err
		}

		serviceDetailListV10 := didProtoV10.ServiceDetailList{
			Services: make([]*didProtoV10.ServiceDetail, len(serviceDetailListV9.Services)),
		}
		for i, service := range serviceDetailListV9.Services {
			serviceDetailListV10.Services[i] = &didProtoV10.ServiceDetail{
				ServiceId:   service.ServiceId,
				ServiceName: service.ServiceName,
				Active:      service.Active,
				Domain:      "",
			}
		}

		serviceDetailListV10Bytes, err := proto.DeterministicMarshal(&serviceDetailListV10)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(bytes.Clone(key), serviceDetailListV10Bytes)
		if err != nil {
			return "", err
		}
	case bytes.HasPrefix(key, byteRequest) && bytes.HasSuffix(key, byteVersions):
		keyType = "Request"
		// Versions of request
		var keyVersionsV9 didProtoV9.KeyVersions
		err := proto.Unmarshal(value, &keyVersionsV9)
		if err != nil {
			return "", err
		}
		latestVersionStr := strconv.FormatInt(keyVersionsV9.Versions[len(keyVersionsV9.Versions)-1], 10)
		keyParts := bytes.Split(key, bytePipe)
		requestID := keyParts[1]

		// Get last version of request detail
		var requestV9Value []byte
		var found bool

		// Check rolling memory cache first
		if bytes.Equal(currentCacheID, requestID) {
			requestV9Value, found = versionCache[latestVersionStr]
		}

		// Fallback if not found in cache
		if !found {
			v9KeyLen := len(byteRequest) + len(bytePipe) + len(requestID) + len(bytePipe) + len(latestVersionStr)
			requestV9Key := make([]byte, 0, v9KeyLen)
			requestV9Key = append(requestV9Key, byteRequest...)
			requestV9Key = append(requestV9Key, bytePipe...)
			requestV9Key = append(requestV9Key, requestID...)
			requestV9Key = append(requestV9Key, bytePipe...)
			requestV9Key = append(requestV9Key, latestVersionStr...)

			requestV9Value, err = dbGet(requestV9Key)
			if err != nil {
				return "", err
			}
		}

		// var requestV9 didProtoV9.Request
		// if err := proto.Unmarshal(requestV9Value, &requestV9); err != nil {
		// 	return "", err
		// }

		// Set to 1 version
		v10KeyLen := len(byteV10RequestKeyPrefix) + len(byteV10KeySeparator) + len(requestID) + len(byteV10KeySeparator) + len(byteOne)
		newReqDetailKey := make([]byte, 0, v10KeyLen)
		newReqDetailKey = append(newReqDetailKey, byteV10RequestKeyPrefix...)
		newReqDetailKey = append(newReqDetailKey, byteV10KeySeparator...)
		newReqDetailKey = append(newReqDetailKey, requestID...)
		newReqDetailKey = append(newReqDetailKey, byteV10KeySeparator...)
		newReqDetailKey = append(newReqDetailKey, byteOne...)
		// Write request detail and Version of request detail
		err = saveKeyValue(newReqDetailKey, requestV9Value)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(bytes.Clone(key), newReqVersionsValue)
		if err != nil {
			return "", err
		}
	case bytes.HasPrefix(key, byteN) && len(value) == 0:
		// nonce
		// Do not save
	default:
		switch {
		case bytes.HasPrefix(key, byteNodeID):
			keyType = "NodeID"
		// case strings.HasPrefix(string(key), "RefGroupCode"):
		// 	keyType = "RefGroupCode"
		case bytes.HasPrefix(key, byteSignData):
			keyType = "SignData"
		case bytes.HasPrefix(key, byteAccessorToRefCodeKey):
			keyType = "accessorToRefCodeKey"
		case bytes.HasPrefix(key, byteIdentityToRefCodeKey):
			keyType = "identityToRefCodeKey"
		}

		err := saveKeyValue(bytes.Clone(key), bytes.Clone(value))
		if err != nil {
			return "", err
		}
	}

	return keyType, nil
}

func AddNewStateDataToV10(
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (keyType string, err error) {

	return "", nil
}

func isKnownKeyV9(key string) bool {
	for _, knownKey := range knownKeysV9 {
		if strings.HasPrefix(string(key), knownKey) {
			return true
		}
	}

	return false
}

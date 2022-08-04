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

	v7 "github.com/ndidplatform/migration-tools/did/v7"
	didProtoV7 "github.com/ndidplatform/migration-tools/did/v7/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

var knownKeysV7 []string = []string{
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

	"val:",
}

func ReadInputStateDBDataV7AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v7.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v7StateDB := v7.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v7StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v7StateDB.Get(key)
	}

	var keyTypeStats map[string]int64 = make(map[string]int64)

	var keysRead int64 = 0

	itr, err := v7StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		keyPrefix, err := ProcessStateDBDataV7(
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

	return nil
}

func ProcessStateDBDataV7(
	key []byte,
	value []byte,
	ndidNodeID string,
	currentChainData *v7.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (keyType string, err error) {
	// Delete prefix
	if bytes.Contains(key, v7.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v7.KvPairPrefixKey)
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
	case strings.HasPrefix(string(key), "IdentityProof"):
		// Identity proof
		// Do not save
	case strings.HasPrefix(string(key), "Accessor"):
		// All key that have associate with Accessor
		// Do not save
	case strings.HasPrefix(string(key), "Request") && !strings.HasSuffix(string(key), "versions"):
		// Request detail
		// Do not save
	case strings.HasPrefix(string(key), "val:"):
		// Validator
		// Do not save

		// err = saveKeyValue(key, value)
		// if err != nil {
		// 	return err
		// }
	case strings.HasPrefix(string(key), "ChainHistoryInfo"):
		var chainHistory v7.ChainHistory
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
	case strings.HasPrefix(string(key), "Request") && strings.HasSuffix(string(key), "versions"):
		keyType = "Request"
		// Versions of request
		var keyVersionsV7 didProtoV7.KeyVersions
		err := proto.Unmarshal([]byte(value), &keyVersionsV7)
		if err != nil {
			panic(err)
		}
		latestVersion := strconv.FormatInt(keyVersionsV7.Versions[len(keyVersionsV7.Versions)-1], 10)
		keyParts := strings.Split(string(key), "|")
		requestID := keyParts[1]

		// Get last version of request detail
		requestV7Key := "Request" + "|" + requestID + "|" + latestVersion
		requestV7Value, err := dbGet([]byte(requestV7Key))
		if err != nil {
			return "", err
		}

		// Set to 1 version
		var newKeyVersionsV7 didProtoV7.KeyVersions = didProtoV7.KeyVersions{
			Versions: append(make([]int64, 0), 1),
		}
		newReqVersionsValue, err := proto.DeterministicMarshal(&newKeyVersionsV7)
		if err != nil {
			panic(err)
		}
		newReqDetailKey := "Request" + "|" + requestID + "|" + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), requestV7Value)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(key, newReqVersionsValue)
		if err != nil {
			return "", err
		}
	case len(value) == 0 && !isKnownKeyV7(string(key)):
		// nonce
		// Do not save
	default:
		switch {
		case strings.HasPrefix(string(key), "NodeID"):
			keyType = "NodeID"
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

func isKnownKeyV7(key string) bool {
	for _, knownKey := range knownKeysV7 {
		if strings.HasPrefix(string(key), knownKey) {
			return true
		}
	}

	return false
}

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

	var keyTypeStats map[string]int64 = make(map[string]int64)

	var keysRead int64 = 0

	itr, err := v9StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	defer itr.Close()
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		keyPrefix, err := ConvertStateDBDataV9ToV10(
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
	ndidNodeID string,
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
		var chainHistory v9.ChainHistory
		if string(value) != "" {
			err := json.Unmarshal([]byte(value), &chainHistory)
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
	case strings.HasPrefix(string(key), "Service"+v9.KeySeparator):
		keyType = "Service"

		// Changes:
		// - Add "Domain"
		// - Add "RequesterNodeWhitelistEnabled"

		var serviceDetailV9 didProtoV9.ServiceDetail
		err := proto.Unmarshal([]byte(value), &serviceDetailV9)
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
		err = saveKeyValue(key, serviceDetailV10Bytes)
		if err != nil {
			return "", err
		}
	case strings.HasPrefix(string(key), "AllService"):
		keyType = "AllService"

		// Changes:
		// - Add "Domain"

		var serviceDetailListV9 didProtoV9.ServiceDetailList
		err := proto.Unmarshal([]byte(value), &serviceDetailListV9)
		if err != nil {
			return "", err
		}

		serviceDetailListV10 := didProtoV10.ServiceDetailList{
			Services: make([]*didProtoV10.ServiceDetail, len(serviceDetailListV9.Services)),
		}
		for _, service := range serviceDetailListV9.Services {
			serviceDetailListV10.Services = append(serviceDetailListV10.Services, &didProtoV10.ServiceDetail{
				ServiceId:   service.ServiceId,
				ServiceName: service.ServiceName,
				Active:      service.Active,
				Domain:      "",
			})
		}

		serviceDetailListV10Bytes, err := proto.DeterministicMarshal(&serviceDetailListV10)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(key, serviceDetailListV10Bytes)
		if err != nil {
			return "", err
		}
	case strings.HasPrefix(string(key), "Request") && strings.HasSuffix(string(key), "versions"):
		keyType = "Request"
		// Versions of request
		var keyVersionsV9 didProtoV9.KeyVersions
		err := proto.Unmarshal([]byte(value), &keyVersionsV9)
		if err != nil {
			return "", err
		}
		latestVersion := strconv.FormatInt(keyVersionsV9.Versions[len(keyVersionsV9.Versions)-1], 10)
		keyParts := strings.Split(string(key), "|")
		requestID := keyParts[1]

		// Get last version of request detail
		requestV9Key := "Request" + "|" + requestID + "|" + latestVersion
		requestV9Value, err := dbGet([]byte(requestV9Key))
		if err != nil {
			return "", err
		}

		// var requestV9 didProtoV9.Request
		// if err := proto.Unmarshal([]byte(requestV9Value), &requestV9); err != nil {
		// 	return "", err
		// }

		// Set to 1 version
		var keyVersionsV10 didProtoV10.KeyVersions = didProtoV10.KeyVersions{
			Versions: append(make([]int64, 0), 1),
		}
		newReqVersionsValue, err := proto.DeterministicMarshal(&keyVersionsV10)
		if err != nil {
			return "", err
		}
		newReqDetailKey := v10.RequestKeyPrefix + v10.KeySeparator + requestID + v10.KeySeparator + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), requestV9Value)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(key, newReqVersionsValue)
		if err != nil {
			return "", err
		}
	case strings.HasPrefix(string(key), "n") && len(value) == 0:
		// nonce
		// Do not save
	default:
		switch {
		case strings.HasPrefix(string(key), "NodeID"):
			keyType = "NodeID"
		// case strings.HasPrefix(string(key), "RefGroupCode"):
		// 	keyType = "RefGroupCode"
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

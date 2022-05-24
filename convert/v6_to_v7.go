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

	v6 "github.com/ndidplatform/migration-tools/did/v6"
	didProtoV6 "github.com/ndidplatform/migration-tools/did/v6/protos/data"
	didProtoV7 "github.com/ndidplatform/migration-tools/did/v7/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

var knownKeys []string = []string{
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

func ConvertInputStateDBDataV6ToV7AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v6.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v6StateDB := v6.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v6StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v6StateDB.Get(key)
	}

	var keyPrefixStats map[string]int64 = make(map[string]int64)

	var keysRead int64 = 0

	itr, err := v6StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		keyPrefix, err := ConvertStateDBDataV6ToV7(
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
		keyPrefixStats[keyPrefix]++
	}

	log.Println("total key read:", keysRead)
	log.Println("key prefix stats:", keyPrefixStats)

	return nil
}

func ConvertStateDBDataV6ToV7(
	key []byte,
	value []byte,
	ndidNodeID string,
	currentChainData *v6.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (keyPrefix string, err error) {
	// Delete prefix
	if bytes.Contains(key, v6.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v6.KvPairPrefixKey)
	}
	switch {
	case strings.Contains(string(key), "lastBlock"):
		// Last block
		// Do not save
	case ndidNodeID != "" && strings.Contains(string(key), string(ndidNodeID)) && !strings.Contains(string(key), "MasterNDID"):
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
		// Do not save

		// err = saveKeyValue(key, value)
		// if err != nil {
		// 	return err
		// }
	case strings.Contains(string(key), "ChainHistoryInfo"):
		var chainHistory v6.ChainHistory
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
	case strings.Contains(string(key), "Request") && strings.Contains(string(key), "versions"):
		// Versions of request
		var keyVersionsV6 didProtoV6.KeyVersions
		err := proto.Unmarshal([]byte(value), &keyVersionsV6)
		if err != nil {
			panic(err)
		}
		latestVersion := strconv.FormatInt(keyVersionsV6.Versions[len(keyVersionsV6.Versions)-1], 10)
		keyParts := strings.Split(string(key), "|")
		requestID := keyParts[1]

		// Get last version of request detail
		requestV6Key := "Request" + "|" + requestID + "|" + latestVersion
		requestV6Value, err := dbGet([]byte(requestV6Key))
		if err != nil {
			return "", err
		}

		var requestV6 didProtoV6.Request
		if err := proto.Unmarshal([]byte(requestV6Value), &requestV6); err != nil {
			panic(err)
		}

		// data request, AS responses
		var dataRequestListV7 []*didProtoV7.DataRequest = make([]*didProtoV7.DataRequest, 0)
		for _, dataRequestV6 := range requestV6.DataRequestList {
			responseListV7 := make([]*didProtoV7.ASResponse, 0)
			for _, response := range dataRequestV6.ResponseList {
				responseListV7 = append(responseListV7, &didProtoV7.ASResponse{
					AsId:         response.AsId,
					Signed:       response.Signed,
					ReceivedData: response.ReceivedData,
					ErrorCode:    response.ErrorCode,
				})
			}

			dataRequestV7 := &didProtoV7.DataRequest{
				ServiceId:         dataRequestV6.ServiceId,
				AsIdList:          dataRequestV6.AsIdList,
				MinAs:             dataRequestV6.MinAs,
				RequestParamsHash: dataRequestV6.RequestParamsHash,
				ResponseList:      responseListV7,
			}
			dataRequestListV7 = append(dataRequestListV7, dataRequestV7)
		}

		// IdP responses
		var responseListV7 []*didProtoV7.Response = make([]*didProtoV7.Response, 0)
		for _, responseV6 := range requestV6.ResponseList {
			responseV7 := &didProtoV7.Response{
				Ial:            responseV6.Ial,
				Aal:            responseV6.Aal,
				Status:         responseV6.Status,
				Signature:      responseV6.Signature,
				IdpId:          responseV6.IdpId,
				ValidIal:       responseV6.ValidIal,
				ValidSignature: responseV6.ValidSignature,
				ErrorCode:      responseV6.ErrorCode,
			}
			responseListV7 = append(responseListV7, responseV7)
		}

		var requestV7 didProtoV7.Request = didProtoV7.Request{
			RequestId:           requestV6.RequestId,
			MinIdp:              requestV6.MinIdp,
			MinAal:              requestV6.MinAal,
			MinIal:              requestV6.MinIal,
			RequestTimeout:      requestV6.RequestTimeout,
			IdpIdList:           requestV6.IdpIdList,
			DataRequestList:     dataRequestListV7,
			RequestMessageHash:  requestV6.RequestMessageHash,
			ResponseList:        responseListV7,
			Closed:              requestV6.Closed,
			TimedOut:            requestV6.TimedOut,
			Purpose:             requestV6.Purpose,
			Owner:               requestV6.Owner,
			Mode:                requestV6.Mode,
			UseCount:            requestV6.UseCount,
			CreationBlockHeight: requestV6.CreationBlockHeight,
			ChainId:             requestV6.ChainId,
			RequestType:         "", // new field
		}

		requestV7Bytes, err := proto.DeterministicMarshal(&requestV7)
		if err != nil {
			panic(err)
		}

		// Set to 1 version
		var keyVersionsV7 didProtoV7.KeyVersions = didProtoV7.KeyVersions{
			Versions: append(make([]int64, 0), 1),
		}
		newReqVersionsValue, err := proto.DeterministicMarshal(&keyVersionsV7)
		if err != nil {
			panic(err)
		}
		newReqDetailKey := "Request" + "|" + requestID + "|" + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), requestV7Bytes)
		if err != nil {
			return "", err
		}
		err = saveKeyValue(key, newReqVersionsValue)
		if err != nil {
			return "", err
		}
	case len(value) == 0 && !isKnownKey(string(key)):
		// nonce
		// Do not save
	default:
		err := saveKeyValue(key, value)
		if err != nil {
			return "", err
		}
	}

	for _, knownKey := range knownKeys {
		if strings.HasPrefix(string(key), knownKey) {
			keyPrefix = knownKey
		}
	}

	return keyPrefix, nil
}

func isKnownKey(key string) bool {
	for _, knownKey := range knownKeys {
		if strings.HasPrefix(string(key), knownKey) {
			return true
		}
	}

	return false
}

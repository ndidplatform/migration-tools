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

	v2 "github.com/ndidplatform/migration-tools/did/v2"
	didProtoV2 "github.com/ndidplatform/migration-tools/did/v2/protos/data"
	didProtoV3 "github.com/ndidplatform/migration-tools/did/v3/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

func ConvertInputStateDBDataV2ToV3AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v2.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v2StateDB := v2.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v2StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v2StateDB.Get(key)
	}

	var keysRead int64 = 0

	itr, err := v2StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		err := ConvertStateDBDataV2ToV3(
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
	}

	log.Println("total key read:", keysRead)

	return nil
}

func ConvertStateDBDataV2ToV3(
	key []byte,
	value []byte,
	ndidNodeID string,
	currentChainData *v2.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	// Delete prefix
	if bytes.Contains(key, v2.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v2.KvPairPrefixKey)
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
	case strings.Contains(string(key), "ProvideService") || strings.Contains(string(key), "ServiceDestination"):
		// AS need to RegisterServiceDestination after migrate chain completed
		// Do not save
	case strings.Contains(string(key), "Accessor"):
		// All key that have associate with Accessor
		// Do not save
	case strings.Contains(string(key), "Request") && !strings.Contains(string(key), "versions"):
		// Request detail
		// Do not save
	case strings.Contains(string(key), "val:"):
		// Validator
		err := saveKeyValue(key, value)
		if err != nil {
			return err
		}
	case strings.Contains(string(key), "ChainHistoryInfo"):
		var chainHistory v2.ChainHistory
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
			return err
		}
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
		newValue, err := proto.DeterministicMarshal(&nodeDetailV3)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue(key, newValue)
		if err != nil {
			return err
		}
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
		reqDetailValue, err := dbGet([]byte(reqDetailKey))
		if err != nil {
			return err
		}

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
		newReqDetailValue, err := proto.DeterministicMarshal(&requestV3)
		if err != nil {
			panic(err)
		}
		// Set to 1 version
		keyVersions.Versions = append(make([]int64, 0), 1)
		newReqVersionsValue, err := proto.DeterministicMarshal(&keyVersions)
		if err != nil {
			panic(err)
		}
		newReqDetailKey := "Request" + "|" + reqID + "|" + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), newReqDetailValue)
		if err != nil {
			return err
		}
		err = saveKeyValue(key, newReqVersionsValue)
		if err != nil {
			return err
		}
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
		newValue, err := proto.DeterministicMarshal(&namespaceV3)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue(key, newValue)
		if err != nil {
			return err
		}
	default:
		err := saveKeyValue(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

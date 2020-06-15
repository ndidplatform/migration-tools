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

	v4 "github.com/ndidplatform/migration-tools/did/v4"
	didProtoV4 "github.com/ndidplatform/migration-tools/did/v4/protos/data"
	didProtoV5 "github.com/ndidplatform/migration-tools/did/v5/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

func ConvertInputStateDBDataV4ToV5AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v4.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v4StateDB := v4.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v4StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v4StateDB.Get(key)
	}

	var keysRead int64 = 0

	itr, err := v4StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		err := ConvertStateDBDataV4ToV5(
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

func ConvertStateDBDataV4ToV5(
	key []byte,
	value []byte,
	ndidNodeID string,
	currentChainData *v4.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	// Delete prefix
	if bytes.Contains(key, v4.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v4.KvPairPrefixKey)
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
		err = saveKeyValue(key, value)
		if err != nil {
			return err
		}
	case strings.Contains(string(key), "ChainHistoryInfo"):
		var chainHistory v4.ChainHistory
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
		requestV4Value, err := dbGet([]byte(requestV4Key))
		if err != nil {
			return err
		}

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

		requestV5Bytes, err := proto.DeterministicMarshal(&requestV5)
		if err != nil {
			panic(err)
		}

		// Set to 1 version
		var keyVersionsV5 didProtoV5.KeyVersions = didProtoV5.KeyVersions{
			Versions: append(make([]int64, 0), 1),
		}
		newReqVersionsValue, err := proto.DeterministicMarshal(&keyVersionsV5)
		if err != nil {
			panic(err)
		}
		newReqDetailKey := "Request" + "|" + requestID + "|" + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), requestV5Bytes)
		if err != nil {
			return err
		}
		err = saveKeyValue(key, newReqVersionsValue)
		if err != nil {
			return err
		}
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

		nodeDetailV5Byte, err := proto.DeterministicMarshal(&nodeDetailV5)
		if err != nil {
			panic(err)
		}
		err = saveKeyValue(key, nodeDetailV5Byte)
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

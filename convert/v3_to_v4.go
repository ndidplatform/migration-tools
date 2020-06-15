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

	v3 "github.com/ndidplatform/migration-tools/did/v3"
	didProtoV4 "github.com/ndidplatform/migration-tools/did/v4/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

func ConvertInputStateDBDataV3ToV4AndBackup(
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	tmHome := viper.GetString("TM_HOME")
	currentChainData, err := v3.GetLastestTendermintData(tmHome)
	if err != nil {
		return err
	}

	dbType := viper.GetString("ABCI_DB_TYPE")
	dbDir := viper.GetString("ABCI_DB_DIR_PATH")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	v3StateDB := v3.GetStateDB(dbType, dbDir)
	ndidNodeID, err := v3StateDB.Get([]byte("MasterNDID"))
	if err != nil {
		return err
	}

	dbGet := func(key []byte) (value []byte, err error) {
		return v3StateDB.Get(key)
	}

	var keysRead int64 = 0

	itr, err := v3StateDB.Iterator(nil, nil)
	if err != nil {
		return err
	}
	for ; itr.Valid(); itr.Next() {
		key := itr.Key()
		value := itr.Value()

		err := ConvertStateDBDataV3ToV4(
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

func ConvertStateDBDataV3ToV4(
	key []byte,
	value []byte,
	ndidNodeID string,
	currentChainData *v3.ChainHistoryDetail,
	dbGet func(key []byte) (value []byte, err error),
	saveNewChainHistory func(chainHistory []byte) (err error),
	saveKeyValue func(key []byte, value []byte) (err error),
) (err error) {
	// Delete prefix
	if bytes.Contains(key, v3.KvPairPrefixKey) {
		key = bytes.TrimPrefix(key, v3.KvPairPrefixKey)
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
		err := saveKeyValue(key, value)
		if err != nil {
			return err
		}
	case strings.Contains(string(key), "ChainHistoryInfo"):
		var chainHistory v3.ChainHistory
		if string(value) != "" {
			err := json.Unmarshal([]byte(value), &chainHistory)
			if err != nil {
				return err
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
		var keyVersions didProtoV4.KeyVersions
		err := proto.Unmarshal([]byte(value), &keyVersions)
		if err != nil {
			return err
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

		// Set version to 1
		keyVersions.Versions = append(make([]int64, 0), 1)
		newReqVersionsValue, err := proto.DeterministicMarshal(&keyVersions)
		if err != nil {
			panic(err)
		}
		newReqDetailKey := "Request" + "|" + reqID + "|" + "1"
		// Write request detail and Version of request detail
		err = saveKeyValue([]byte(newReqDetailKey), reqDetailValue)
		if err != nil {
			return err
		}
		err = saveKeyValue(key, newReqVersionsValue)
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

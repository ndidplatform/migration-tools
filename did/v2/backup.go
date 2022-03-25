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

package v2

import (
	"encoding/hex"
	"strconv"
	"strings"

	dbm "github.com/tendermint/tm-db"

	tendermint_0_30_2 "github.com/ndidplatform/migration-tools/tendermint/0_30_2"
)

type tomlConfig struct {
	DBBackend string `toml:"db_backend"`
	DBPath    string `toml:"db_dir"`
}

type ChainHistoryDetail struct {
	ChainID           string `json:"chain_id"`
	LatestBlockHash   string `json:"latest_block_hash"`
	LatestAppHash     string `json:"latest_app_hash"`
	LatestBlockHeight string `json:"latest_block_height"`
}

type ChainHistory struct {
	Chains []ChainHistoryDetail `json:"chains"`
}

func GetLastestTendermintData(tmHome string) (chainData *ChainHistoryDetail, err error) {
	tendermintStateInfo, err := tendermint_0_30_2.GetTendermintInfo(tmHome)
	if err != nil {
		return nil, err
	}

	chainData = new(ChainHistoryDetail)
	chainData.ChainID = tendermintStateInfo.ChainID
	chainData.LatestBlockHeight = strconv.FormatInt(tendermintStateInfo.LatestBlockHeight, 10)
	chainData.LatestBlockHash = strings.ToUpper(hex.EncodeToString(tendermintStateInfo.LatestBlockHash))
	chainData.LatestAppHash = strings.ToUpper(hex.EncodeToString(tendermintStateInfo.LatestAppHash))

	return chainData, nil
}

var (
	KvPairPrefixKey = []byte("kvPairKey:")
)

type StateDB dbm.DB

func GetStateDB(dbType string, dbDir string) StateDB {
	dbName := "didDB"
	db, err := dbm.NewDB(dbName, dbm.BackendType(dbType), dbDir)
	if err != nil {
		panic(err)
	}
	return db
}

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

package cmd

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/ndidplatform/migration-tools/convert"
	"github.com/ndidplatform/migration-tools/rand"
	"github.com/ndidplatform/migration-tools/utils"
)

const tmpDirectoryName = "ndid_migrate"

var stateDBDataVersions []string = []string{"1", "2", "3", "4", "5", "6", "7"}

var logKeysWritten = false
var logKeysWrittenEvery int64 = 100000

type BackupKeyValue struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

func convertAndBackupStateDBData(fromVersion string, toVersion string) (err error) {
	startTime := time.Now()

	startTimeStr := startTime.Format("20060102_150405")
	randomStr := rand.Str(7)

	instanceDirName := startTimeStr + "_" + randomStr

	defer cleanup(instanceDirName)

	var stateDBDataFromVersionIndex int = -1
	var stateDBDataToVersionIndex int = -1
	for index, stateDBDataVersion := range stateDBDataVersions {
		if stateDBDataVersion == fromVersion {
			stateDBDataFromVersionIndex = index
		}
		if stateDBDataVersion == toVersion {
			stateDBDataToVersionIndex = index
		}
	}
	if stateDBDataFromVersionIndex < 0 {
		return errors.New("unknown fromVersion or not supported")
	}
	if stateDBDataToVersionIndex < 0 {
		return errors.New("unknown toVersion or not supported")
	}

	if stateDBDataToVersionIndex < stateDBDataFromVersionIndex {
		return errors.New("migrate to older versions is not supported")
	}

	logKeysWritten = viper.GetBool("LOG_KEYS_WRITTEN")
	logKeysWrittenEvery = viper.GetInt64("LOG_KEYS_WRITTEN_EVERY")

	backupDataDirectoryPath := viper.GetString("BACKUP_DATA_DIR")
	backupDataDirectoryPath = path.Join(backupDataDirectoryPath, instanceDirName)
	utils.CreateDirIfNotExist(backupDataDirectoryPath)

	backupDataFilename := viper.GetString("BACKUP_DATA_FILENAME")
	// backupValidatorsFilename := viper.GetString("BACKUP_VALIDATORS_FILENAME")
	backupChainHistoryFilename := viper.GetString("CHAIN_HISTORY_FILENAME")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")

	if fromVersion == toVersion {
		// TODO: migrate from/to same version, no data structure conversion
		return errors.New("not supported yet")
	}

	for i := stateDBDataFromVersionIndex; i < stateDBDataToVersionIndex; i++ {
		err = loopConvert(
			i,
			stateDBDataFromVersionIndex,
			stateDBDataToVersionIndex,
			instanceDirName,
			backupDataDirectoryPath,
			backupChainHistoryFilename,
			backupDataFilename,
		)
		if err != nil {
			return err
		}
	}

	backupDataDirectoryAbsolutePath, err := filepath.Abs(backupDataDirectoryPath)
	if err != nil {
		log.Println("backup directory:", backupDataDirectoryPath)
	} else {
		log.Println("backup directory:", backupDataDirectoryAbsolutePath)
	}
	log.Println("convert and backup done")
	log.Println("time used:", time.Since(startTime))

	return nil
}

func loopConvert(
	i int,
	stateDBDataFromVersionIndex int,
	stateDBDataToVersionIndex int,
	instanceDirName string,
	backupDataDirectoryPath string,
	backupChainHistoryFilename string,
	backupDataFilename string,
) (err error) {
	log.Println("converting version:", stateDBDataVersions[i], "to version:", stateDBDataVersions[i+1])

	var tempInputDb *leveldb.DB
	var dbGet func(key []byte) (value []byte, err error)
	if i != stateDBDataFromVersionIndex {
		log.Println("read from temp DB")

		tempInputDb, err = leveldb.OpenFile(path.Join(os.TempDir(), tmpDirectoryName, instanceDirName, "db_version_"+stateDBDataVersions[i]), nil) // TODO: random string prefix on each run (prevent overwrite)
		if err != nil {
			return err
		}
		defer tempInputDb.Close()

		dbGet = func(key []byte) (value []byte, err error) {
			return tempInputDb.Get(key, nil)
		}
	} else {
		log.Println("read from input DB")
	}

	var backupKeyCount int64 = 0

	var saveNewChainHistory func(chainHistory []byte) (err error)
	var saveKeyValue func(key []byte, value []byte) (err error)

	if stateDBDataToVersionIndex-i == 1 {
		// Write to file
		log.Println("write to file")

		saveNewChainHistory = func(chainHistory []byte) (err error) {
			err = utils.AppendLineToFile(
				path.Join(backupDataDirectoryPath, backupChainHistoryFilename),
				chainHistory,
			)
			if err != nil {
				return err
			}
			backupKeyCount++
			if logKeysWritten && backupKeyCount%logKeysWrittenEvery == 0 {
				log.Println("keys written:", backupKeyCount)
			}
			return nil
		}
		saveKeyValue = func(key, value []byte) (err error) {
			var kv BackupKeyValue
			kv.Key = key
			kv.Value = value
			jsonStr, err := json.Marshal(kv)
			if err != nil {
				return err
			}
			err = utils.AppendLineToFile(
				path.Join(backupDataDirectoryPath, backupDataFilename),
				jsonStr,
			)
			if err != nil {
				return err
			}
			backupKeyCount++
			if logKeysWritten && backupKeyCount%logKeysWrittenEvery == 0 {
				log.Println("keys written:", backupKeyCount)
			}
			return nil
		}
	} else {
		// Write to Temp DB
		log.Println("write to temp DB")

		tempOutputDb, err := leveldb.OpenFile(path.Join(os.TempDir(), tmpDirectoryName, instanceDirName, "db_version_"+stateDBDataVersions[i+1]), nil) // TODO: random string prefix on each run (prevent overwrite)
		if err != nil {
			return err
		}
		defer tempOutputDb.Close()

		saveNewChainHistory = func(chainHistory []byte) (err error) {
			err = tempOutputDb.Put(
				[]byte("ChainHistoryInfo"), // TODO: get key from version module instead (support different version's key name)
				chainHistory,
				&opt.WriteOptions{
					// Sync: true,
				},
			)
			if err != nil {
				return err
			}
			backupKeyCount++
			if logKeysWritten && backupKeyCount%logKeysWrittenEvery == 0 {
				log.Println("keys written:", backupKeyCount)
			}
			return nil
		}
		saveKeyValue = func(key, value []byte) (err error) {
			err = tempOutputDb.Put(
				key,
				value,
				&opt.WriteOptions{
					// Sync: true,
				},
			)
			if err != nil {
				return err
			}
			backupKeyCount++
			if logKeysWritten && backupKeyCount%logKeysWrittenEvery == 0 {
				log.Println("keys written:", backupKeyCount)
			}
			return nil
		}
	}

	switch stateDBDataVersions[i] {
	case "1":
		// TODO: v1 -> v2
		return errors.New("not supported yet")
	case "2":
		if i == stateDBDataFromVersionIndex {
			err = convert.ConvertInputStateDBDataV2ToV3AndBackup(saveNewChainHistory, saveKeyValue)
		} else {
			iter := tempInputDb.NewIterator(nil, nil)
			for iter.Next() {
				key := iter.Key()
				value := iter.Value()

				err = convert.ConvertStateDBDataV2ToV3(
					key,
					value,
					"",
					nil,
					dbGet,
					saveNewChainHistory,
					saveKeyValue,
				)
				if err != nil {
					return err
				}
			}
			iter.Release()
			err = iter.Error()
			if err != nil {
				return err
			}
		}
	case "3":
		if i == stateDBDataFromVersionIndex {
			err = convert.ConvertInputStateDBDataV3ToV4AndBackup(saveNewChainHistory, saveKeyValue)
		} else {
			iter := tempInputDb.NewIterator(nil, nil)
			for iter.Next() {
				key := iter.Key()
				value := iter.Value()

				err = convert.ConvertStateDBDataV3ToV4(
					key,
					value,
					"",
					nil,
					dbGet,
					saveNewChainHistory,
					saveKeyValue,
				)
				if err != nil {
					return err
				}
			}
			iter.Release()
			err = iter.Error()
			if err != nil {
				return err
			}
		}
	case "4":
		if i == stateDBDataFromVersionIndex {
			err = convert.ConvertInputStateDBDataV4ToV5AndBackup(saveNewChainHistory, saveKeyValue)
		} else {
			iter := tempInputDb.NewIterator(nil, nil)
			for iter.Next() {
				key := iter.Key()
				value := iter.Value()

				err = convert.ConvertStateDBDataV4ToV5(
					key,
					value,
					"",
					nil,
					dbGet,
					saveNewChainHistory,
					saveKeyValue,
				)
				if err != nil {
					return err
				}
			}
			iter.Release()
			err = iter.Error()
			if err != nil {
				return err
			}
		}
	case "5":
		// v5 -> v6
		return errors.New("not supported")
	case "6":
		if i == stateDBDataFromVersionIndex {
			err = convert.ConvertInputStateDBDataV6ToV7AndBackup(saveNewChainHistory, saveKeyValue)
		} else {
			iter := tempInputDb.NewIterator(nil, nil)
			for iter.Next() {
				key := iter.Key()
				value := iter.Value()

				_, err = convert.ConvertStateDBDataV6ToV7(
					key,
					value,
					"",
					nil,
					dbGet,
					saveNewChainHistory,
					saveKeyValue,
				)
				if err != nil {
					return err
				}
			}
			iter.Release()
			err = iter.Error()
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}

	log.Println("total backup key count:", backupKeyCount)

	return nil
}

func cleanup(instanceDirName string) {
	utils.DeleteDirAndFiles(path.Join(os.TempDir(), tmpDirectoryName, instanceDirName))
}

var convertAndBackupCmd = &cobra.Command{
	Use:   "convert-and-backup [fromVersion] [toVersion]",
	Short: "Convert and backup data for migration",
	Args:  cobra.MinimumNArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		curDir, _ := os.Getwd()
		viper.SetDefault("TM_HOME", path.Join(curDir, "../smart-contract/config/tendermint/IdP"))

		viper.SetDefault("ABCI_DB_TYPE", "goleveldb")
		viper.SetDefault("ABCI_DB_DIR_PATH", path.Join(curDir, "../smart-contract/DB1"))

		viper.SetDefault("LOG_KEYS_WRITTEN", false)
		viper.SetDefault("LOG_KEYS_WRITTEN_EVERY", 100000)
		viper.SetDefault("BACKUP_DATA_DIR", "./_backup_data/")
		viper.SetDefault("BACKUP_DATA_FILENAME", "data")
		viper.SetDefault("BACKUP_VALIDATORS_FILENAME", "validators")
		viper.SetDefault("CHAIN_HISTORY_FILENAME", "chain_history")
		viper.SetDefault("BLOCK_NUMBER", "")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return convertAndBackupStateDBData(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(convertAndBackupCmd)
}

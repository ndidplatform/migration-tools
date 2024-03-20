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

type ABCIDataVersion struct {
	ABCIStateVersion string
	ABCIAppVersions  []string
}

var stateDBDataVersions []ABCIDataVersion = []ABCIDataVersion{
	{"1", []string{"1"}},
	{"2", []string{"2"}},
	{"3", []string{"3"}},
	{"4", []string{"4"}},
	{"5", []string{"5"}},
	{"6", []string{"6"}},
	{"7", []string{"7", "8"}}, // state DB data structure v7 and v8 are the same
	{"9", []string{"9"}},
}

var logKeysWritten = false
var logKeysWrittenEvery int64 = 100000

type KeyValue struct {
	Key   []byte `json:"key"`
	Value []byte `json:"value"`
}

type Metadata struct {
	TotalKeyCount int64 `json:"total_key_count"`
}

func contains(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func createInitialStateData(fromVersion string, toVersion string) (err error) {
	startTime := time.Now()

	startTimeStr := startTime.Format("20060102_150405")
	randomStr := rand.Str(7)

	instanceDirName := startTimeStr + "_" + randomStr

	defer cleanup(instanceDirName)

	var stateDBDataFromVersionIndex int = -1
	var stateDBDataToVersionIndex int = -1
	for index, stateDBDataVersion := range stateDBDataVersions {
		if contains(fromVersion, stateDBDataVersion.ABCIAppVersions) {
			stateDBDataFromVersionIndex = index
		}
		if contains(toVersion, stateDBDataVersion.ABCIAppVersions) {
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

	initialStateDataDirectoryPath := viper.GetString("INITIAL_STATE_DATA_DIR")
	initialStateDataDirectoryPath = path.Join(initialStateDataDirectoryPath, instanceDirName)
	utils.CreateDirIfNotExist(initialStateDataDirectoryPath)

	initialStateDataFilename := viper.GetString("INITIAL_STATE_DATA_FILENAME")
	// backupValidatorsFilename := viper.GetString("BACKUP_VALIDATORS_FILENAME")
	chainHistoryFilename := viper.GetString("CHAIN_HISTORY_FILENAME")
	// backupBlockNumberStr := viper.GetString("BLOCK_NUMBER")
	initialStateMetadataFilename := viper.GetString("METADATA_FILENAME")

	if stateDBDataFromVersionIndex == stateDBDataToVersionIndex {
		err = createInitStateDataSameVersion(
			stateDBDataVersions[stateDBDataFromVersionIndex].ABCIStateVersion,
			instanceDirName,
			initialStateDataDirectoryPath,
			chainHistoryFilename,
			initialStateDataFilename,
			initialStateMetadataFilename,
		)
		if err != nil {
			return err
		}
	} else {
		for i := stateDBDataFromVersionIndex; i < stateDBDataToVersionIndex; i++ {
			err = loopConvert(
				i,
				stateDBDataFromVersionIndex,
				stateDBDataToVersionIndex,
				instanceDirName,
				initialStateDataDirectoryPath,
				chainHistoryFilename,
				initialStateDataFilename,
				initialStateMetadataFilename,
			)
			if err != nil {
				return err
			}
		}
	}

	initialStateDataDirectoryAbsolutePath, err := filepath.Abs(initialStateDataDirectoryPath)
	if err != nil {
		log.Println("initial state directory:", initialStateDataDirectoryPath)
	} else {
		log.Println("initial state directory:", initialStateDataDirectoryAbsolutePath)
	}
	log.Println("create initial state data done")
	log.Println("time used:", time.Since(startTime))

	return nil
}

func createInitStateDataSameVersion(
	stateVersion string,
	instanceDirName string,
	initialStateDataDirectoryPath string,
	chainHistoryFilename string,
	initialStateDataFilename string,
	initialStateMetadataFilename string,
) (err error) {
	log.Println("processing version:", stateVersion)

	log.Println("read from input DB")

	var initialStateKeyCount int64 = 0

	var saveNewChainHistory func(chainHistory []byte) (err error)
	var saveKeyValue func(key []byte, value []byte) (err error)

	// Write to file
	log.Println("write to file")

	saveNewChainHistory = func(chainHistory []byte) (err error) {
		err = utils.AppendLineToFile(
			path.Join(initialStateDataDirectoryPath, chainHistoryFilename),
			chainHistory,
		)
		if err != nil {
			return err
		}
		// initialStateKeyCount++
		// if logKeysWritten && initialStateKeyCount%logKeysWrittenEvery == 0 {
		// 	log.Println("keys written:", initialStateKeyCount)
		// }
		log.Println("chain history written")
		return nil
	}

	initialStateDataFile, err := utils.OpenFileForAppend(path.Join(initialStateDataDirectoryPath, initialStateDataFilename))
	if err != nil {
		return err
	}
	defer initialStateDataFile.Close()

	saveKeyValue = func(key, value []byte) (err error) {
		var kv KeyValue
		kv.Key = key
		kv.Value = value
		jsonStr, err := json.Marshal(kv)
		if err != nil {
			return err
		}
		err = utils.AppendLineToOpenedFile(
			initialStateDataFile,
			jsonStr,
		)
		if err != nil {
			return err
		}
		initialStateKeyCount++
		if logKeysWritten && initialStateKeyCount%logKeysWrittenEvery == 0 {
			log.Println("keys written:", initialStateKeyCount)
		}
		return nil
	}

	switch stateVersion {
	case "7":
		err = convert.ReadInputStateDBDataV7AndBackup(saveNewChainHistory, saveKeyValue)
	default:
		err = errors.New("not supported")
	}
	if err != nil {
		return err
	}

	// write metadata file
	var metadata Metadata
	metadata.TotalKeyCount = initialStateKeyCount
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(initialStateDataDirectoryPath, initialStateMetadataFilename), metadataJson, 0644)
	if err != nil {
		return err
	}

	log.Println("total initial state key count:", initialStateKeyCount)

	return nil
}

func loopConvert(
	i int,
	stateDBDataFromVersionIndex int,
	stateDBDataToVersionIndex int,
	instanceDirName string,
	initialStateDataDirectoryPath string,
	chainHistoryFilename string,
	initialStateDataFilename string,
	initialStateMetadataFilename string,
) (err error) {
	log.Println("converting version:", stateDBDataVersions[i], "to version:", stateDBDataVersions[i+1])

	var tempInputDb *leveldb.DB
	var dbGet func(key []byte) (value []byte, err error)
	if i != stateDBDataFromVersionIndex {
		log.Println("read from temp DB")

		tempInputDb, err = leveldb.OpenFile(path.Join(os.TempDir(), tmpDirectoryName, instanceDirName, "db_version_"+stateDBDataVersions[i].ABCIStateVersion), nil) // TODO: random string prefix on each run (prevent overwrite)
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

	var initialStateKeyCount int64 = 0

	var saveNewChainHistory func(chainHistory []byte) (err error)
	var saveKeyValue func(key []byte, value []byte) (err error)

	if stateDBDataToVersionIndex-i == 1 {
		// Write to file
		log.Println("write to file")

		saveNewChainHistory = func(chainHistory []byte) (err error) {
			err = utils.AppendLineToFile(
				path.Join(initialStateDataDirectoryPath, chainHistoryFilename),
				chainHistory,
			)
			if err != nil {
				return err
			}
			// initialStateKeyCount++
			// if logKeysWritten && initialStateKeyCount%logKeysWrittenEvery == 0 {
			// 	log.Println("keys written:", initialStateKeyCount)
			// }
			log.Println("chain history written")
			return nil
		}

		initialStateDataFile, err := utils.OpenFileForAppend(path.Join(initialStateDataDirectoryPath, initialStateDataFilename))
		if err != nil {
			return err
		}
		defer initialStateDataFile.Close()

		saveKeyValue = func(key, value []byte) (err error) {
			var kv KeyValue
			kv.Key = key
			kv.Value = value
			jsonStr, err := json.Marshal(kv)
			if err != nil {
				return err
			}
			err = utils.AppendLineToOpenedFile(
				initialStateDataFile,
				jsonStr,
			)
			if err != nil {
				return err
			}
			initialStateKeyCount++
			if logKeysWritten && initialStateKeyCount%logKeysWrittenEvery == 0 {
				log.Println("keys written:", initialStateKeyCount)
			}
			return nil
		}
	} else {
		// Write to Temp DB
		log.Println("write to temp DB")

		tempOutputDb, err := leveldb.OpenFile(path.Join(os.TempDir(), tmpDirectoryName, instanceDirName, "db_version_"+stateDBDataVersions[i+1].ABCIStateVersion), nil) // TODO: random string prefix on each run (prevent overwrite)
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
			// initialStateKeyCount++
			// if logKeysWritten && initialStateKeyCount%logKeysWrittenEvery == 0 {
			// 	log.Println("keys written:", initialStateKeyCount)
			// }
			log.Println("chain history written")
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
			initialStateKeyCount++
			if logKeysWritten && initialStateKeyCount%logKeysWrittenEvery == 0 {
				log.Println("keys written:", initialStateKeyCount)
			}
			return nil
		}
	}

	switch stateDBDataVersions[i].ABCIStateVersion {
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
	case "7":
		// v7,v8 -> v9
		if i == stateDBDataFromVersionIndex {
			err = convert.ConvertInputStateDBDataV8ToV9AndBackup(saveNewChainHistory, saveKeyValue)
		} else {
			iter := tempInputDb.NewIterator(nil, nil)
			for iter.Next() {
				key := iter.Key()
				value := iter.Value()

				_, err = convert.ConvertStateDBDataV8ToV9(
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

			_, err = convert.AddNewStateDataToV9(
				dbGet,
				saveNewChainHistory,
				saveKeyValue,
			)
			if err != nil {
				return err
			}
		}
	}
	if err != nil {
		return err
	}

	// write metadata file
	var metadata Metadata
	metadata.TotalKeyCount = initialStateKeyCount
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	err = os.WriteFile(path.Join(initialStateDataDirectoryPath, initialStateMetadataFilename), metadataJson, 0644)
	if err != nil {
		return err
	}

	log.Println("total initial state key count:", initialStateKeyCount)

	return nil
}

func cleanup(instanceDirName string) {
	utils.DeleteDirAndFiles(path.Join(os.TempDir(), tmpDirectoryName, instanceDirName))
}

var createInitialStateDataCmd = &cobra.Command{
	Use:   "create-initial-state-data [fromVersion] [toVersion]",
	Short: "Create initial ABCI state data for InitChain on migration",
	Args:  cobra.MinimumNArgs(2),
	PreRun: func(cmd *cobra.Command, args []string) {
		curDir, _ := os.Getwd()
		viper.SetDefault("TM_HOME", path.Join(curDir, "../smart-contract/config/tendermint/IdP"))

		viper.SetDefault("ABCI_DB_TYPE", "goleveldb")
		viper.SetDefault("ABCI_DB_DIR_PATH", path.Join(curDir, "../smart-contract/DB1"))

		viper.SetDefault("LOG_KEYS_WRITTEN", false)
		viper.SetDefault("LOG_KEYS_WRITTEN_EVERY", 100000)
		viper.SetDefault("INITIAL_STATE_DATA_DIR", "./_initial_state_data/")
		viper.SetDefault("INITIAL_STATE_DATA_FILENAME", "data")
		viper.SetDefault("BACKUP_VALIDATORS_FILENAME", "validators")
		viper.SetDefault("CHAIN_HISTORY_FILENAME", "chain_history")
		viper.SetDefault("METADATA_FILENAME", "metadata")
		viper.SetDefault("BLOCK_NUMBER", "")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return createInitialStateData(args[0], args[1])
	},
}

func init() {
	rootCmd.AddCommand(createInitialStateDataCmd)
}

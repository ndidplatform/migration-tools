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
	"errors"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	v3 "github.com/ndidplatform/migration-tools/did/v3"
	v4 "github.com/ndidplatform/migration-tools/did/v4"
	v5 "github.com/ndidplatform/migration-tools/did/v5"
	v6 "github.com/ndidplatform/migration-tools/did/v6"
	v7 "github.com/ndidplatform/migration-tools/did/v7"
)

func restore(toVersion string) (err error) {
	startTime := time.Now()

	ndidID := viper.GetString("NDID_NODE_ID")
	backupDataDir := viper.GetString("BACKUP_DATA_DIR")
	backupDataFileName := viper.GetString("BACKUP_DATA_FILENAME")
	chainHistoryFileName := viper.GetString("CHAIN_HISTORY_FILENAME")
	keyDir := viper.GetString("KEY_DIR")
	tendermintRPCHost := viper.GetString("TENDERMINT_RPC_HOST")
	tendermintRPCPort := viper.GetString("TENDERMINT_RPC_PORT")
	tendermintRPCAddress := "http://" + tendermintRPCHost + ":" + tendermintRPCPort

	switch toVersion {
	case "3":
		err = v3.Restore(
			ndidID,
			backupDataDir,
			backupDataFileName,
			chainHistoryFileName,
			keyDir,
			tendermintRPCAddress,
		)
	case "4":
		err = v4.Restore(
			ndidID,
			backupDataDir,
			backupDataFileName,
			chainHistoryFileName,
			keyDir,
			tendermintRPCAddress,
		)
	case "5":
		err = v5.Restore(
			ndidID,
			backupDataDir,
			backupDataFileName,
			chainHistoryFileName,
			keyDir,
			tendermintRPCHost,
			tendermintRPCPort,
		)
	case "6":
		err = v6.Restore(
			ndidID,
			backupDataDir,
			backupDataFileName,
			chainHistoryFileName,
			keyDir,
			tendermintRPCHost,
			tendermintRPCPort,
		)
	case "7":
		err = v7.Restore(
			ndidID,
			backupDataDir,
			backupDataFileName,
			chainHistoryFileName,
			keyDir,
			tendermintRPCHost,
			tendermintRPCPort,
		)
	default:
		return errors.New("unsupported Tendermint version")
	}
	if err != nil {
		return err
	}

	log.Println("init and restore done")
	log.Println("time used:", time.Since(startTime))

	return err
}

var restoreCmd = &cobra.Command{
	Use:   "restore [toVersion]",
	Short: "Initialize converted data to new chain",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// curDir, _ := os.Getwd()
		viper.SetDefault("NDID_NODE_ID", "NDID")
		viper.SetDefault("BACKUP_DATA_DIR", "./_backup_data/")
		viper.SetDefault("BACKUP_DATA_FILENAME", "data")
		// viper.SetDefault("BACKUP_VALIDATORS_FILENAME", "validators")
		viper.SetDefault("CHAIN_HISTORY_FILENAME", "chain_history")
		viper.SetDefault("KEY_DIR", "./dev_keys/")
		viper.SetDefault("TENDERMINT_RPC_HOST", "localhost")
		viper.SetDefault("TENDERMINT_RPC_PORT", "45000")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return restore(args[0])
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

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

	v7 "github.com/ndidplatform/migration-tools/did/v7"
)

func initNdid(version string) (err error) {
	startTime := time.Now()

	ndidID := viper.GetString("NDID_NODE_ID")
	keyDir := viper.GetString("KEY_DIR")

	nodeMasterPublicKeyFilepath := viper.GetString("NODE_NEW_MASTER_PUBLIC_KEY_FILEPATH")
	nodePublicKeyFilepath := viper.GetString("NODE_NEW_PUBLIC_KEY_FILEPATH")

	tendermintRPCHost := viper.GetString("TENDERMINT_RPC_HOST")
	tendermintRPCPort := viper.GetString("TENDERMINT_RPC_PORT")

	backupDataDir := viper.GetString("BACKUP_DATA_DIR")
	chainHistoryFileName := viper.GetString("CHAIN_HISTORY_FILENAME")

	switch version {
	case "7":
		err = v7.InitNDID(
			ndidID,
			nodeMasterPublicKeyFilepath,
			nodePublicKeyFilepath,
			keyDir,
			tendermintRPCHost,
			tendermintRPCPort,
			backupDataDir,
			chainHistoryFileName,
		)
	default:
		return errors.New("unsupported ABCI version")
	}
	if err != nil {
		return err
	}

	log.Println("update node done")
	log.Println("time used:", time.Since(startTime))

	return err
}

var initNdidCmd = &cobra.Command{
	Use:   "init-ndid [version]",
	Short: "Init NDID",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// curDir, _ := os.Getwd()
		viper.SetDefault("NDID_NODE_ID", "NDID")
		viper.SetDefault("BACKUP_DATA_DIR", "./_backup_data/")
		viper.SetDefault("CHAIN_HISTORY_FILENAME", "chain_history")
		viper.SetDefault("KEY_DIR", "./dev_keys/")
		viper.SetDefault("TENDERMINT_RPC_HOST", "localhost")
		viper.SetDefault("TENDERMINT_RPC_PORT", "45000")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return initNdid(args[0])
	},
}

func init() {
	rootCmd.AddCommand(initNdidCmd)
}

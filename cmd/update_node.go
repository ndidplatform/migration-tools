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
	v9 "github.com/ndidplatform/migration-tools/did/v9"
)

func updateNode(version string) (err error) {
	startTime := time.Now()

	ndidID := viper.GetString("NDID_NODE_ID")
	keyDir := viper.GetString("KEY_DIR")

	// for v7, v8
	nodeMasterPublicKeyFilepath := viper.GetString("NODE_NEW_MASTER_PUBLIC_KEY_FILEPATH")
	nodePublicKeyFilepath := viper.GetString("NODE_NEW_PUBLIC_KEY_FILEPATH")

	// for v9 or later
	nodeSigningMasterPublicKeyFilepath := viper.GetString("NODE_NEW_SIGNING_MASTER_PUBLIC_KEY_FILEPATH")
	nodeSigningMasterAlgorithm := viper.GetString("NODE_NEW_SIGNING_MASTER_ALGORITHM")
	nodeSigningPublicKeyFilepath := viper.GetString("NODE_NEW_SIGNING_PUBLIC_KEY_FILEPATH")
	nodeSigningAlgorithm := viper.GetString("NODE_NEW_SIGNING_ALGORITHM")
	nodeEncryptionPublicKeyFilepath := viper.GetString("NODE_NEW_ENCRYPTION_PUBLIC_KEY_FILEPATH")
	nodeEncryptionAlgorithm := viper.GetString("NODE_NEW_ENCRYPTION_ALGORITHM")

	tendermintRPCHost := viper.GetString("TENDERMINT_RPC_HOST")
	tendermintRPCPort := viper.GetString("TENDERMINT_RPC_PORT")

	switch version {
	case "7", "8":
		err = v7.SetNodeKeys(
			ndidID,
			nodeMasterPublicKeyFilepath,
			nodePublicKeyFilepath,
			keyDir,
			tendermintRPCHost,
			tendermintRPCPort,
		)
	case "9":
		err = v9.SetNodeKeys(
			ndidID,
			nodeSigningMasterPublicKeyFilepath,
			nodeSigningMasterAlgorithm,
			nodeSigningPublicKeyFilepath,
			nodeSigningAlgorithm,
			nodeEncryptionPublicKeyFilepath,
			nodeEncryptionAlgorithm,
			keyDir,
			tendermintRPCHost,
			tendermintRPCPort,
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

var updateNodeCmd = &cobra.Command{
	Use:   "update-node [version]",
	Short: "Update node (public) keys",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// curDir, _ := os.Getwd()
		viper.SetDefault("NDID_NODE_ID", "NDID")
		viper.SetDefault("KEY_DIR", "./dev_keys/")
		viper.SetDefault("TENDERMINT_RPC_HOST", "localhost")
		viper.SetDefault("TENDERMINT_RPC_PORT", "45000")
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateNode(args[0])
	},
}

func init() {
	rootCmd.AddCommand(updateNodeCmd)
}

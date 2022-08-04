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
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tendermint_0_26_4 "github.com/ndidplatform/migration-tools/tendermint/0_26_4"
	tendermint_0_30_2 "github.com/ndidplatform/migration-tools/tendermint/0_30_2"
	tendermint_0_32_1 "github.com/ndidplatform/migration-tools/tendermint/0_32_1"
	tendermint_0_33_2 "github.com/ndidplatform/migration-tools/tendermint/0_33_2"
	tendermint_0_34_19 "github.com/ndidplatform/migration-tools/tendermint/0_34_19"
)

func loadTendermintInfo(tendermintVersion string) (err error) {
	tmHome := viper.GetString("TM_HOME")

	switch tendermintVersion {
	case "0.26.4":
		_, err = tendermint_0_26_4.GetTendermintInfo(tmHome)
	case "0.30.2":
		_, err = tendermint_0_30_2.GetTendermintInfo(tmHome)
	case "0.32.1":
		_, err = tendermint_0_32_1.GetTendermintInfo(tmHome)
	case "0.33.2":
		_, err = tendermint_0_33_2.GetTendermintInfo(tmHome)
	case "0.34.19":
		_, err = tendermint_0_34_19.GetTendermintInfo(tmHome)
	default:
		return errors.New("unsupported Tendermint version")
	}
	return err
}

var loadTendermintInfoCmd = &cobra.Command{
	Use:   "tendermint_state_info [tendermintVersion]",
	Short: "Print Tendermint state info used for migration",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		curDir, _ := os.Getwd()
		viper.SetDefault("TM_HOME", path.Join(curDir, "../smart-contract/config/tendermint/IdP"))
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return loadTendermintInfo(args[0])
	},
}

func init() {
	rootCmd.AddCommand(loadTendermintInfoCmd)
}

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
	"fmt"

	"github.com/spf13/cobra"

	didProtoV7 "github.com/ndidplatform/migration-tools/did/v7/protos/data"
	"github.com/ndidplatform/migration-tools/proto"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test",
	// Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var requestType didProtoV7.RequestType
		value, err := proto.DeterministicMarshal(&requestType)
		if err != nil {
			return err
		}

		fmt.Println(len(value))
		fmt.Printf("%x\n", value)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}

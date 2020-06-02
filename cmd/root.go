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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "migrate",
	Short: "NDID blockchain migrate",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// var configFile string

func init() {
	cobra.OnInitialize(initConfig)
	// rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is config.yaml)")
}

func initConfig() {
	// if configFile != "" {
	// 	viper.SetConfigFile(configFile)
	// } else {
	// 	viper.SetConfigName("config")
	// 	viper.AddConfigPath(".")
	// 	// viper.AddConfigPath("/etc/ndid-migrate")
	// 	// viper.AddConfigPath("$HOME/.ndid-migrate")
	// }

	viper.SetDefault("LOG_LEVEL", "debug")

	viper.AutomaticEnv()

	// if err := viper.ReadInConfig(); err != nil {
	// 	fmt.Printf("unable to read config: %v\n", err)
	// 	os.Exit(1)
	// }
}

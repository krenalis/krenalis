//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package main

import (
	"log"
	"strings"

	"chichi-cli/chichiapis"
	"chichi-cli/cmd"

	"github.com/spf13/viper"
)

func main() {

	viper.AddConfigPath(".")
	viper.SetConfigName("chichi-cli")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("[error] %s", err)
	}

	// Initialize 'chichiapis'.
	url := viper.GetStringMapString("apis")["url"]
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	workspaceID := viper.GetInt("workspace")
	chichiapis.Init(url, workspaceID)

	cmd.Execute()
}

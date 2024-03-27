//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package main

import (
	"log"
	"strings"

	"github.com/open2b/chichi/chichi-cli/chichiapis"
	"github.com/open2b/chichi/chichi-cli/cmd"

	"github.com/spf13/viper"
)

func main() {

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/chichi-cli")
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

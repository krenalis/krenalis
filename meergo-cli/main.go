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

	"github.com/meergo/meergo/meergo-cli/cmd"
	"github.com/meergo/meergo/meergo-cli/meergoapis"

	"github.com/spf13/viper"
)

func main() {

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/meergo-cli")
	viper.SetConfigName("meergo-cli")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("[error] %s", err)
	}

	// Initialize 'meergoapis'.
	url := viper.GetStringMapString("apis")["url"]
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	workspaceID := viper.GetInt("workspace")
	meergoapis.Init(url, workspaceID)

	cmd.Execute()
}

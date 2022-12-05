//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events [ --source source ] [ --server server ] [ --stream stream ]",
	Short: "stream live events",
	Run: func(cmd *cobra.Command, args []string) {

		flags := flag.NewFlagSet("", flag.ContinueOnError)
		source := flags.Int("source", 0, "mobile or website source connector")
		server := flags.Int("server", 0, "server source connector")
		stream := flags.Int("stream", 0, "stream source connector")

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		listener := chichiapis.AddEventListener(*source, *server, *stream)
		defer chichiapis.RemoveEventListener(listener)

		i := 1

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			events, discarded := chichiapis.Events(listener)
			for _, event := range events {
				fmt.Printf("%d. ", i)
				if event.Header == nil {
					fmt.Print("unknown date")
				} else {
					fmt.Printf("%s", event.Header.ReceivedAt)
				}
				if discarded > 0 {
					percentage := float64(100*len(events)) / float64(len(events)+discarded)
					fmt.Printf(" - Sampled %.0f%%", percentage)
				}
				fmt.Printf("\n%s\n\n", event.Data)
				i++
			}
			time.Sleep(1 * time.Second)
		}
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
}

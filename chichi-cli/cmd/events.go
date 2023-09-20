//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"chichi-cli/chichiapis"

	"github.com/spf13/cobra"
)

var eventsCmd *cobra.Command

func init() {

	var rateS string
	var source, server, stream int

	eventsCmd = &cobra.Command{
		Use:   "events",
		Short: "Stream live events",
		Run: func(cmd *cobra.Command, args []string) {

			ws := workspace(cmd)

			rate, err := time.ParseDuration(rateS)
			if err != nil {
				log.Fatal(errors.New("invalid rate"))
			}
			size := int(1 / float64(rate))
			if size == 0 {
				size = 1
			}

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()
			listener := chichiapis.AddEventListener(ws, size, source, server, stream)
			defer chichiapis.RemoveEventListener(ws, listener)

			i := 1

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				events, discarded := chichiapis.Events(ws, listener)
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
					time.Sleep(rate)
					i++
				}
			}
		},
	}

	flags := eventsCmd.Flags()
	flags.StringVar(&rateS, "rate", "1s", "events rate")
	flags.IntVar(&source, "source", 0, "mobile or website source connector")
	flags.IntVar(&server, "server", 0, "server source connector")
	flags.IntVar(&stream, "stream", 0, "stream source connector")

	rootCmd.AddCommand(eventsCmd)
}

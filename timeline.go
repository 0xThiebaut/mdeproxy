package main

import (
	"encoding/json"
	"github.com/0xThiebaut/mdeproxy/internal/times"
	"github.com/spf13/cobra"
	"time"
)

func timeline() (*cobra.Command, error) {
	now := time.Now().UTC()
	from := now.AddDate(0, -6, 0).Add(time.Minute).Format(times.Layout)
	to := now.Format(times.Layout)
	var machine string

	command := &cobra.Command{
		Use:   "timeline",
		Short: "Export a device's timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := time.Parse(times.Layout, from)
			if err != nil {
				return err
			}
			t, err := time.Parse(times.Layout, to)
			if err != nil {
				return err
			}

			encoder := json.NewEncoder(output)
			events := client.Timeline(cmd.Context(), f, t, machine)
			for {
				select {
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				case event, ok := <-events:
					if !ok {
						return client.Error()
					}
					if err = encoder.Encode(event); err != nil {
						return err
					}
				}
			}
		},
	}

	flags := command.PersistentFlags()
	flags.StringVar(&from, "from", from, "start date")
	flags.StringVar(&to, "to", to, "end date")
	flags.StringVarP(&machine, "machine", "m", machine, "machine identifier")
	return command, command.MarkPersistentFlagRequired("machine")
}

package main

import (
	"context"
	"github.com/0xThiebaut/mdeproxy/lib"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"os/signal"
)

var client lib.Client
var output io.WriteCloser = os.Stdout

func main() {
	var cookie string
	var xsrf string
	var path string

	cmd := cobra.Command{
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if len(path) > 0 {
				if output, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644); err != nil {
					return err
				}
			}

			client, err = lib.New(cookie, xsrf)
			return err
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return output.Close()
		},
	}

	cmd.PersistentFlags().StringVar(&cookie, "cookie", cookie, "cookie for the Defender API proxy")
	if err := cmd.MarkPersistentFlagRequired("cookie"); err != nil {
		log.Fatal(err)
	}

	cmd.PersistentFlags().StringVar(&xsrf, "xsrf", xsrf, "XSRF token for the Defender API proxy")
	if err := cmd.MarkPersistentFlagRequired("xsrf"); err != nil {
		log.Fatal(err)
	}

	cmd.PersistentFlags().StringVar(&path, "output", path, "output file")

	sub, err := timeline()
	if err != nil {
		log.Fatal(err)
	}
	cmd.AddCommand(sub)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		cancel()
	}()

	if err = cmd.ExecuteContext(ctx); err != nil {
		log.Fatal(err)
	}
}

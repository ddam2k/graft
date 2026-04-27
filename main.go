package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/ddam2k/graft/internal/cmd"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:               "graft",
		Short:             "Build a Dockerfile",
		RunE:              func(cmd *cobra.Command, _ []string) error { return cmd.Usage() },
		DisableAutoGenTag: true,
		SilenceUsage:      true,
	}

	root.AddCommand(cmd.NewCmdBuild())
	root.AddCommand(cmd.NewCmdPull())
	root.AddCommand(cmd.NewCmdPush())
	root.AddCommand(cmd.NewCmdDiffPull())
	root.AddCommand(cmd.NewCmdDiffPush())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := root.ExecuteContext(ctx); err != nil {
		root.Usage()
	}
}

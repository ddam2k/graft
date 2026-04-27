package cmd

import (
	"runtime"

	"github.com/ddam2k/graft/internal/diff_push"
	"github.com/spf13/cobra"
)

func NewCmdDiffPush() *cobra.Command {
	imageUrl := ""
	file := ""
	username := ""
	password := ""
	platform := ""
	bInsecure := false

	cmd := &cobra.Command{
		Use:   "diff-push [file] [image]",
		Short: "Push partial layers that are different between base and target",
		Args:  cobra.ExactArgs(2), //cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				file = args[0]
				imageUrl = args[1]
				return diff_push.DiffPush(diff_push.DiffPushConfig{
					ImageUrl: imageUrl,
					Username: username,
					Password: password,
					FilePath: file,
					Platform: platform,
					Insecure: bInsecure,
				})
			} else {
				return cmd.Usage()
			}
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "username for registry")
	cmd.Flags().StringVarP(&password, "password", "p", "", "password for registry")
	cmd.Flags().StringVar(&platform, "platform", "linux/"+runtime.GOARCH, "platform for build")
	cmd.Flags().BoolVar(&bInsecure, "insecure", false, "insecure registry")

	return cmd
}

package cmd

import (
	"runtime"

	"github.com/ddam2k/graft/internal/push"
	"github.com/spf13/cobra"
)

func NewCmdPush() *cobra.Command {
	imageUrl := ""
	username := ""
	password := ""
	platform := ""
	bInsecure := false

	cmd := &cobra.Command{
		Use:   "push [file] [image]",
		Short: "push a Image to registry",
		Args:  cobra.ExactArgs(2), //cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				filePath := args[0]
				imageUrl = args[1]
				return push.Push(push.PushConfig{
					ImageUrl: imageUrl,
					Username: username,
					Password: password,
					FilePath: filePath,
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

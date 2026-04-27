package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"mgkim.hpy.kr/graft/internal/pull"
)

func NewCmdPull() *cobra.Command {
	imageUrl := ""
	file := ""
	username := ""
	password := ""
	platform := ""
	bInsecure := false

	cmd := &cobra.Command{
		Use:   "pull [image]",
		Short: "Pull a Image and export to file",
		Args:  cobra.MinimumNArgs(1), //cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				imageUrl = args[0]
				return pull.Pull(pull.PullConfig{
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

	cmd.Flags().StringVarP(&file, "file", "f", "", "filename to export")
	cmd.Flags().StringVarP(&username, "username", "u", "", "username for registry")
	cmd.Flags().StringVarP(&password, "password", "p", "", "password for registry")
	cmd.Flags().StringVar(&platform, "platform", "linux/"+runtime.GOARCH, "platform for build")
	cmd.Flags().BoolVar(&bInsecure, "insecure", false, "insecure registry")

	return cmd
}

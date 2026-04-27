package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"mgkim.hpy.kr/graft/internal/diff_pull"
)

func NewCmdDiffPull() *cobra.Command {
	imageUrl := ""
	file := ""
	username := ""
	password := ""
	platform := ""
	baseTag := ""
	targetTag := ""
	bInsecure := false

	cmd := &cobra.Command{
		Use:   "diff-pull [image]",
		Short: "Pull partial layers that are different between base and target",
		Args:  cobra.MinimumNArgs(1), //cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				imageUrl = args[0]
				return diff_pull.DiffPull(diff_pull.DiffPullConfig{
					ImageUrl:  imageUrl,
					Username:  username,
					Password:  password,
					FilePath:  file,
					Platform:  platform,
					Insecure:  bInsecure,
					BaseTag:   baseTag,
					TargetTag: targetTag,
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
	cmd.Flags().StringVar(&baseTag, "base", "", "base tag")
	cmd.Flags().StringVar(&targetTag, "target", "latest", "target tag")

	return cmd
}

package cmd

import (
	"runtime"

	"github.com/spf13/cobra"
	"mgkim.hpy.kr/graft/internal/build"
)

// graft build -f ./Dockerfile -t myeongkr/build-test:latest

func NewCmdBuild() *cobra.Command {
	dockerfilePath := ""
	tag := ""
	bPush := false
	username := ""
	password := ""
	pullUsername := ""
	pullPassword := ""
	platform := ""
	bInsecure := false
	cache := ""

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a Dockerfile",
		Args:  cobra.MinimumNArgs(0), //cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			b := build.NewBuilder(build.BuilderConfig{
				DockerFilePath: dockerfilePath,
				RepositoryUrl:  tag,
				Push:           bPush,
				Username:       username,
				Password:       password,
				PullUsername:   pullUsername,
				PullPassword:   pullPassword,
				Platform:       platform,
				Insecure:       bInsecure,
				CacheDir:       cache,
			})
			return b.Build()
		},
	}

	cmd.Flags().StringVarP(&dockerfilePath, "file", "f", "Dockerfile", "Name of the Dockerfile (default: \"PATH/Dockerfile\")")
	cmd.Flags().StringVarP(&tag, "tag", "t", "my-image:latest", "Image identifier (format: \"[registry/]repository[:tag]\")")
	cmd.Flags().BoolVar(&bPush, "push", false, "push image to registry")
	cmd.Flags().BoolVar(&bInsecure, "insecure", false, "insecure registry")
	cmd.Flags().StringVarP(&username, "username", "u", "", "username for registry to push")
	cmd.Flags().StringVarP(&password, "password", "p", "", "password for registry to push")
	cmd.Flags().StringVarP(&pullUsername, "pull-username", "U", "", "username of registry to pull")
	cmd.Flags().StringVarP(&pullPassword, "pull-password", "P", "", "password of registry to pull")
	cmd.Flags().StringVar(&platform, "platform", "linux/"+runtime.GOARCH, "platform for build")
	cmd.Flags().StringVar(&cache, "cache", "", "cache directory for build")

	return cmd
}

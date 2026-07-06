package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
)

var rmAll bool

var rmCmd = &cobra.Command{
	Use:   "rm [name]",
	Short: "Remove a spin-managed container and its volume",
	Args: func(cmd *cobra.Command, args []string) error {
		if rmAll {
			if len(args) != 0 {
				return fmt.Errorf("name cannot be specified with --all")
			}
			return nil
		}
		if len(args) != 1 {
			return fmt.Errorf("name required")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cli, err := docker.NewClient(ctx)
		exitOnError(err)

		if rmAll {
			exitOnError(runOnAll(ctx, cli, docker.Rm))
			return
		}
		exitOnError(docker.Rm(ctx, cli, args[0]))
	},
}

func init() {
	rmCmd.Flags().BoolVar(&rmAll, "all", false, "remove all spin-managed containers and their volumes")
	rootCmd.AddCommand(rmCmd)
}

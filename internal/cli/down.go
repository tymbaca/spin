package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
)

var downAll bool

var downCmd = &cobra.Command{
	Use:   "down [name]",
	Short: "Stop a spin-managed container",
	Args: func(cmd *cobra.Command, args []string) error {
		if downAll {
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

		if downAll {
			exitOnError(runOnAll(ctx, cli, docker.Down))
			return
		}
		exitOnError(docker.Down(ctx, cli, args[0]))
	},
}

func init() {
	downCmd.Flags().BoolVar(&downAll, "all", false, "stop all spin-managed containers")
	rootCmd.AddCommand(downCmd)
}

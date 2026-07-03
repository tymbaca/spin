package cli

import (
	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
)

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Stop a spin-managed container",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cli, err := docker.NewClient(ctx)
		exitOnError(err)
		exitOnError(docker.Down(ctx, cli, args[0]))
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
}

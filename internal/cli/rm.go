package cli

import (
	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
)

var rmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a spin-managed container and its volume",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cli, err := docker.NewClient(ctx)
		exitOnError(err)
		exitOnError(docker.Rm(ctx, cli, args[0]))
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}

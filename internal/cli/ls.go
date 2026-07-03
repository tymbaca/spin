package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List spin-managed containers",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		cli, err := docker.NewClient(ctx)
		exitOnError(err)

		containers, err := docker.ListManaged(ctx, cli)
		exitOnError(err)

		if len(containers) == 0 {
			fmt.Println("no spin-managed containers")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSERVICE\tPORT\tSTATUS\tVOLUME")
		for _, c := range containers {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", c.Name, c.Service, c.Port, c.Status, c.Volume)
		}
		exitOnError(w.Flush())
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

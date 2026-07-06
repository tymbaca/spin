package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
	"github.com/tymbaca/spin/internal/kafka"
)

var lsOutput string

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List spin-managed containers",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		if lsOutput != "" && lsOutput != "wide" {
			exitOnError(fmt.Errorf("unsupported output %q; use --o wide", lsOutput))
		}

		cli, err := docker.NewClient(ctx)
		exitOnError(err)

		containers, err := docker.ListManaged(ctx, cli)
		exitOnError(err)

		if len(containers) == 0 {
			fmt.Println("no spin-managed containers")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if lsOutput == "wide" {
			fmt.Fprintln(w, "NAME\tSERVICE\tPORT\tSTATUS\tVOLUME\tCREDENTIALS")
		} else {
			fmt.Fprintln(w, "NAME\tSERVICE\tPORT\tSTATUS\tVOLUME")
		}
		for _, c := range containers {
			if lsOutput == "wide" {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n", c.Name, c.Service, c.Port, c.Status, c.Volume, credentialsFor(c))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", c.Name, c.Service, c.Port, c.Status, c.Volume)
			}
		}
		exitOnError(w.Flush())
	},
}

func credentialsFor(c docker.ContainerInfo) string {
	switch c.Service {
	case docker.ServicePostgres:
		return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", c.Port)
	case docker.ServiceKafka:
		return kafka.Credentials(c.Port)
	case docker.ServiceKafkaUI:
		return fmt.Sprintf("http://127.0.0.1:%d", c.Port)
	default:
		return ""
	}
}

func init() {
	lsCmd.Flags().StringVarP(&lsOutput, "out", "o", "", "output format (use \"wide\" to include credentials)")
	rootCmd.AddCommand(lsCmd)
}

package cli

import (
	"fmt"
	"net/url"
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
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n", c.Name, c.Service, c.Port, c.State, c.Volume, credentialsFor(c))
			} else {
				fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", c.Name, c.Service, c.Port, c.State, c.Volume)
			}
		}
		exitOnError(w.Flush())
	},
}

func credentialsFor(c docker.ContainerInfo) string {
	switch c.Service {
	case docker.ServicePostgres:
		user := firstNonEmpty(c.Credentials.User, docker.DefaultPostgresUser)
		password := firstNonEmpty(c.Credentials.Password, docker.DefaultPostgresPassword)
		database := firstNonEmpty(c.Credentials.Database, docker.DefaultPostgresDatabase)
		return fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable", user, password, c.Port, database)
	case docker.ServiceKafka:
		return kafka.CredentialsWithAuth(kafka.AuthConfig{
			Port:      c.Port,
			Protocol:  c.Credentials.Protocol,
			Mechanism: c.Credentials.Mechanism,
			User:      c.Credentials.User,
			Password:  c.Credentials.Password,
		})
	case docker.ServiceKafkaUI:
		return fmt.Sprintf("http://127.0.0.1:%d", c.Port)
	case docker.ServiceRedis:
		if c.Credentials.Password != "" {
			u := url.URL{
				Scheme: "redis",
				User:   url.UserPassword("", c.Credentials.Password),
				Host:   fmt.Sprintf("127.0.0.1:%d", c.Port),
			}
			return u.String()
		}
		return fmt.Sprintf("redis://127.0.0.1:%d", c.Port)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func init() {
	lsCmd.Flags().StringVarP(&lsOutput, "out", "o", "", "output format (use \"wide\" to include credentials)")
	rootCmd.AddCommand(lsCmd)
}

package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
	"github.com/tymbaca/spin/internal/migrate"
	"github.com/tymbaca/spin/internal/prompt"
)

var (
	upPort       int
	upMigrations string
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Create or start spin-managed containers",
}

var upPostgresCmd = &cobra.Command{
	Use:   "postgres <name>",
	Short: "Create or start a Postgres container",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		name := args[0]

		cli, err := docker.NewClient(ctx)
		exitOnError(err)

		exitOnError(resolvePortConflict(ctx, cli, name, upPort))

		result, err := docker.UpPostgres(ctx, cli, docker.PostgresUpOptions{
			Name: name,
			Port: upPort,
		})
		exitOnError(err)

		if upMigrations != "" {
			migrateCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			exitOnError(migrate.RunGoose(migrateCtx, result.Port, upMigrations))
		}
	},
}

func resolvePortConflict(ctx context.Context, cli *client.Client, name string, port int) error {
	conflict, err := docker.FindByHostPort(ctx, cli, port)
	if err != nil {
		return err
	}
	if conflict == nil || conflict.Name == name {
		return nil
	}

	msg := fmt.Sprintf(
		"port %d is used by spin container %q (%s). Spin it down and continue?",
		port,
		conflict.Name,
		conflict.Status,
	)
	ok, err := prompt.Confirm(msg)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("aborted")
	}

	return docker.Down(ctx, cli, conflict.Name)
}

func init() {
	upPostgresCmd.Flags().IntVar(&upPort, "port", 5432, "host port to bind Postgres on")
	upPostgresCmd.Flags().StringVar(&upMigrations, "migrations", "", "directory of goose SQL migrations to apply")

	upCmd.AddCommand(upPostgresCmd)
	rootCmd.AddCommand(upCmd)
}

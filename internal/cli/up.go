package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/tymbaca/spin/internal/docker"
	"github.com/tymbaca/spin/internal/kafka"
	"github.com/tymbaca/spin/internal/migrate"
	"github.com/tymbaca/spin/internal/prompt"
)

var (
	postgresPort       int
	postgresUser       string
	postgresPassword   string
	postgresDatabase   string
	postgresMigrations string
	kafkaPort          int
	kafkaProtocol      string
	kafkaMechanism     string
	kafkaUser          string
	kafkaPassword      string
	kafkaTopicList     string
	kafkaUIPort        int
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

		exitOnError(resolvePortConflict(ctx, cli, name, postgresPort))

		result, err := docker.UpPostgres(ctx, cli, docker.PostgresUpOptions{
			Name:     name,
			Port:     postgresPort,
			User:     postgresUser,
			Password: postgresPassword,
			Database: postgresDatabase,
		})
		exitOnError(err)

		if postgresMigrations != "" {
			migrateCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()
			exitOnError(migrate.RunGoose(migrateCtx, migrate.PostgresConfig{
				Port:     result.Port,
				User:     postgresUser,
				Password: postgresPassword,
				Database: postgresDatabase,
			}, postgresMigrations))
		}
	},
}

var upKafkaCmd = &cobra.Command{
	Use:   "kafka <name>",
	Short: "Create or start a Kafka container",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		name := args[0]

		cli, err := docker.NewClient(ctx)
		exitOnError(err)

		exitOnError(resolvePortConflict(ctx, cli, name, kafkaPort))

		result, err := docker.UpKafka(ctx, cli, docker.KafkaUpOptions{
			Name:      name,
			Port:      kafkaPort,
			Protocol:  kafkaProtocol,
			Mechanism: kafkaMechanism,
			User:      kafkaUser,
			Password:  kafkaPassword,
			WithUI:    kafkaUIPort != 0,
		})
		exitOnError(err)

		if kafkaTopicList != "" {
			topicCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			defer cancel()
			exitOnError(kafka.CreateTopicsFromFile(topicCtx, kafka.AuthConfig{
				Port:      result.Port,
				Protocol:  kafkaProtocol,
				Mechanism: kafkaMechanism,
				User:      kafkaUser,
				Password:  kafkaPassword,
			}, kafkaTopicList))
		}

		if kafkaUIPort != 0 {
			uiName := docker.KafkaUIName(name)
			exitOnError(resolvePortConflict(ctx, cli, uiName, kafkaUIPort))
			_, err := docker.UpKafkaUI(ctx, cli, docker.KafkaUIUpOptions{
				Name:      uiName,
				KafkaName: name,
				Port:      kafkaUIPort,
				Protocol:  kafkaProtocol,
				Mechanism: kafkaMechanism,
				User:      kafkaUser,
				Password:  kafkaPassword,
			})
			exitOnError(err)
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
	upPostgresCmd.Flags().IntVar(&postgresPort, "port", 5432, "host port to bind Postgres on")
	upPostgresCmd.Flags().StringVar(&postgresUser, "user", "postgres", "Postgres user")
	upPostgresCmd.Flags().StringVar(&postgresPassword, "password", "postgres", "Postgres password")
	upPostgresCmd.Flags().StringVar(&postgresDatabase, "database", "postgres", "Postgres database name")
	upPostgresCmd.Flags().StringVar(&postgresMigrations, "migrations", "", "directory of goose SQL migrations to apply")
	upKafkaCmd.Flags().IntVar(&kafkaPort, "port", 9092, "host port to bind Kafka on")
	upKafkaCmd.Flags().StringVar(&kafkaProtocol, "protocol", "SASL_PLAINTEXT", "Kafka listener security protocol")
	upKafkaCmd.Flags().StringVar(&kafkaMechanism, "mechanism", "PLAIN", "Kafka SASL mechanism")
	upKafkaCmd.Flags().StringVar(&kafkaUser, "user", "admin", "Kafka SASL username")
	upKafkaCmd.Flags().StringVar(&kafkaPassword, "password", "admin", "Kafka SASL password")
	upKafkaCmd.Flags().StringVar(&kafkaTopicList, "topic-list", "", "file containing Kafka topic names to create")
	upKafkaCmd.Flags().IntVar(&kafkaUIPort, "ui-port", 0, "host port to bind Kafka UI on (creates a separate spin container named <name>-ui)")

	upCmd.AddCommand(upPostgresCmd)
	upCmd.AddCommand(upKafkaCmd)
	rootCmd.AddCommand(upCmd)
}

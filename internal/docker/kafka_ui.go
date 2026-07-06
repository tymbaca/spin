package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type KafkaUIUpOptions struct {
	Name      string
	KafkaName string
	Port      int
	Protocol  string
	Mechanism string
	User      string
	Password  string
}

func (opts KafkaUIUpOptions) withDefaults() KafkaUIUpOptions {
	if opts.Protocol == "" {
		opts.Protocol = DefaultKafkaProtocol
	}
	if opts.Mechanism == "" {
		opts.Mechanism = DefaultKafkaMechanism
	}
	if opts.User == "" {
		opts.User = DefaultKafkaUser
	}
	if opts.Password == "" {
		opts.Password = DefaultKafkaPassword
	}
	return opts
}

func UpKafkaUI(ctx context.Context, cli *client.Client, opts KafkaUIUpOptions) (UpResult, error) {
	opts = opts.withDefaults()
	result := UpResult{Port: opts.Port}

	if opts.KafkaName != "" {
		kafkaContainer, err := FindByName(ctx, cli, opts.KafkaName)
		if err != nil {
			return result, err
		}
		if kafkaContainer == nil {
			return result, fmt.Errorf("kafka container %q not found; run spin up kafka %q first", opts.KafkaName, opts.KafkaName)
		}
		if kafkaContainer.Service != ServiceKafka {
			return result, fmt.Errorf("container %q is %s, not kafka", opts.KafkaName, kafkaContainer.Service)
		}
		if err := requireKafkaWithUI(ctx, cli, opts.KafkaName); err != nil {
			return result, err
		}
	}

	existing, err := FindByName(ctx, cli, opts.Name)
	if err != nil {
		return result, err
	}

	if existing != nil {
		if existing.Port != opts.Port {
			return result, fmt.Errorf("container %q already exists on port %d; use that port or run spin rm %q first", opts.Name, existing.Port, opts.Name)
		}

		state, err := ContainerState(ctx, cli, opts.Name)
		if err != nil {
			return result, err
		}

		if state == StateRunning {
			fmt.Printf("container %q is already running on http://127.0.0.1:%d\n", opts.Name, opts.Port)
			return result, nil
		}

		if err := cli.ContainerStart(ctx, ContainerName(opts.Name), container.StartOptions{}); err != nil {
			return result, fmt.Errorf("start container: %w", err)
		}
		fmt.Printf("started container %q on http://127.0.0.1:%d\n", opts.Name, opts.Port)
		result.Started = true
		return result, nil
	}

	conflict, err := FindByHostPort(ctx, cli, opts.Port)
	if err != nil {
		return result, err
	}
	if conflict != nil {
		return result, fmt.Errorf("port %d is used by spin container %q", opts.Port, conflict.Name)
	}

	var networkingConfig *network.NetworkingConfig
	if opts.KafkaName != "" {
		networkName := KafkaNetworkName(opts.KafkaName)
		if err := EnsureNetwork(ctx, cli, networkName); err != nil {
			return result, err
		}
		networkingConfig = networkConfig(networkName)
	}

	const imageRef = "provectuslabs/kafka-ui:v0.7.2"
	if err := EnsureImage(ctx, cli, imageRef); err != nil {
		return result, err
	}

	containerName := ContainerName(opts.Name)
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: imageRef,
			Env:   kafkaUIEnv(opts),
			Labels: map[string]string{
				LabelManaged: "true",
				LabelName:    opts.Name,
				LabelService: ServiceKafkaUI,
				LabelPort:    fmt.Sprintf("%d", opts.Port),
			},
		},
		&container.HostConfig{
			PortBindings: kafkaUIPortMap(opts.Port),
		},
		networkingConfig,
		nil,
		containerName,
	)
	if err != nil {
		return result, fmt.Errorf("create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return result, fmt.Errorf("start container: %w", err)
	}

	fmt.Printf("created and started container %q on http://127.0.0.1:%d\n", opts.Name, opts.Port)
	result.Started = true
	return result, nil
}

func kafkaUIEnv(opts KafkaUIUpOptions) []string {
	env := []string{"DYNAMIC_CONFIG_ENABLED=true"}
	if opts.KafkaName == "" {
		return env
	}

	env = append(env,
		"KAFKA_CLUSTERS_0_NAME=local",
		fmt.Sprintf("KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=%s", KafkaBootstrapForUI(opts.KafkaName)),
	)

	if opts.Protocol != "PLAINTEXT" {
		env = append(env, fmt.Sprintf("KAFKA_CLUSTERS_0_PROPERTIES_SECURITY_PROTOCOL=%s", opts.Protocol))
	}

	if kafkaUsesSASL(opts.Protocol) && opts.Mechanism == "PLAIN" {
		env = append(env,
			fmt.Sprintf("KAFKA_CLUSTERS_0_PROPERTIES_SASL_MECHANISM=%s", opts.Mechanism),
			fmt.Sprintf(
				`KAFKA_CLUSTERS_0_PROPERTIES_SASL_JAAS_CONFIG=org.apache.kafka.common.security.plain.PlainLoginModule required username="%s" password="%s";`,
				opts.User, opts.Password,
			),
		)
	}

	return env
}

func kafkaUIPortMap(port int) nat.PortMap {
	return nat.PortMap{
		nat.Port("8080/tcp"): []nat.PortBinding{{
			HostIP:   "127.0.0.1",
			HostPort: fmt.Sprintf("%d", port),
		}},
	}
}

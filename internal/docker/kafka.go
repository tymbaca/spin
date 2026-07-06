package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	DefaultKafkaProtocol  = "SASL_PLAINTEXT"
	DefaultKafkaMechanism = "PLAIN"
	DefaultKafkaUser      = "admin"
	DefaultKafkaPassword  = "admin"

	kafkaHostListenerPort   = 9092
	kafkaDockerListenerPort = 9094
)

type KafkaUpOptions struct {
	Name     string
	Port     int
	Protocol string
	Mechanism string
	User     string
	Password string
	WithUI   bool
}

func (opts KafkaUpOptions) withDefaults() KafkaUpOptions {
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

func UpKafka(ctx context.Context, cli *client.Client, opts KafkaUpOptions) (UpResult, error) {
	opts = opts.withDefaults()
	result := UpResult{Port: opts.Port}

	existing, err := FindByName(ctx, cli, opts.Name)
	if err != nil {
		return result, err
	}

	if existing != nil {
		if opts.WithUI {
			if err := requireKafkaWithUI(ctx, cli, opts.Name); err != nil {
				return result, err
			}
		}

		if existing.Port != opts.Port {
			return result, fmt.Errorf("container %q already exists on port %d; use that port or run spin rm %q first", opts.Name, existing.Port, opts.Name)
		}

		state, err := ContainerState(ctx, cli, opts.Name)
		if err != nil {
			return result, err
		}

		if state == "running" {
			fmt.Printf("container %q is already running on 127.0.0.1:%d\n", opts.Name, opts.Port)
			return result, nil
		}

		if err := cli.ContainerStart(ctx, ContainerName(opts.Name), container.StartOptions{}); err != nil {
			return result, fmt.Errorf("start container: %w", err)
		}
		fmt.Printf("started container %q on 127.0.0.1:%d\n", opts.Name, opts.Port)
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

	volumeName := VolumeName(opts.Name)
	if err := ensureVolume(ctx, cli, volumeName, opts.Name, ServiceKafka); err != nil {
		return result, err
	}

	const imageRef = "apache/kafka:3.9.0"
	if err := EnsureImage(ctx, cli, imageRef); err != nil {
		return result, err
	}

	labels := map[string]string{
		LabelManaged: "true",
		LabelName:    opts.Name,
		LabelService: ServiceKafka,
		LabelPort:    fmt.Sprintf("%d", opts.Port),
	}
	if opts.WithUI {
		labels[LabelKafkaWithUI] = "true"
	}

	var networkingConfig *network.NetworkingConfig
	if opts.WithUI {
		networkName := KafkaNetworkName(opts.Name)
		if err := EnsureNetwork(ctx, cli, networkName); err != nil {
			return result, err
		}
		networkingConfig = networkConfig(networkName)
	}

	containerName := ContainerName(opts.Name)
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:      imageRef,
			Env:        kafkaEnv(opts),
			Entrypoint: []string{"/bin/bash", "-lc"},
			Cmd:        []string{kafkaStartScript(opts)},
			Labels:     labels,
		},
		&container.HostConfig{
			PortBindings: kafkaPortMap(opts.Port),
			Binds:        []string{volumeName + ":/var/lib/kafka/data"},
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

	fmt.Printf("created and started container %q on 127.0.0.1:%d\n", opts.Name, opts.Port)
	result.Started = true
	return result, nil
}

func kafkaEnv(opts KafkaUpOptions) []string {
	if opts.WithUI {
		return kafkaEnvWithUI(opts)
	}
	return kafkaEnvSingle(opts)
}

func kafkaEnvSingle(opts KafkaUpOptions) []string {
	env := []string{
		"KAFKA_NODE_ID=1",
		"KAFKA_PROCESS_ROLES=broker,controller",
		"KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093",
		fmt.Sprintf("KAFKA_LISTENERS=%s://0.0.0.0:%d,CONTROLLER://:9093", opts.Protocol, kafkaHostListenerPort),
		fmt.Sprintf("KAFKA_ADVERTISED_LISTENERS=%s://127.0.0.1:%d", opts.Protocol, opts.Port),
		fmt.Sprintf("KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=%s:%s,CONTROLLER:PLAINTEXT", opts.Protocol, opts.Protocol),
		"KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
		fmt.Sprintf("KAFKA_INTER_BROKER_LISTENER_NAME=%s", opts.Protocol),
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1",
		"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0",
		"KAFKA_AUTO_CREATE_TOPICS_ENABLE=false",
		"KAFKA_LOG_DIRS=/var/lib/kafka/data",
	}
	return append(env, kafkaSASLEnv(opts, opts.Protocol)...)
}

func kafkaEnvWithUI(opts KafkaUpOptions) []string {
	brokerName := ContainerName(opts.Name)
	env := []string{
		"KAFKA_NODE_ID=1",
		"KAFKA_PROCESS_ROLES=broker,controller",
		"KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093",
		fmt.Sprintf(
			"KAFKA_LISTENERS=HOST://0.0.0.0:%d,DOCKER://0.0.0.0:%d,CONTROLLER://:9093",
			kafkaHostListenerPort, kafkaDockerListenerPort,
		),
		fmt.Sprintf(
			"KAFKA_ADVERTISED_LISTENERS=HOST://127.0.0.1:%d,DOCKER://%s:%d",
			opts.Port, brokerName, kafkaDockerListenerPort,
		),
		fmt.Sprintf(
			"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=HOST:%s,DOCKER:%s,CONTROLLER:PLAINTEXT",
			opts.Protocol, opts.Protocol,
		),
		"KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
		"KAFKA_INTER_BROKER_LISTENER_NAME=DOCKER",
		"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1",
		"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1",
		"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0",
		"KAFKA_AUTO_CREATE_TOPICS_ENABLE=false",
		"KAFKA_LOG_DIRS=/var/lib/kafka/data",
	}
	env = append(env, kafkaSASLEnv(opts, "HOST", "DOCKER")...)
	return env
}

func kafkaSASLEnv(opts KafkaUpOptions, listenerNames ...string) []string {
	if !kafkaUsesSASL(opts.Protocol) {
		return nil
	}

	if len(listenerNames) == 0 {
		listenerNames = []string{opts.Protocol}
	}

	env := []string{
		fmt.Sprintf("KAFKA_SASL_ENABLED_MECHANISMS=%s", opts.Mechanism),
		fmt.Sprintf("KAFKA_SASL_MECHANISM_INTER_BROKER_PROTOCOL=%s", opts.Mechanism),
		"KAFKA_OPTS=-Djava.security.auth.login.config=/tmp/kafka_server_jaas.conf",
	}
	for _, listener := range listenerNames {
		env = append(env, fmt.Sprintf(
			"KAFKA_LISTENER_NAME_%s_%s_SASL_JAAS_CONFIG=org.apache.kafka.common.security.plain.PlainLoginModule required username=%q password=%q user_%s=%q;",
			listener, opts.Mechanism, opts.User, opts.Password, opts.User, opts.Password,
		))
	}
	return env
}

func kafkaUsesSASL(protocol string) bool {
	return protocol != "PLAINTEXT" && protocol != "SSL"
}

func kafkaStartScript(opts KafkaUpOptions) string {
	if !kafkaUsesSASL(opts.Protocol) {
		return "exec /etc/kafka/docker/run"
	}

	return fmt.Sprintf(`cat >/tmp/kafka_server_jaas.conf <<'EOF'
KafkaServer {
  org.apache.kafka.common.security.plain.PlainLoginModule required
  username=%q
  password=%q
  user_%s=%q;
};
EOF
exec /etc/kafka/docker/run`, opts.User, opts.Password, opts.User, opts.Password)
}

func kafkaPortMap(port int) nat.PortMap {
	return nat.PortMap{
		nat.Port(fmt.Sprintf("%d/tcp", kafkaHostListenerPort)): []nat.PortBinding{{
			HostIP:   "127.0.0.1",
			HostPort: fmt.Sprintf("%d", port),
		}},
	}
}

func requireKafkaWithUI(ctx context.Context, cli *client.Client, name string) error {
	inspect, err := cli.ContainerInspect(ctx, ContainerName(name))
	if err != nil {
		return fmt.Errorf("inspect container: %w", err)
	}
	if inspect.Config.Labels[LabelKafkaWithUI] == "true" {
		return nil
	}
	return fmt.Errorf(
		"container %q was created without kafka-ui support; run spin rm %q and create again with --ui-port",
		name, name,
	)
}

func KafkaBootstrapForUI(kafkaName string) string {
	return fmt.Sprintf("%s:%d", ContainerName(kafkaName), kafkaDockerListenerPort)
}

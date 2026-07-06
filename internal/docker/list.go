package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type CredentialInfo struct {
	User      string
	Password  string
	Database  string
	Protocol  string
	Mechanism string
}

type State string

const (
	StateRunning State = "running"
	StateExited  State = "exited"
)

type ContainerInfo struct {
	ID          string
	Name        string
	Service     string
	Port        int
	Status      State
	Volume      string
	Credentials CredentialInfo
}

func ListManaged(ctx context.Context, cli *client.Client) ([]ContainerInfo, error) {
	args := filters.NewArgs(filters.Arg("label", LabelManaged+"=true"))
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		inspect, err := cli.ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, fmt.Errorf("inspect container %s: %w", c.ID[:12], err)
		}
		var env []string
		if inspect.Config != nil {
			env = inspect.Config.Env
		}

		info, err := containerInfoFromSummary(c, env)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

func containerInfoFromSummary(c container.Summary, env []string) (ContainerInfo, error) {
	spinName := c.Labels[LabelName]
	if spinName == "" {
		return ContainerInfo{}, fmt.Errorf("container %s missing %s label", c.ID[:12], LabelName)
	}

	port, err := strconv.Atoi(c.Labels[LabelPort])
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("container %s has invalid %s label: %w", spinName, LabelPort, err)
	}

	service := c.Labels[LabelService]
	return ContainerInfo{
		ID:          c.ID[:12],
		Name:        spinName,
		Service:     service,
		Port:        port,
		Status:      State(c.State),
		Volume:      VolumeName(spinName),
		Credentials: credentialsFromContainer(service, c.Labels, env),
	}, nil
}

func FindByName(ctx context.Context, cli *client.Client, name string) (*ContainerInfo, error) {
	args := filters.NewArgs(
		filters.Arg("label", LabelManaged+"=true"),
		filters.Arg("label", LabelName+"="+name),
	)
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return nil, fmt.Errorf("find container: %w", err)
	}
	if len(containers) == 0 {
		return nil, nil
	}

	info, err := containerInfoFromSummary(containers[0], nil)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func credentialsFromContainer(service string, labels map[string]string, env []string) CredentialInfo {
	envs := envMap(env)
	credentials := credentialsFromLabels(service, labels)

	switch service {
	case ServicePostgres:
		credentials.User = firstNonEmpty(credentials.User, envs["POSTGRES_USER"])
		credentials.Password = firstNonEmpty(credentials.Password, envs["POSTGRES_PASSWORD"])
		credentials.Database = firstNonEmpty(credentials.Database, envs["POSTGRES_DB"])
	case ServiceKafka:
		credentials.Protocol = firstNonEmpty(credentials.Protocol, kafkaProtocolFromEnv(envs))
		credentials.Mechanism = firstNonEmpty(credentials.Mechanism, envs["KAFKA_SASL_ENABLED_MECHANISMS"])
		if credentials.User == "" || credentials.Password == "" {
			user, password := kafkaSASLCredentialsFromEnv(envs)
			credentials.User = firstNonEmpty(credentials.User, user)
			credentials.Password = firstNonEmpty(credentials.Password, password)
		}
	}

	return credentials
}

func credentialsFromLabels(service string, labels map[string]string) CredentialInfo {
	switch service {
	case ServicePostgres:
		return CredentialInfo{
			User:     labels[LabelCredentialPostgresUser],
			Password: labels[LabelCredentialPostgresPassword],
			Database: labels[LabelCredentialPostgresDatabase],
		}
	case ServiceKafka:
		return CredentialInfo{
			User:      labels[LabelCredentialKafkaUser],
			Password:  labels[LabelCredentialKafkaPassword],
			Protocol:  labels[LabelCredentialKafkaProtocol],
			Mechanism: labels[LabelCredentialKafkaMechanism],
		}
	default:
		return CredentialInfo{}
	}
}

func envMap(env []string) map[string]string {
	result := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

func kafkaProtocolFromEnv(env map[string]string) string {
	if listeners := env["KAFKA_ADVERTISED_LISTENERS"]; listeners != "" {
		if protocol, _, ok := strings.Cut(listeners, "://"); ok && protocol != "HOST" && protocol != "DOCKER" {
			return protocol
		}
	}

	for _, mapping := range strings.Split(env["KAFKA_LISTENER_SECURITY_PROTOCOL_MAP"], ",") {
		listener, protocol, ok := strings.Cut(mapping, ":")
		if ok && listener != "CONTROLLER" {
			return protocol
		}
	}
	return ""
}

func kafkaSASLCredentialsFromEnv(env map[string]string) (string, string) {
	for key, value := range env {
		if strings.HasPrefix(key, "KAFKA_LISTENER_NAME_") && strings.HasSuffix(key, "_SASL_JAAS_CONFIG") {
			return jaasValue(value, "username"), jaasValue(value, "password")
		}
	}
	return "", ""
}

func jaasValue(config, key string) string {
	_, value, ok := strings.Cut(config, key+"=")
	if !ok {
		return ""
	}

	value = strings.TrimLeft(value, " \t")
	if value == "" {
		return ""
	}

	if value[0] == '"' {
		end := 1
		escaped := false
		for end < len(value) {
			switch {
			case escaped:
				escaped = false
			case value[end] == '\\':
				escaped = true
			case value[end] == '"':
				unquoted, err := strconv.Unquote(value[:end+1])
				if err == nil {
					return unquoted
				}
				return strings.Trim(value[1:end], `"`)
			}
			end++
		}
		return ""
	}

	end := strings.IndexAny(value, " \t;")
	if end == -1 {
		return value
	}
	return value[:end]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func FindByHostPort(ctx context.Context, cli *client.Client, port int) (*ContainerInfo, error) {
	managed, err := ListManaged(ctx, cli)
	if err != nil {
		return nil, err
	}
	for _, c := range managed {
		if c.Port == port && c.Status == StateRunning {
			return &c, nil
		}
	}
	return nil, nil
}

func ContainerID(ctx context.Context, cli *client.Client, name string) (string, error) {
	containerName := ContainerName(name)
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("inspect container: %w", err)
	}
	return inspect.ID, nil
}

func ContainerState(ctx context.Context, cli *client.Client, name string) (State, error) {
	containerName := ContainerName(name)
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("inspect container: %w", err)
	}
	return State(inspect.State.Status), nil
}

func Rm(ctx context.Context, cli *client.Client, name string) error {
	containerName := ContainerName(name)
	volumeName := VolumeName(name)

	_ = cli.ContainerStop(ctx, containerName, container.StopOptions{})

	if err := cli.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true}); err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("container %q not found", name)
		}
		return fmt.Errorf("remove container: %w", err)
	}

	if err := cli.VolumeRemove(ctx, volumeName, true); err != nil {
		if !client.IsErrNotFound(err) {
			return fmt.Errorf("remove volume: %w", err)
		}
	}

	fmt.Printf("removed container %q and volume %q\n", name, volumeName)
	return nil
}

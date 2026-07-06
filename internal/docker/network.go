package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func KafkaNetworkName(kafkaName string) string {
	return "spin-" + kafkaName + "-net"
}

func EnsureNetwork(ctx context.Context, cli *client.Client, name string) error {
	_, err := cli.NetworkInspect(ctx, name, network.InspectOptions{})
	if err == nil {
		return nil
	}
	if !client.IsErrNotFound(err) {
		return fmt.Errorf("inspect network %q: %w", name, err)
	}

	_, err = cli.NetworkCreate(ctx, name, network.CreateOptions{
		Labels: map[string]string{
			LabelManaged: "true",
		},
	})
	if err != nil {
		return fmt.Errorf("create network %q: %w", name, err)
	}
	return nil
}

func networkConfig(networkName string) *network.NetworkingConfig {
	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			networkName: {},
		},
	}
}

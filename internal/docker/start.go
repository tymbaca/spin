package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func Start(ctx context.Context, cli *client.Client, name string) error {
	containerName := ContainerName(name)
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("container %q not found", name)
		}
		return fmt.Errorf("inspect container: %w", err)
	}

	if inspect.State.Running {
		fmt.Printf("container %q is already running\n", name)
		return nil
	}

	if err := cli.ContainerStart(ctx, containerName, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	fmt.Printf("started container %q\n", name)
	return nil
}

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func Down(ctx context.Context, cli *client.Client, name string) error {
	containerName := ContainerName(name)
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return fmt.Errorf("container %q not found", name)
		}
		return fmt.Errorf("inspect container: %w", err)
	}

	if !inspect.State.Running {
		fmt.Printf("container %q is already stopped\n", name)
		return nil
	}

	if err := cli.ContainerStop(ctx, containerName, container.StopOptions{}); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}

	fmt.Printf("stopped container %q\n", name)
	return nil
}

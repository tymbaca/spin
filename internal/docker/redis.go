package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type RedisUpOptions struct {
	Name string
	Port int
}

func UpRedis(ctx context.Context, cli *client.Client, opts RedisUpOptions) (UpResult, error) {
	result := UpResult{Port: opts.Port}

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
	if err := ensureVolume(ctx, cli, volumeName, opts.Name, ServiceRedis); err != nil {
		return result, err
	}

	const imageRef = "redis:7"
	if err := EnsureImage(ctx, cli, imageRef); err != nil {
		return result, err
	}

	containerName := ContainerName(opts.Name)
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: imageRef,
			Labels: map[string]string{
				LabelManaged: "true",
				LabelName:    opts.Name,
				LabelService: ServiceRedis,
				LabelPort:    fmt.Sprintf("%d", opts.Port),
			},
		},
		&container.HostConfig{
			PortBindings: redisPortMap(opts.Port),
			Binds:        []string{volumeName + ":/data"},
		},
		nil,
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

func redisPortMap(port int) nat.PortMap {
	return nat.PortMap{
		nat.Port("6379/tcp"): []nat.PortBinding{{
			HostIP:   "127.0.0.1",
			HostPort: fmt.Sprintf("%d", port),
		}},
	}
}

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type PostgresUpOptions struct {
	Name string
	Port int
}

type UpResult struct {
	Port    int
	Started bool
}

func UpPostgres(ctx context.Context, cli *client.Client, opts PostgresUpOptions) (UpResult, error) {
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
	if err := ensureVolume(ctx, cli, volumeName, opts.Name); err != nil {
		return result, err
	}

	const imageRef = "postgres:16"
	if err := EnsureImage(ctx, cli, imageRef); err != nil {
		return result, err
	}

	containerName := ContainerName(opts.Name)
	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image: imageRef,
			Env: []string{
				"POSTGRES_USER=postgres",
				"POSTGRES_PASSWORD=postgres",
				"POSTGRES_DB=postgres",
			},
			Labels: map[string]string{
				LabelManaged: "true",
				LabelName:    opts.Name,
				LabelService: ServicePostgres,
				LabelPort:    fmt.Sprintf("%d", opts.Port),
			},
		},
		&container.HostConfig{
			PortBindings: natPortMap(opts.Port),
			Binds:        []string{volumeName + ":/var/lib/postgresql/data"},
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

func ensureVolume(ctx context.Context, cli *client.Client, volumeName, spinName string) error {
	_, err := cli.VolumeInspect(ctx, volumeName)
	if err == nil {
		return nil
	}
	if !client.IsErrNotFound(err) {
		return fmt.Errorf("inspect volume: %w", err)
	}

	_, err = cli.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
		Labels: map[string]string{
			LabelManaged: "true",
			LabelName:    spinName,
			LabelService: ServicePostgres,
		},
	})
	if err != nil {
		return fmt.Errorf("create volume: %w", err)
	}
	return nil
}

func natPortMap(port int) nat.PortMap {
	return nat.PortMap{
		nat.Port("5432/tcp"): []nat.PortBinding{{
			HostIP:   "127.0.0.1",
			HostPort: fmt.Sprintf("%d", port),
		}},
	}
}

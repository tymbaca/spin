package docker

import (
	"context"
	"fmt"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type ContainerInfo struct {
	ID      string
	Name    string
	Service string
	Port    int
	Status  string
	Volume  string
}

func ListManaged(ctx context.Context, cli *client.Client) ([]ContainerInfo, error) {
	args := filters.NewArgs(filters.Arg("label", LabelManaged+"=true"))
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		info, err := containerInfoFromSummary(c)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

func containerInfoFromSummary(c container.Summary) (ContainerInfo, error) {
	spinName := c.Labels[LabelName]
	if spinName == "" {
		return ContainerInfo{}, fmt.Errorf("container %s missing %s label", c.ID[:12], LabelName)
	}

	port, err := strconv.Atoi(c.Labels[LabelPort])
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("container %s has invalid %s label: %w", spinName, LabelPort, err)
	}

	return ContainerInfo{
		ID:      c.ID[:12],
		Name:    spinName,
		Service: c.Labels[LabelService],
		Port:    port,
		Status:  c.State,
		Volume:  VolumeName(spinName),
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

	info, err := containerInfoFromSummary(containers[0])
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func FindByHostPort(ctx context.Context, cli *client.Client, port int) (*ContainerInfo, error) {
	managed, err := ListManaged(ctx, cli)
	if err != nil {
		return nil, err
	}
	for _, c := range managed {
		if c.Port == port && c.Status == "running" {
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

func ContainerState(ctx context.Context, cli *client.Client, name string) (string, error) {
	containerName := ContainerName(name)
	inspect, err := cli.ContainerInspect(ctx, containerName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("inspect container: %w", err)
	}
	return inspect.State.Status, nil
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

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
)

func NewClient(ctx context.Context) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connect to docker: %w", err)
	}
	if _, err := cli.Ping(ctx); err != nil {
		return nil, fmt.Errorf("docker is not running: %w", err)
	}
	return cli, nil
}

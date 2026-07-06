package cli

import (
	"context"
	"fmt"

	"github.com/docker/docker/client"
	"github.com/tymbaca/spin/internal/docker"
)

func runOnAll(ctx context.Context, cli *client.Client, action func(context.Context, *client.Client, string) error) error {
	containers, err := docker.ListManaged(ctx, cli)
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		fmt.Println("no spin-managed containers")
		return nil
	}
	for _, c := range containers {
		if err := action(ctx, cli, c.Name); err != nil {
			return err
		}
	}
	return nil
}

package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func EnsureImage(ctx context.Context, cli *client.Client, ref string) error {
	_, err := cli.ImageInspect(ctx, ref)
	if err == nil {
		return nil
	}
	if !client.IsErrNotFound(err) {
		return fmt.Errorf("inspect image %q: %w", ref, err)
	}

	fmt.Printf("pulling image %s...\n", ref)
	reader, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

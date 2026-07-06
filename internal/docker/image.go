package docker

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func EnsureImage(ctx context.Context, cli *client.Client, ref string) error {
	_, err := cli.ImageInspect(ctx, ref)
	if err == nil {
		return nil
	}
	if !client.IsErrNotFound(err) {
		return fmt.Errorf("inspect image %q: %w", ref, err)
	}

	reader, err := cli.ImagePull(ctx, ref, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	defer reader.Close()

	fd, isTTY := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(reader, os.Stdout, fd, isTTY, nil); err != nil {
		return fmt.Errorf("pull image %q: %w", ref, err)
	}
	return nil
}

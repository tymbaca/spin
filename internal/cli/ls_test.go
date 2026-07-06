package cli

import (
	"testing"

	"github.com/tymbaca/spin/internal/docker"
)

func TestCredentialsForRedis(t *testing.T) {
	got := credentialsFor(docker.ContainerInfo{
		Service: docker.ServiceRedis,
		Port:    6379,
	})

	const want = "redis://127.0.0.1:6379"
	if got != want {
		t.Fatalf("credentialsFor(redis) = %q, want %q", got, want)
	}
}
